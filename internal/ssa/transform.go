package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type ssaTransformer struct {
	f   *ircode.Function
	log *errlog.ErrorLog
	// Groupings
	namedGroupings map[string]*Grouping
	// Groupings which are linked to a scope.
	scopedGroupings map[*ircode.CommandScope]*Grouping
	// Groupings which are linked to a function parameter.
	parameterGroupings map[*types.GroupSpecifier]*Grouping
	scopes             []*ssaScope
	topLevelScope      *ssaScope
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
		c.Scope.Grouping = vs.newScopedGrouping(c.Scope)
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
			c.Else.Scope.Grouping = vs.newScopedGrouping(c.Else.Scope)
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
		c.Scope.Grouping = vs.newScopedGrouping(c.Scope)
		loopScope := newScope(s, c)
		loopScope.parent = vs
		loopScope.kind = scopeLoop
		doesLoop := s.transformBlock(c, loopScope)
		if doesLoop {
			closeScopeCommand := c.Block[len(c.Block)-1]
			if closeScopeCommand.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			// The loop can run more than once
			loopScope.mergeVariablesOnContinue(closeScopeCommand, loopScope, s.log)
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
		v := c.Dest[0]
		if v.Kind == ircode.VarParameter {
			vs.defineVariable(v)
		}
		if types.TypeHasPointers(v.Type.Type) {
			if v.Type.PointerDestGroupSpecifier != nil {
				if v.Kind == ircode.VarParameter {
					// Parameters with pointers have groups already when the function is being called.
					grouping, ok := s.parameterGroupings[c.Dest[0].Type.PointerDestGroupSpecifier]
					if !ok {
						panic("Ooooops")
					}
					setGrouping(v, grouping)
					/*
						if gspec.Kind == types.GroupSpecifierIsolate {
							setGrouping(c.Dest[0], vs.newGrouping())
						} else {
							setGrouping(c.Dest[0], vs.newNamedGrouping(gspec.Name))
						}
					*/
				} else {
					setGrouping(c.Dest[0], vs.newGrouping())
				}
			}
		}
	case ircode.OpPrintln:
		s.transformArguments(c, vs)
	case ircode.OpSet:
		s.transformArguments(c, vs)
		gDest := s.accessChainGrouping(c, vs)
		gSrc := argumentGrouping(c, c.Args[len(c.Args)-1], vs, c.Location)
		// Assigning a pointer type?
		if types.TypeHasPointers(c.Args[len(c.Args)-1].Type().Type) {
			outType := c.AccessChain[len(c.AccessChain)-1].OutputType
			// When assigning a variable with isolated grouping, this variable becomes unavailable
			if outType.PointerDestGroupSpecifier != nil && outType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				gSrc.makeUnavailable()
			} else {
				s.generateMerge(c, gDest, gSrc, vs)
			}
		}
		if len(c.Dest) > 1 {
			panic("Oooops")
		} else if len(c.Dest) == 1 {
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
		if types.TypeHasPointers(v.Type.Type) {
			// The group resulting in the Get operation becomes the group of the destination
			setGrouping(v, s.accessChainGrouping(c, vs))
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
			setGrouping(v, argumentGrouping(c, c.Args[0], vs, c.Location))
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
			var gv *Grouping
			for i := range c.Args {
				gArg := argumentGrouping(c, c.Args[i], vs, c.Location)
				if gArg != nil {
					if gv == nil {
						gv = gArg
					} else {
						gv = s.generateMerge(c, gv, gArg, vs)
					}
				}
			}
			if gv == nil {
				gv = vs.newGrouping()
			}
			setGrouping(v, gv)
			if allocMemory {
				gv.Allocations++
			}
		} else if allocMemory {
			gv := vs.newGrouping()
			setGrouping(v, gv)
			gv.Allocations++
		}
	case ircode.OpAppend:
		s.transformArguments(c, vs)
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
		gv := argumentGrouping(c, c.Args[0], vs, c.Location)
		sl, ok := types.GetSliceType(v.Type.Type)
		if !ok {
			panic("Ooooops")
		}
		if types.TypeHasPointers(sl.ElementType) {
			// Ignore the first two arguments which are the slice and the amount of elements to add
			for _, arg := range c.Args[2:] {
				gArg := argumentGrouping(c, arg, vs, c.Location)
				gv = s.generateMerge(c, gv, gArg, vs)
			}
		}
		setGrouping(v, gv)
	case ircode.OpOpenScope,
		ircode.OpCloseScope:
		// Do nothing by intention
	case ircode.OpReturn:
		s.transformArguments(c, vs)
		for i, p := range s.f.Type().Out {
			if types.TypeHasPointers(p.Type) {
				et := types.NewExprType(p.Type)
				if et.PointerDestGroupSpecifier == nil {
					panic("Oooops")
				}
				gv, ok := s.parameterGroupings[et.PointerDestGroupSpecifier]
				if !ok {
					panic("Oooops")
				}
				gArg := argumentGrouping(c, c.Args[i], vs, c.Location)
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
		parameterGroupings := make(map[*types.GroupSpecifier]*Grouping)
		// Determine the groupings that are bound to the group specifiers of the callee.
		for i, p := range irft.In {
			if types.TypeHasPointers(p.Type) {
				pet := types.NewExprType(p.Type)
				if pet.PointerDestGroupSpecifier == nil {
					panic("Oooops")
				}
				gArg := argumentGrouping(c, c.Args[i+1], vs, p.Location)
				gv, ok := parameterGroupings[pet.PointerDestGroupSpecifier]
				if !ok {
					parameterGroupings[pet.PointerDestGroupSpecifier] = gArg
				} else {
					parameterGroupings[pet.PointerDestGroupSpecifier] = s.generateMerge(c, gv, gArg, vs)
				}
			}
		}
		// Determine (or create) th
		for i, p := range irft.Out {
			v := vs.createDestinationVariableByIndex(c, i)
			// The destination variable is now initialized
			v.IsInitialized = true
			ret := types.NewExprType(p.Type)
			if types.TypeHasPointers(ret.Type) {
				if ret.PointerDestGroupSpecifier == nil {
					panic("Oooops")
				}
				gv, ok := parameterGroupings[ret.PointerDestGroupSpecifier]
				if !ok {
					gv = vs.newGrouping()
					// Assume that the function being called allocates some memory
					// and adds it to the group `gv`
					gv.Allocations++
					parameterGroupings[ret.PointerDestGroupSpecifier] = gv
				}
				setGrouping(v, gv)
			}
		}
		// Add the group pointers to the list of group-arguments
		for _, g := range irft.GroupSpecifiers {
			gv, ok := parameterGroupings[g]
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
				gv := grouping(v2)
				if gv != nil {
					_, gv2 := vs.lookupGrouping(gv)
					if gv2.IsProbablyUnavailable() {
						s.log.AddError(errlog.ErrorGroupUnavailable, c.Location)
					}
					// If the group of the variable changed in the meantime, update the variable such that it refers to its current group
					if gv != gv2 {
						v := vs.newVariableUsageVersion(c.Args[i].Var)
						setGrouping(v, gv2)
						c.Args[i].Var = v
					}
				}
			}
		}
	}
}

func (s *ssaTransformer) generateMerge(c *ircode.Command, group1 *Grouping, group2 *Grouping, vs *ssaScope) *Grouping {
	newGroup, doMerge := vs.merge(group1, group2, nil, c, s.log)
	if doMerge {
		cmdMerge := &ircode.Command{Op: ircode.OpMerge, GroupArgs: []ircode.IGrouping{group1, group2}, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, Location: c.Location, Scope: c.Scope}
		c.PreBlock = append(c.PreBlock, cmdMerge)
	}
	return newGroup
}

func (s *ssaTransformer) accessChainGrouping(c *ircode.Command, vs *ssaScope) *Grouping {
	// Shortcut in case the result of the access chain carries no pointers at all.
	// if !types.TypeHasPointers(c.Type.Type) {
	//	return nil
	//}
	if len(c.AccessChain) == 0 {
		panic("No access chain")
	}
	// The variable on which this access chain starts is stored as local variable in a scope.
	// Thus, the group of this value is a scoped group.
	var valueGroup *Grouping
	// The variable on which this access chain starts might have pointers.
	// Determine to group to which these pointers are pointing.
	var ptrDestGroup *Grouping
	if c.Args[0].Var != nil {
		valueGroup = scopeGrouping(c.Args[0].Var.Scope)
		ptrDestGroup = grouping(c.Args[0].Var)
	} else if c.Args[0].Const != nil {
		valueGroup = scopeGrouping(c.Scope)
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
			if ac.InputType.PointerDestGroupSpecifier != nil && ac.InputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
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
			valueGroup = scopeGrouping(c.Scope)
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
			valueGroup = scopeGrouping(c.Scope)
			argIndex += 2
		case ircode.AccessStruct:
			_, ok := types.GetStructType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newNamedGrouping(ac.OutputType.PointerDestGroupSpecifier)
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
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newNamedGrouping(ac.OutputType.PointerDestGroupSpecifier)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newNamedGrouping(ac.OutputType.PointerDestGroupSpecifier)
			}
			argIndex++
		case ircode.AccessSliceIndex:
			_, ok := types.GetSliceType(ac.InputType.Type)
			if !ok {
				panic("Not a slice")
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newNamedGrouping(ac.OutputType.PointerDestGroupSpecifier)
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
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newNamedGrouping(ac.OutputType.PointerDestGroupSpecifier)
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

// polishScope creates group pointer variables for all variables with phi-grouping.
func (s *ssaTransformer) polishScope(block *ircode.Command, vs *ssaScope) {
	vs.polishBlock(block.Block)
	vs.polishBlock(block.PreBlock)
	// Variables which are subject to a phi-grouping need their own variable to track their group-pointer at runtime.
	// This variable is created once for the original variable in its defining scope and it is shared with all other
	// versions of the same variable.
	for v := range vs.vars {
		if v.Original != v || v.Scope != block.Scope || !v.HasPhiGrouping {
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

// Add code to free groups and add variables for storing group pointers.
// The group pointer variables are added to the top-most scope in which memory belonging to a grouping
// could possibly be used.
func (s *ssaTransformer) transformScopesPhase1() {
	for _, scope := range s.scopes {
		for gv, gvNew := range scope.groupings {
			// Ignore groups that have been merged by others (gv != gvNew).
			if gv != gvNew {
				continue
			}
			// Ignore parameter groups, since these are not free'd and their pointers are parameters of the function.
			// Ignore groups that merge other groups (len(gv.In) != 0).
			// Ignore phi-groups, since they only point to underlying group pointers. Thus phi-groups are not free'd themselfes.
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

			if gv.Via == nil && gv.Constraints.NamedGroup == "" && !gv.unavailable {
				c := &ircode.Command{Op: ircode.OpFree, Args: []ircode.Argument{ircode.NewVarArg(gv.Var)}, Location: vs.block.Location, Scope: vs.block.Scope}
				closeScope := vs.block.Block[len(vs.block.Block)-1]
				if closeScope.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScope.Block = append(closeScope.Block, c)
			}
		}
		/*
			// Search for places where groups are first used.
			// Some of these groups require a helper group variable.
			for _, c := range scope.block.Block {
				for _, arg := range c.Args {
					if arg.Var != nil {
						if arg.Var.Grouping != nil {
							s.addHelperGroupVar(scope.block, c, arg.Var.Grouping.(*Grouping))
						}
					} else if arg.Const != nil {
						if arg.Const.Grouping != nil {
							s.addHelperGroupVar(scope.block, c, arg.Const.Grouping.(*Grouping))
						}
					}
				}
				for _, dest := range c.Dest {
					if dest != nil && dest.Grouping != nil {
						s.addHelperGroupVar(scope.block, c, dest.Grouping.(*Grouping))
					}
				}
			}
		*/
	}
}

/*
func (s *ssaTransformer) addHelperGroupVar(block *ircode.Command, c *ircode.Command, gv *Grouping) {
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
	cmdSet := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{v}, Args: []ircode.Argument{ircode.NewVarArg(gv.In[0].GroupVariable())}, Type: v.Type, Location: block.Location, Scope: block.Scope}
	c.PreBlock = append(c.PreBlock, cmdVar, cmdSet)
	gv.Var = v
}
*/

// Searches the top-most scope in which a group variable (or one of its dependent groups) is used.
// This is the scope where a group can be savely free'd.
func findTerminatingScope(grouping *Grouping, vs *ssaScope) *ssaScope {
	if len(grouping.Out) == 0 || grouping.marked {
		return vs
	}
	grouping.marked = true
	p := vs
	for _, out := range grouping.Out {
		outScope := findTerminatingScope(out, out.scope)
		if p.hasParent(outScope) {
			p = outScope
		}
	}
	grouping.marked = false
	return p
}

// transformScopesPhase2 adds instructions for assigning a group pointer value to phi-group-pointer variables
// if the corresponding variable is assigned in this scope.
func (s *ssaTransformer) transformScopesPhase2() {
	for _, scope := range s.scopes {
		for _, c := range scope.block.Block {
			// The computation has an assignment and the variable being assigned to has a phi-grouping?
			if len(c.Dest) == 1 && c.Dest[0] != nil && c.Dest[0].Original.HasPhiGrouping {
				phi := c.Dest[0].Original.PhiGroupVariable
				gv := c.Dest[0].Grouping.GroupVariable()
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
func TransformToSSA(f *ircode.Function, parameterGroupVars map[*types.GroupSpecifier]*ircode.Variable, globalVars []*ircode.Variable, log *errlog.ErrorLog) {
	s := &ssaTransformer{f: f, log: log}
	s.topLevelScope = newScope(s, &f.Body)
	s.topLevelScope.kind = scopeFunc
	s.scopedGroupings = make(map[*ircode.CommandScope]*Grouping)
	s.parameterGroupings = make(map[*types.GroupSpecifier]*Grouping)
	// Add global variables to the top-level scope
	for _, v := range globalVars {
		s.topLevelScope.vars[v] = v
	}
	openScope := f.Body.Block[0]
	if openScope.Op != ircode.OpOpenScope {
		panic("Oooops")
	}
	// Create a Grouping for all group variables used in the function's parameters
	for g, v := range parameterGroupVars {
		// Note that `v` is the variable that stores the group pointer for the grouping `g`.
		paramGrouping := s.topLevelScope.newNamedGrouping(g)
		paramGrouping.isParameter = true
		paramGrouping.Close()
		paramGrouping.Var = v
		s.parameterGroupings[g] = paramGrouping
		/*
			// Create a second grouping that is open and takes the first one as input.
			paramGrouping2 := s.topLevelScope.newGrouping()
			paramGrouping2.Var = v
			paramGrouping2.addInput(paramGrouping)
			paramGrouping.addOutput(paramGrouping2)
			s.parameterGroupings[g] = paramGrouping2
			s.topLevelScope.groupings[paramGrouping] = paramGrouping2
		*/
	}
	// Mark all input parameters as initialized.
	for _, v := range f.InVars {
		// Parameters are always initialized upon function invcation.
		v.IsInitialized = true
	}
	// Add a grouping to the function scope of the ircode.
	f.Body.Scope.Grouping = s.topLevelScope.newScopedGrouping(f.Body.Scope)
	s.transformBlock(&f.Body, s.topLevelScope)
	s.polishScope(&f.Body, s.topLevelScope)
	s.transformScopesPhase1()
	s.transformScopesPhase2()
}
