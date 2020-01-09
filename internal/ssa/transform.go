package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type ssaTransformer struct {
	f                       *ircode.Function
	log                     *errlog.ErrorLog
	namedGroupVariables     map[string]*GroupVariable
	scopedGroupVariables    map[*ircode.CommandScope]*GroupVariable
	parameterGroupVariables map[*types.Group]*GroupVariable
	scopes                  []*ssaScope
	topLevelScope           *ssaScope
}

func (s *ssaTransformer) transformBlock(c *ircode.Command, vs *ssaScope) bool {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for i, c2 := range c.Block {
		if !s.transformCommand(c2, vs) {
			if i+1 < len(c.Block) && c.Block[i+1].Op != ircode.OpCloseScope {
				s.log.AddError(errlog.ErrorUnreachable, c.Block[i+1].Location)
			}
			return false
		}
	}
	return true
}

func (s *ssaTransformer) transformPreBlock(c *ircode.Command, vs *ssaScope) bool {
	for i, c2 := range c.PreBlock {
		if !s.transformCommand(c2, vs) {
			if i+1 < len(c.Block) && c.Block[i+1].Op != ircode.OpCloseScope {
				s.log.AddError(errlog.ErrorUnreachable, c.Block[i+1].Location)
			}
			return false
		}
	}
	return true
}

func (s *ssaTransformer) transformCommand(c *ircode.Command, vs *ssaScope) bool {
	if c.PreBlock != nil {
		s.transformPreBlock(c, vs)
	}
	switch c.Op {
	case ircode.OpBlock:
		s.transformBlock(c, vs)
	case ircode.OpIf:
		c.Scope.GroupInfo = vs.newScopedGroupVariable(c.Scope)
		// visit the condition
		s.transformArguments(c, vs)
		// visit the if-clause
		ifScope := newScope(s, c)
		ifScope.parent = vs
		ifScope.kind = scopeIf
		ifCompletes := s.transformBlock(c, ifScope)
		s.polishScope(c, ifScope)
		// visit the else-clause
		if c.Else != nil {
			c.Else.Scope.GroupInfo = vs.newScopedGroupVariable(c.Else.Scope)
			elseScope := newScope(s, c.Else)
			elseScope.parent = vs
			elseScope.kind = scopeIf
			elseCompletes := s.transformBlock(c.Else, elseScope)
			s.polishScope(c.Else, elseScope)
			if ifCompletes && elseCompletes {
				// Control flow flows through the if-clause or else-clause and continues afterwards
				vs.mergeVariablesOnIfElse(ifScope, elseScope)
			} else if ifCompletes {
				// Control flow can either continue through the if-clause, or it does not reach past the end of the else-clause
				vs.mergeVariables(ifScope)
			} else if elseCompletes {
				// Control flow can either continue through the else-clause, or it does not reach past the end of the if-clause
				vs.mergeVariables(elseScope)
			}
			return ifCompletes || elseCompletes
		} else if ifCompletes {
			// No else, but control flow continues after the if
			vs.mergeVariablesOnIf(ifScope)
			return true
		}
		// No else, and control flow does not come past the if.
		return true
	case ircode.OpLoop:
		c.Scope.GroupInfo = vs.newScopedGroupVariable(c.Scope)
		loopScope := newScope(s, c)
		loopScope.parent = vs
		loopScope.kind = scopeLoop
		doesLoop := s.transformBlock(c, loopScope)
		if doesLoop {
			closeScope := c.Block[len(c.Block)-1]
			if closeScope.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			// The loop can run more than once
			loopScope.mergeVariablesOnContinue(closeScope, loopScope, s.log)
		}
		loopScope.mergeVariablesOnBreaks()
		s.polishScope(c, loopScope)
		// How many breaks are breaking exactly at this loop?
		// Breaks targeting an outer loop are not considered.
		return loopScope.breakCount > 0
	case ircode.OpBreak:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		loopScope := vs
		for loopDepth > 0 {
			if loopScope.kind == scopeLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
			loopScope = loopScope.parent
			if s == nil {
				panic("Ooooops")
			}
		}
		loopScope.mergeVariablesOnBreak(vs)
		loopScope.breakCount++
		return false
	case ircode.OpContinue:
		// Find the loop-scope
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		loopScope := vs
		for loopDepth > 0 {
			if loopScope.kind == scopeLoop {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
			loopScope = loopScope.parent
			if s == nil {
				panic("Ooooops")
			}
		}
		loopScope.mergeVariablesOnContinue(c, vs, s.log)
		loopScope.continueCount++
		return false
	case ircode.OpDefVariable:
		if types.TypeHasPointers(c.Dest[0].Type.Type) {
			if c.Dest[0].Kind == ircode.VarParameter && c.Dest[0].Type.PointerDestGroup != nil {
				// Parameters with pointers have groups already when the function is being called.
				grp := c.Dest[0].Type.PointerDestGroup
				if grp.Kind == types.GroupIsolate {
					setGroupVariable(c.Dest[0], vs.newGroupVariable())
				} else {
					setGroupVariable(c.Dest[0], vs.newNamedGroupVariable(grp.Name))
				}
			} else {
				setGroupVariable(c.Dest[0], vs.newGroupVariable())
			}
		}
	case ircode.OpPrintln:
		s.transformArguments(c, vs)
	case ircode.OpSet:
		s.transformArguments(c, vs)
		gDest := s.accessChainGroupVariable(c, vs)
		gSrc := argumentGroupVariable(c, c.Args[len(c.Args)-1], vs, c.Location)
		if types.TypeHasPointers(c.Args[len(c.Args)-1].Type().Type) /*|| accessChainHasPointers(c)*/ {
			gv := s.generateMerge(c, gDest, gSrc, vs)
			outType := c.AccessChain[len(c.AccessChain)-1].OutputType
			if outType.PointerDestGroup != nil && outType.PointerDestGroup.Kind == types.GroupIsolate {
				gv.makeUnavailable()
			}
		}
		if len(c.Dest) == 1 {
			_, dest := vs.lookupVariable(c.Dest[0])
			if dest == nil {
				panic("Oooops, variable does not exist")
			}
			if !ircode.IsVarInitialized(dest) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, dest.Original.Name)
			}
			v := vs.newVariableVersion(dest)
			c.Dest[0] = v
		}
	case ircode.OpGet:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		if types.TypeHasPointers(v.Type.Type) /* || accessChainHasPointers(c)*/ {
			// The group resulting in the Get operation becomes the group of the destination
			setGroupVariable(v, s.accessChainGroupVariable(c, vs))
		}
		// The destination variable is now initialized
		v.IsInitialized = true
	case ircode.OpSetVariable:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// If assigning a constant, change the variable's ExprType, because it carries the constant value.
		// This way the compiler knows that the current value of this variable is a constant.
		if c.Args[0].Const != nil {
			v.Type = c.Args[0].Const.ExprType
		}
		// If the type has pointers, update the group variable
		if types.TypeHasPointers(v.Type.Type) {
			// The group of the argument becomes the group of the destination
			setGroupVariable(v, argumentGroupVariable(c, c.Args[0], vs, c.Location))
		}
	case ircode.OpAdd,
		ircode.OpSub,
		ircode.OpMul,
		ircode.OpDiv,
		ircode.OpRemainder,
		ircode.OpBinaryXor,
		ircode.OpBinaryOr,
		ircode.OpBinaryAnd,
		ircode.OpShiftLeft,
		ircode.OpShiftRight,
		ircode.OpBitClear,
		ircode.OpLogicalOr,
		ircode.OpLogicalAnd,
		ircode.OpEqual,
		ircode.OpNotEqual,
		ircode.OpLess,
		ircode.OpGreater,
		ircode.OpLessOrEqual,
		ircode.OpGreaterOrEqual,
		ircode.OpNot,
		ircode.OpMinusSign,
		ircode.OpBitwiseComplement,
		ircode.OpSizeOf,
		ircode.OpLen,
		ircode.OpCap:

		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// No groups to update here, because these ops work on primitive types.
	case ircode.OpAssert:
		s.transformArguments(c, vs)
	case ircode.OpArray,
		ircode.OpStruct:

		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		// Determine whether data is allocated on the heap
		allocMemory := false
		t := v.Type.Type
		if pt, ok := types.GetPointerType(t); ok {
			t = pt.ElementType
			allocMemory = true
		} else if sl, ok := types.GetSliceType(t); ok {
			allocMemory = true
			t = sl.ElementType
		}
		// If the type has pointers, update the group variable
		if types.TypeHasPointers(t) {
			var gv *GroupVariable
			for i := range c.Args {
				gArg := argumentGroupVariable(c, c.Args[i], vs, c.Location)
				if gArg != nil {
					if gv == nil {
						gv = gArg
					} else {
						gv = s.generateMerge(c, gv, gArg, vs)
					}
				}
			}
			if gv == nil {
				gv = vs.newGroupVariable()
			}
			setGroupVariable(v, gv)
			if allocMemory {
				gv.Allocations++
			}
		} else if allocMemory {
			gv := vs.newGroupVariable()
			setGroupVariable(v, gv)
			gv.Allocations++
		}
	case ircode.OpAppend:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		gv := argumentGroupVariable(c, c.Args[0], vs, c.Location)
		sl, ok := types.GetSliceType(v.Type.Type)
		if !ok {
			panic("Ooooops")
		}
		if types.TypeHasPointers(sl.ElementType) {
			// Ignore the first two arguments which are the slice and the amount of elements to add
			for _, arg := range c.Args[2:] {
				gArg := argumentGroupVariable(c, arg, vs, c.Location)
				gv = s.generateMerge(c, gv, gArg, vs)
			}
		}
		setGroupVariable(v, gv)
	case ircode.OpOpenScope,
		ircode.OpCloseScope:
		// Do nothing by intention
	case ircode.OpReturn:
		s.transformArguments(c, vs)
		for i, p := range s.f.Type().Out {
			if types.TypeHasPointers(p.Type) {
				et := types.NewExprType(p.Type)
				if et.PointerDestGroup == nil {
					panic("Oooops")
				}
				gv, ok := s.parameterGroupVariables[et.PointerDestGroup]
				if !ok {
					panic("Oooops")
				}
				gArg := argumentGroupVariable(c, c.Args[i], vs, c.Location)
				s.generateMerge(c, gv, gArg, vs)
			}
		}
	case ircode.OpCall:
		s.transformArguments(c, vs)
		ft, ok := types.GetFuncType(c.Args[0].Type().Type)
		if !ok {
			panic("Not a func")
		}
		irft := ircode.NewFunctionType(ft)
		var parameterGroupVariables map[*types.Group]*GroupVariable
		for i, p := range irft.In {
			if types.TypeHasPointers(p.Type) {
				if parameterGroupVariables == nil {
					parameterGroupVariables = make(map[*types.Group]*GroupVariable)
				}
				et := types.NewExprType(p.Type)
				if et.PointerDestGroup == nil {
					panic("Oooops")
				}
				gArg := argumentGroupVariable(c, c.Args[i+1], vs, p.Location)
				gv, ok := parameterGroupVariables[et.PointerDestGroup]
				if !ok {
					parameterGroupVariables[et.PointerDestGroup] = gArg
				} else {
					parameterGroupVariables[et.PointerDestGroup] = s.generateMerge(c, gv, gArg, vs)
				}
			}
		}
		for i, p := range irft.Out {
			v := vs.createDestinationVariableByIndex(c, i)
			// The destination variable is now initialized
			v.IsInitialized = true
			et := types.NewExprType(p.Type)
			if types.TypeHasPointers(et.Type) {
				if et.PointerDestGroup == nil {
					panic("Oooops")
				}
				gv, ok := parameterGroupVariables[et.PointerDestGroup]
				if !ok {
					gv = vs.newGroupVariable()
					// Assume that the function being called allocates some memory
					// and adds it to the group `gv`
					gv.Allocations++
					parameterGroupVariables[et.PointerDestGroup] = gv
				}
				setGroupVariable(v, gv)
			}
		}
		for _, g := range irft.GroupParameters {
			gv, ok := parameterGroupVariables[g]
			if !ok {
				panic("Oooops")
			}
			c.GroupArgs = append(c.GroupArgs, gv)
		}
	default:
		panic("Ooop")
	}
	return true
}

func (s *ssaTransformer) transformArguments(c *ircode.Command, vs *ssaScope) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			s.transformCommand(c.Args[i].Cmd, vs)
		} else if c.Args[i].Var != nil {
			_, v2 := vs.lookupVariable(c.Args[i].Var)
			if v2 == nil {
				panic("Oooops, variable does not exist " + c.Args[i].Var.Name)
			}
			if !ircode.IsVarInitialized(v2) {
				s.log.AddError(errlog.ErrorUninitializedVariable, c.Location, v2.Original.Name)
			}
			// Do not replace the first argument to OpSet/OpGet with a constant
			if v2.Type.IsConstant() && !(c.Op == ircode.OpSet && i == 0) && !(c.Op == ircode.OpGet && i == 0) {
				c.Args[i].Const = &ircode.Constant{ExprType: v2.Type}
				c.Args[i].Var = nil
			} else {
				c.Args[i].Var = v2
			}
			if c.Args[i].Var != nil {
				gv := groupVar(v2)
				if gv != nil {
					_, gv2 := vs.lookupGroup(gv)
					if gv2.Unavailable {
						s.log.AddError(errlog.ErrorGroupUnavailable, c.Location)
					}
					// If the group of the variable changed in the meantime, update the variable such that it refers to its current group
					if gv != gv2 {
						v := vs.newVariableUsageVersion(c.Args[i].Var)
						setGroupVariable(v, gv2)
						c.Args[i].Var = v
					}
				}
			}
		}
	}
}

func (s *ssaTransformer) generateMerge(c *ircode.Command, group1 *GroupVariable, group2 *GroupVariable, vs *ssaScope) *GroupVariable {
	newGroup, doMerge := vs.merge(group1, group2, nil, c, s.log)
	if doMerge {
		cmdMerge := &ircode.Command{Op: ircode.OpMerge, GroupArgs: []ircode.IGroupVariable{group1, group2}, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, Location: c.Location, Scope: c.Scope}
		c.PreBlock = append(c.PreBlock, cmdMerge)
	}
	return newGroup
}

/*
// accessChainHasPointers is used to check whether an accessChain needs the treatment of
// `accessChainGroupVariable`.
func accessChainHasPointers(c *ircode.Command) bool {
	for _, ac := range c.AccessChain {
		switch ac.Kind {
		case ircode.AccessAddressOf,
			ircode.AccessSlice,
			ircode.AccessSliceIndex,
			ircode.AccessPointerToStruct,
			ircode.AccessDereferencePointer:
			return true
		}
	}
	return false
}
*/

func (s *ssaTransformer) accessChainGroupVariable(c *ircode.Command, vs *ssaScope) *GroupVariable {
	// Shortcut in case the result of the access chain carries no pointers at all.
	// if !types.TypeHasPointers(c.Type.Type) {
	//	return nil
	//}
	if len(c.AccessChain) == 0 {
		panic("No access chain")
	}
	// The variable on which this access chain starts is stored as local variable in a scope.
	// Thus, the group of this value is a scoped group.
	var valueGroup *GroupVariable
	// The variable on which this access chain starts might have pointers.
	// Determine to group to which these pointers are pointing.
	var ptrDestGroup *GroupVariable
	if c.Args[0].Var != nil {
		valueGroup = scopeGroupVar(c.Args[0].Var.Scope)
		ptrDestGroup = groupVar(c.Args[0].Var)
	} else if c.Args[0].Const != nil {
		valueGroup = scopeGroupVar(c.Scope)
	} else {
		panic("Oooops")
	}
	if ptrDestGroup == nil {
		// The variable has no pointers. In this case the only possible operation is to take the address of take a slice.
		ptrDestGroup = valueGroup
	}
	argIndex := 1
	for _, ac := range c.AccessChain {
		switch ac.Kind {
		case ircode.AccessAddressOf:
			// The result of `&expr` must be a pointer.
			pt, ok := types.GetPointerType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a pointer")
			}
			// The resulting pointer is assigned to an unsafe pointer? -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			if ac.InputType.PointerDestGroup != nil && ac.InputType.PointerDestGroup.Kind == types.GroupIsolate {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value is in turn an isolate pointer. But that is ok, since the type system has this information in form of a GroupType.
				ptrDestGroup = valueGroup
			} else {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into one group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = s.generateMerge(c, valueGroup, ptrDestGroup, vs)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = scopeGroupVar(c.Scope)
		case ircode.AccessSlice:
			// The result of `expr[a:b]` must be a slice.
			_, ok := types.GetSliceType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a slice")
			}
			if _, ok := types.GetSliceType(ac.InputType.Type); ok {
				// Do nothing by intention. A slice of a slice points to the same group as the original slice.
			} else {
				_, ok := types.GetArrayType(ac.InputType.Type)
				if !ok {
					panic("Input is not a slice and not an array")
				}
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into one group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = s.generateMerge(c, valueGroup, ptrDestGroup, vs)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = scopeGroupVar(c.Scope)
			argIndex += 2
		case ircode.AccessStruct:
			_, ok := types.GetStructType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessPointerToStruct:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			_, ok = types.GetStructType(pt.ElementType)
			if !ok {
				panic("Not a struct")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
			argIndex++
		case ircode.AccessSliceIndex:
			_, ok := types.GetSliceType(ac.InputType.Type)
			if !ok {
				panic("Not a slice")
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
			argIndex++
		case ircode.AccessDereferencePointer:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessCast:
			// Do nothing by intention
			if types.IsUnsafePointerType(ac.OutputType.Type) {
				ptrDestGroup = nil
			}
		case ircode.AccessInc, ircode.AccessDec:
			// Do nothing by intention
		default:
			panic("Oooops")
		}
	}
	return ptrDestGroup
}

func (s *ssaTransformer) polishScope(block *ircode.Command, vs *ssaScope) {
	vs.polishBlock(block.Block)
	vs.polishBlock(block.PreBlock)
	// Variables which are subject to a phi-group need their own variable to track their group-pointer at runtime.
	for v := range vs.vars {
		if v.Original != v || v.Scope != block.Scope || !v.HasPhiGroup {
			continue
		}
		openScope := block.Block[0]
		if openScope.Op != ircode.OpOpenScope {
			panic("Oooops")
		}
		t := &types.ExprType{Type: &types.PointerType{ElementType: types.PrimitiveTypeUintptr, Mode: types.PtrUnsafe}}
		gv := &ircode.Variable{Kind: ircode.VarDefault, Name: "gv_" + v.Original.Name, Type: t, Scope: block.Scope}
		gv.Original = gv
		c := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{gv}, Type: t, Location: block.Location, Scope: block.Scope}
		openScope.Block = append(openScope.Block, c)
		v.PhiGroupVariable = gv
	}
}

// Add code to free memory groups and add variables for storing memory group pointers.
func (s *ssaTransformer) transformScopesPhase1() {
	for _, scope := range s.scopes {
		for gv, gvNew := range scope.groups {
			// Ignore groups that have been merged by others (gv != gvNew).
			if gv != gvNew {
				continue
			}
			// Ignore parameter groups, since these are not free'd and their pointers are parameters of the function.
			// Ignore groups that merge other groups (len(gv.In) != 0)
			// Ignore groups which are never associated with any allocation.
			if gv.IsParameter() || len(gv.In) != 0 || len(gv.InPhi) != 0 || (gv.Via == nil && scope.NoAllocations(gv)) {
				continue
			}
			vs := findTerminatingScope(gv, scope)
			openScope := vs.block.Block[0]
			if openScope.Op != ircode.OpOpenScope {
				panic("Oooops")
			}
			t := &types.ExprType{Type: types.PrimitiveTypeUintptr}
			v := &ircode.Variable{Kind: ircode.VarDefault, Name: gv.Name, Type: t, Scope: vs.block.Scope}
			v.Original = v
			s.f.Vars = append(s.f.Vars, v)
			c := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{v}, Type: v.Type, Location: vs.block.Location, Scope: vs.block.Scope}
			c2 := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{v}, Args: []ircode.Argument{ircode.NewIntArg(0)}, Type: v.Type, Location: vs.block.Location, Scope: vs.block.Scope}
			gv.Var = v
			gv.Close()
			// Add the variable definition and its assignment to openScope of the block
			openScope.Block = append(openScope.Block, c, c2)

			if gv.Via == nil && gv.Constraints.NamedGroup == "" {
				c := &ircode.Command{Op: ircode.OpFree, GroupArgs: []ircode.IGroupVariable{gv}, Location: vs.block.Location, Scope: vs.block.Scope}
				closeScope := vs.block.Block[len(vs.block.Block)-1]
				if closeScope.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScope.Block = append(closeScope.Block, c)
			}
		}
		// Search for places where groups are first used.
		// Some of these groups require a helper group variable.
		for _, c := range scope.block.Block {
			for _, arg := range c.Args {
				if arg.Var != nil {
					if arg.Var.GroupInfo != nil {
						s.addHelperGroupVar(scope.block, c, arg.Var.GroupInfo.(*GroupVariable))
					}
				} else if arg.Const != nil {
					if arg.Const.GroupInfo != nil {
						s.addHelperGroupVar(scope.block, c, arg.Const.GroupInfo.(*GroupVariable))
					}
				}
			}
			for _, dest := range c.Dest {
				if dest != nil && dest.GroupInfo != nil {
					s.addHelperGroupVar(scope.block, c, dest.GroupInfo.(*GroupVariable))
				}
			}
		}
	}
}

func (s *ssaTransformer) addHelperGroupVar(block *ircode.Command, c *ircode.Command, gv *GroupVariable) {
	if len(gv.In) == 0 || gv.Var != nil {
		return
	}
	for _, gIn := range gv.In {
		if !gIn.isPhi() {
			return
		}
	}
	// If the new group must rely on the PhiVariables of its input-groups, the new group needs a new variable of its own,
	// because the PhiVariables are subject to change
	t := &types.ExprType{Type: &types.PointerType{ElementType: types.PrimitiveTypeUintptr, Mode: types.PtrUnsafe}}
	v := &ircode.Variable{Kind: ircode.VarDefault, Name: gv.Name, Type: t, Scope: block.Scope}
	v.Original = v
	s.f.Vars = append(s.f.Vars, v)
	cmdVar := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{v}, Type: v.Type, Location: block.Location, Scope: block.Scope}
	cmdSet := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{v}, Args: []ircode.Argument{ircode.NewVarArg(gv.In[0].Variable())}, Type: v.Type, Location: block.Location, Scope: block.Scope}
	c.PreBlock = append(c.PreBlock, cmdVar, cmdSet)
	gv.Var = v
}

// Searches the top-most scope in which a group variable (or one of its dependent groups) is used.
// This is the scope where a group can be savely free'd.
func findTerminatingScope(gv *GroupVariable, vs *ssaScope) *ssaScope {
	if len(gv.Out) == 0 || gv.marked {
		return vs
	}
	gv.marked = true
	var p *ssaScope = vs
	for _, out := range gv.Out {
		outScope := findTerminatingScope(gv, out.scope)
		if p.hasParent(outScope) {
			p = outScope
		}
	}
	gv.marked = true
	return p
}

// Add code to free memory groups and add variables for storing memory group pointers.
func (s *ssaTransformer) transformScopesPhase2() {
	for _, scope := range s.scopes {
		for _, c := range scope.block.Block {
			if len(c.Dest) == 1 && c.Dest[0] != nil && c.Dest[0].Original.HasPhiGroup {
				phi := c.Dest[0].Original.PhiGroupVariable
				gv := c.Dest[0].GroupInfo.Variable()
				if gv == nil {
					panic("Oooops")
				}
				if _, ok := types.GetPointerType(gv.Type.Type); ok {
					cmdSet := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{phi}, Args: []ircode.Argument{ircode.NewVarArg(gv)}, Type: phi.Type, Location: c.Location, Scope: c.Scope}
					c.PreBlock = append(c.PreBlock, cmdSet)
				} else {
					ac := []ircode.AccessChainElement{ircode.AccessChainElement{Kind: ircode.AccessAddressOf, InputType: gv.Type, OutputType: phi.Type}}
					cmdSet := &ircode.Command{Op: ircode.OpGet, Dest: []*ircode.Variable{phi}, Args: []ircode.Argument{ircode.NewVarArg(gv)}, AccessChain: ac, Type: phi.Type, Location: c.Location, Scope: c.Scope}
					c.PreBlock = append(c.PreBlock, cmdSet)
				}
			}
		}
	}
}

// TransformToSSA checks the control flow and detects unreachable code.
// Thereby it translates IR-code Variables into Single-Static-Assignment which is
// required for further optimizations and code analysis.
// Furthermore, it checks groups and whether the IR-code (and therefore the original code)
// complies with the grouping rules.
// In addition, the transformation adds code for merging and freeing memory and additional
// variables to track such memory.
func TransformToSSA(f *ircode.Function, groupStorageVars map[*types.Group]*ircode.Variable, globalVars []*ircode.Variable, log *errlog.ErrorLog) {
	s := &ssaTransformer{f: f, log: log}
	s.topLevelScope = newScope(s, &f.Body)
	s.namedGroupVariables = make(map[string]*GroupVariable)
	s.scopedGroupVariables = make(map[*ircode.CommandScope]*GroupVariable)
	s.parameterGroupVariables = make(map[*types.Group]*GroupVariable)
	s.topLevelScope.kind = scopeFunc
	for _, v := range globalVars {
		s.topLevelScope.vars[v] = v
	}
	openScope := f.Body.Block[0]
	if openScope.Op != ircode.OpOpenScope {
		panic("Oooops")
	}
	// Create a GroupVariable for all groups used in the function's parameters
	for g, v := range groupStorageVars {
		gvNamed := s.topLevelScope.newNamedGroupVariable(g.Name)
		gvNamed.Close()
		gvNamed.Var = v
		gv := s.topLevelScope.newGroupVariable()
		gv.Var = v
		s.namedGroupVariables[g.Name] = gv
		gv.addInput(gvNamed)
		gvNamed.addOutput(gv)
		s.parameterGroupVariables[g] = gv
	}
	// Iterate over all parameters and determine their group
	for _, v := range f.Vars {
		if v.Kind != ircode.VarParameter {
			continue
		}
		v.IsInitialized = true
		if v.Type.PointerDestGroup != nil {
			/*
				gvNamed := s.topLevelScope.newNamedGroupVariable(groupName)
				gvNamed.Close()
				gv := s.topLevelScope.newGroupVariable()
				s.namedGroupVariables[groupName] = gv
				gv.addInput(gvNamed)
				gvNamed.addOutput(gv)
			*/
			gv, ok := s.parameterGroupVariables[v.Type.PointerDestGroup]
			if !ok {
				panic("Oooops")
			}
			setGroupVariable(v, gv)
			/*
				// Create a variable that stores the pointer to this group
				t := &types.ExprType{Type: types.PrimitiveTypeUintptr}
				// TODO: Make this a VarParameter
				vgrp := &ircode.Variable{Kind: ircode.VarDefault, Name: v.Type.PointerDestGroup.Name, Type: t, Scope: f.Body.Scope, IsInitialized: true}
				vgrp.Original = vgrp
				s.f.Vars = append(s.f.Vars, vgrp)
				c := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{vgrp}, Type: v.Type, Location: f.Body.Location, Scope: f.Body.Scope}
				openScope.Block = append(openScope.Block, c)
				gv.Var = vgrp
			*/
		}
		s.topLevelScope.vars[v] = v
	}
	f.Body.Scope.GroupInfo = s.topLevelScope.newScopedGroupVariable(f.Body.Scope)
	s.transformBlock(&f.Body, s.topLevelScope)
	s.polishScope(&f.Body, s.topLevelScope)
	s.transformScopesPhase1()
	s.transformScopesPhase2()
}
