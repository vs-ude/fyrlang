package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type stackUnwinding struct {
	command      *ircode.Command
	commandScope *ssaScope
	topScope     *ssaScope
}

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
	step               int
	stackUnwindings    []stackUnwinding
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

func (s *ssaTransformer) transformIterBlock(c *ircode.Command, vs *ssaScope) bool {
	for i, c2 := range c.IterBlock {
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
	s.step++
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
		// visit the else-clause
		if c.Else != nil {
			c.Else.Scope.Grouping = vs.newScopedGrouping(c.Else.Scope)
			elseScope := newScope(s, c.Else)
			elseScope.parent = vs
			elseScope.kind = scopeIf
			elseCompletes := s.transformBlock(c.Else, elseScope)
			if ifCompletes && elseCompletes {
				// Control flow flows through the if-clause or else-clause and continues afterwards
				s.mergeVariablesOnIfElse(c, vs, ifScope, elseScope)
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
			s.mergeVariablesOnIf(c, vs, ifScope)
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
		s.transformIterBlock(c, loopScope)
		s.createLoopPhiGroupVars(c, loopScope)
		s.mergeVariablesOnBreaks(loopScope)
		if doesLoop {
			// The loop can run more than once
			s.mergeVariablesOnLoop(c, loopScope)
		}
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
		s.mergeVariablesOnBreak(c, loopScope, vs)
		loopScope.breakCount++
		s.stackUnwindings = append(s.stackUnwindings, stackUnwinding{command: c, commandScope: vs, topScope: loopScope})
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
		s.mergeVariablesOnContinue(c, loopScope, vs)
		loopScope.continueCount++
		s.stackUnwindings = append(s.stackUnwindings, stackUnwinding{command: c, commandScope: vs, topScope: loopScope})
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
				} else {
					setGrouping(c.Dest[0], vs.newDefaultGrouping())
				}
			}
		}
	case ircode.OpSet:
		s.transformArguments(c, vs)
		gDest := s.accessChainGrouping(c, vs)
		gSrc := argumentGrouping(c, c.Args[len(c.Args)-1], vs, c.Location)
		// Assigning a pointer type?
		if types.TypeHasPointers(c.Args[len(c.Args)-1].Type().Type) {
			outType := c.AccessChain[len(c.AccessChain)-1].OutputType
			// When assigning a variable with isolated grouping, this variable becomes unavailable
			if outType.PointerDestGroupSpecifier != nil && outType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				vs.newUnavailableGroupingVersion(gSrc)
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
	case ircode.OpPanic,
		ircode.OpPrintln:
		s.transformArguments(c, vs)
		c.GroupArgs = append(c.GroupArgs, argumentGrouping(c, c.Args[0], vs, c.Location))
	case ircode.OpGroupOf:
		s.transformArguments(c, vs)
		c.GroupArgs = append(c.GroupArgs, argumentGrouping(c, c.Args[0], vs, c.Location))
		v := vs.createDestinationVariable(c)
		// The destination variable is now initialized
		v.IsInitialized = true
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
			if gv == nil || gv.IsConstant() {
				gv = vs.newDefaultGrouping()
			}
			setGrouping(v, gv)
			if allocMemory {
				gv.Allocations++
			}
		} else if allocMemory {
			gv := vs.newDefaultGrouping()
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
		s.stackUnwindings = append(s.stackUnwindings, stackUnwinding{command: c, commandScope: vs, topScope: s.topLevelScope})
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
					gv = vs.newDefaultGrouping()
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
					gv2 := vs.lookupGrouping(gv)
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

func (s *ssaTransformer) setStaticMergePoint(gv *Grouping, staticMergePoint *groupingAllocationPoint, vs *ssaScope, force bool) {
	if !force && gv.Original.staticMergePoint == staticMergePoint {
		return
	}
	gv.Original.staticMergePoint = staticMergePoint
	if gv.Kind == StaticMergeGrouping {
		for _, group := range gv.Input {
			if group.Kind == StaticMergeGrouping || group.Kind == DefaultGrouping {
				group = vs.lookupGrouping(group)
				s.setStaticMergePoint(group, staticMergePoint, vs, false)
			}
		}
	}
	for _, group := range gv.Output {
		if group.Kind == StaticMergeGrouping {
			group = vs.lookupGrouping(group)
			s.setStaticMergePoint(group, staticMergePoint, vs, false)
		}
	}
}

func (s *ssaTransformer) setPhiAllocationPoint(gv *Grouping, phiAllocationPoint *groupingAllocationPoint, vs *ssaScope) {
	if gv.Original.phiAllocationPoint == phiAllocationPoint {
		return
	}
	gv.Original.phiAllocationPoint = phiAllocationPoint
	if gv.Kind == StaticMergeGrouping {
		for _, group := range gv.Input {
			if group.Kind == StaticMergeGrouping || group.Kind == DefaultGrouping {
				group = vs.lookupGrouping(group)
				s.setPhiAllocationPoint(group, phiAllocationPoint, vs)
			}
		}
	}
	for _, group := range gv.Output {
		if group.Kind == StaticMergeGrouping {
			group = vs.lookupGrouping(group)
			s.setPhiAllocationPoint(group, phiAllocationPoint, vs)
		}
	}
}

func (s *ssaTransformer) generateMerge(c *ircode.Command, group1 *Grouping, group2 *Grouping, vs *ssaScope) *Grouping {
	newGroup, doMerge := s.merge(vs, group1, group2, c, s.log)
	if doMerge {
		cmdMerge := &ircode.Command{Op: ircode.OpMerge, GroupArgs: []ircode.IGrouping{group1, group2}, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, Location: c.Location, Scope: c.Scope}
		c.PreBlock = append(c.PreBlock, cmdMerge)
	}
	return newGroup
}

func (s *ssaTransformer) merge(vs *ssaScope, gv1 *Grouping, gv2 *Grouping, c *ircode.Command, log *errlog.ErrorLog) (*Grouping, bool) {
	// Get the latest versions and the scope in which they have been defined
	gvA := vs.lookupGrouping(gv1)
	gvB := vs.lookupGrouping(gv2)

	// The trivial case
	if gvA.Original == gvB.Original {
		return gvA, false
	}

	// Never merge constant groupings with any other groupings
	if gvA.IsConstant() {
		return gvB, false
	}
	if gvB.IsConstant() {
		return gvA, false
	}

	var staticMergePoint *groupingAllocationPoint
	canMergeStatically := false
	if gvB.Original.staticMergePoint != nil && gvB.Original.staticMergePoint.scope == vs && (vs == gvA.Original.scope || vs.hasParent(gvA.Original.scope)) {
		canMergeStatically = true
		staticMergePoint = gvA.Original.staticMergePoint
	} else if gvA.Original.staticMergePoint != nil && gvA.Original.staticMergePoint.scope == vs && (vs == gvB.Original.scope || vs.hasParent(gvB.Original.scope)) {
		staticMergePoint = gvB.Original.staticMergePoint
		tmp := gvA
		gvA = gvB
		gvB = tmp
		canMergeStatically = true
	}

	if canMergeStatically && gvA.Original.phiAllocationPoint != nil && !gvA.Original.phiAllocationPoint.isEarlierThan(gvB.Original.staticMergePoint) {
		// Do not merge statically with a phi-group when the phi-group-var has not been assigned at the time `gvB` is being used.
		canMergeStatically = false
	} else if !canMergeStatically && gvA.Original.phiAllocationPoint != nil && gvB.Original.phiAllocationPoint == nil {
		// Make the phi-group as the second input to a DynamicMergeGrouping.
		// This way its phiAllocationPoint remains nil and the DynamicMergeGrouping can still be used as input to static merges.
		tmp := gvA
		gvA = gvB
		gvB = tmp
	} else if !canMergeStatically && gvA.Original.phiAllocationPoint != nil && gvB.Original.phiAllocationPoint != nil && gvB.Original.phiAllocationPoint.isEarlierThan(gvA.Original.phiAllocationPoint) {
		tmp := gvA
		gvA = gvB
		gvB = tmp
	}

	if gvA.scope != vs {
		gvA = vs.newGroupingVersion(gvA)
	}
	if gvB.scope != vs {
		gvB = vs.newGroupingVersion(gvB)
	}

	if canMergeStatically {
		// Merge `gvA` and `gvB` statically into a new grouping
		grouping := vs.newStaticMergeGrouping()
		grouping.addInput(gvA)
		grouping.addInput(gvB)
		gvA.addOutput(grouping)
		gvB.addOutput(grouping)
		s.setStaticMergePoint(grouping, staticMergePoint, vs, true)
		s.setPhiAllocationPoint(grouping, gvA.Original.phiAllocationPoint, vs)
		println("----> STATIC MERGE", grouping.GroupingName(), "=", gvA, gvA.GroupingName(), gvB, gvB.GroupingName())
		return grouping, false
	}

	// Merge `gvA` and `gvB` dynamically into a new grouping
	grouping := vs.newDynamicMergeGrouping()
	grouping.addInput(gvA)
	grouping.addInput(gvB)
	gvA.addOutput(grouping)
	gvB.addOutput(grouping)
	s.setPhiAllocationPoint(grouping, gvA.Original.phiAllocationPoint, vs)

	println("----> DYN MERGE", grouping.GroupingName(), "=", gvA.GroupingName(), gvB.GroupingName())
	return grouping, true
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
				ptrDestGroup = vs.newGroupingFromSpecifier(ac.OutputType.PointerDestGroupSpecifier)
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
				ptrDestGroup = vs.newGroupingFromSpecifier(ac.OutputType.PointerDestGroupSpecifier)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				ptrDestGroup = vs.newViaGrouping(valueGroup)
			} else if ac.OutputType.PointerDestGroupSpecifier != nil && ac.OutputType.PointerDestGroupSpecifier.Kind == types.GroupSpecifierNamed {
				ptrDestGroup = vs.newGroupingFromSpecifier(ac.OutputType.PointerDestGroupSpecifier)
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
				ptrDestGroup = vs.newGroupingFromSpecifier(ac.OutputType.PointerDestGroupSpecifier)
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
				ptrDestGroup = vs.newGroupingFromSpecifier(ac.OutputType.PointerDestGroupSpecifier)
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

func (s *ssaTransformer) mergeScopes(dest *ssaScope, src *ssaScope) {
	s.mergeScopesIntern(dest.groupings, src.groupings)
}

func (s *ssaTransformer) mergeScopesIntern(dest map[*Grouping]*Grouping, src map[*Grouping]*Grouping) {
	for original, latest := range src {
		if destLatest, ok := dest[original]; ok {
			println("MERGE SCOPES", destLatest, latest)
			for _, grp := range latest.Input {
				destLatest.addInput(grp)
			}
			for _, grp := range latest.Output {
				destLatest.addOutput(grp)
			}
		} else {
			// println("SET SCOPES", original, latest)
			dest[original] = latest
		}
	}
}

// mergeVariablesOnIf generates phi-variables and phi-groups.
// `c` is the `OpIf` command.
// `vs` is the parent scope and `ifScope` is the scope of the if-clause.
func (s *ssaTransformer) mergeVariablesOnIf(c *ircode.Command, vs *ssaScope, ifScope *ssaScope) {
	// println("---> mergeIf", len(ifScope.vars))
	// Search for all variables that are assigned in the ifScope and the parent scope.
	// These variable become phi-variables, because they are assigned inside and outside the if-clause.
	// We ignore variables which are only "used" (but not assigned) inside the if-clause.
	for vo, v1 := range ifScope.vars {
		_, v2 := vs.searchVariable(vo)
		// The variable does not exist in the parent scope? Ignore.
		if v2 == nil {
			continue
		}
		// The variable has been changed in the if-clause? (the name includes the version number)
		if v1.Name != v2.Name {
			phiVar := vs.newPhiVariable(vo)
			phiVar.Phi = append(phiVar.Phi, v1, v2)
			vs.vars[vo] = phiVar
			phiGrouping := s.createPhiGrouping(phiVar, v1, v2, vs, ifScope, vs)
			if phiGrouping != nil {
				// Set the phi-group-variable before the if-clause executes
				cmdSet1 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[1]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				c.PreBlock = append(c.PreBlock, cmdSet1)
				// Set the phi-group-variable in the if-clause
				cmdSet0 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[0]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				closeScopeCommand := c.Block[len(c.Block)-1]
				if closeScopeCommand.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet0)
			}
		}
	}

	s.mergeScopes(vs, ifScope)
}

// mergeVariablesOnIfElse generates phi-variables and phi-groups.
// `c` is the `OpIf` command.
// `vs` is the parent scope and `ifScope` is the scope of the if-clause.
func (s *ssaTransformer) mergeVariablesOnIfElse(c *ircode.Command, vs *ssaScope, ifScope *ssaScope, elseScope *ssaScope) {
	for vo, v1 := range ifScope.vars {
		_, v2 := vs.searchVariable(vo)
		// The variable does not exist in the parent scope? Ignore.
		if v2 == nil {
			continue
		}
		// The variable has been changed in the if-clause? (the name includes the version number). If not, there is nothing to do here.
		if v1.Name == v2.Name {
			continue
		}
		if v3, ok := elseScope.vars[vo]; ok && v1.Name != v3.Name {
			// The variable has been changed in the if-clause and the else-clause
			phiVar := vs.newPhiVariable(vo)
			phiVar.Phi = append(phiVar.Phi, v1, v3)
			vs.vars[vo] = phiVar
			phiGrouping := s.createPhiGrouping(phiVar, v1, v3, vs, ifScope, elseScope)
			if phiGrouping != nil {
				// Set the phi-group-variable in the if-clause
				cmdSet0 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[0]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				closeScopeCommand := c.Block[len(c.Block)-1]
				if closeScopeCommand.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet0)
				// Set the phi-group-variable in the else-clause
				cmdSet1 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[1]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				closeScopeCommand = c.Else.Block[len(c.Else.Block)-1]
				if closeScopeCommand.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet1)
			}
		} else {
			// Variable has been changed in ifScope, but not in elseScope
			phiVar := vs.newPhiVariable(vo)
			phiVar.Phi = append(phiVar.Phi, v1, v2)
			vs.vars[vo] = phiVar
			phiGrouping := s.createPhiGrouping(phiVar, v1, v2, vs, ifScope, vs)
			if phiGrouping != nil {
				// Set the phi-group-variable before the if-clause executes
				cmdSet1 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[1]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				c.PreBlock = append(c.PreBlock, cmdSet1)
				// Set the phi-group-variable in the if-clause
				cmdSet0 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[0]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
				closeScopeCommand := c.Block[len(c.Block)-1]
				if closeScopeCommand.Op != ircode.OpCloseScope {
					panic("Oooops")
				}
				closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet0)
			}
		}
	}
	for vo, v1 := range elseScope.vars {
		// If the variable is used in the ifScope as well, ignore
		if _, ok := ifScope.vars[vo]; ok {
			continue
		}
		// The variable does not exist in the parent scope? Ignore.
		_, v3 := vs.searchVariable(vo)
		if v3 == nil {
			continue
		}
		// The variable has been changed in the else-clause? (the name includes the version number). If not, there is nothing to do here.
		if v1.Name == v3.Name {
			continue
		}
		// Variable has been changed in elseScope, but not in ifScope
		phiVar := vs.newPhiVariable(vo)
		phiVar.Phi = append(phiVar.Phi, v1, v3)
		vs.vars[vo] = phiVar
		phiGrouping := s.createPhiGrouping(phiVar, v1, v3, vs, elseScope, vs)
		if phiGrouping != nil {
			// Set the phi-group-variable before the if-clause executes
			cmdSet1 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[1]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
			c.PreBlock = append(c.PreBlock, cmdSet1)
			// Set the phi-group-variable in the else-clause
			cmdSet0 := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiGrouping.Input[0]}, Type: phiGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
			closeScopeCommand := c.Else.Block[len(c.Else.Block)-1]
			if closeScopeCommand.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet0)
		}
	}

	s.mergeScopes(vs, ifScope)
	s.mergeScopes(vs, elseScope)
}

// createLoopPhiGroupVars creates phi-group-vars for phi-groups created by a loop.
func (s *ssaTransformer) createLoopPhiGroupVars(c *ircode.Command, loopScope *ssaScope) {
	if c.Op != ircode.OpLoop {
		panic("Oooops")
	}
	if loopScope.kind != scopeLoop {
		panic("Oooops")
	}
	// Iterate over all variables that are defined in an outer scope, but used/changed inside the loop.
	// This list has been populated by `lookupVariable` with phi-variables.
	for _, phiVar := range loopScope.loopPhis {
		if phiVar.Grouping != nil {
			phiVarGrouping := phiVar.Grouping.(*Grouping)
			// Create a phi-group-variable for the phi-grouping
			t := &types.ExprType{Type: &types.PointerType{ElementType: types.PrimitiveTypeUintptr, Mode: types.PtrUnsafe}}
			v := &ircode.Variable{Kind: ircode.VarDefault, Name: phiVarGrouping.Name, Type: t, Scope: s.f.Body.Scope}
			v.Original = v
			phiVarGrouping.groupVar = v
			// s.SetGroupVariable(phiVarGrouping, v, loopScope)
			println("------>SETGV PHI ", phiVarGrouping, phiVarGrouping.Name, v.Name)
			// Add the phi-group-variable to the top-level scope of the function
			s.f.Vars = append(s.f.Vars, v)
			openScope := s.f.Body.Block[0]
			if openScope.Op != ircode.OpOpenScope {
				panic("Oooops")
			}
			cmdVar := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{v}, Type: v.Type, Location: s.f.Body.Location, Scope: s.f.Body.Scope}
			openScope.Block = append(openScope.Block, cmdVar)
			// Set the phi-group-variable before the loop starts
			cmdSet := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiVarGrouping.groupVar}, GroupArgs: []ircode.IGrouping{phiVarGrouping.Input[0]}, Type: phiVarGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
			c.PreBlock = append(c.PreBlock, cmdSet)
		}
	}
}

func (s *ssaTransformer) mergeVariablesOnContinue(c *ircode.Command, loopScope *ssaScope, continueScope *ssaScope) {
	s.mergeVariablesOnLoop(c, loopScope)

	for scope := continueScope; scope != loopScope; scope = scope.parent {
		s.mergeScopes(loopScope, scope)
	}
}

// mergeVariablesOnLoop completes the phi-groupings, which are created by `ssaScope.lookupVariable`.
// This can only be done after the entire loop body has been transformed and
// phi-group-vars have been created (see `createLoopPhiGroupVars`).
func (s *ssaTransformer) mergeVariablesOnLoop(c *ircode.Command, loopScope *ssaScope) {
	if c.Op != ircode.OpLoop && c.Op != ircode.OpContinue {
		panic("Oooops")
	}
	if loopScope.kind != scopeLoop {
		panic("Oooops")
	}
	// Iterate over all variables that are defined in an outer scope, but used/changed inside the loop.
	// This list has been populated by `lookupVariable` with phi-variables.
	for _, phiVar := range loopScope.loopPhis {
		// Determine the variable version at the end of the loop body
		v, ok := loopScope.vars[phiVar.Original]
		if !ok {
			panic("Ooooops")
		}
		// Avoid loops in the phi-dependency
		if phiVar == v {
			continue
		}
		// If the `phiVar` has not been changed inside the loop, do nothing.
		if phiVar.Name == v.Name {
			continue
		}
		// Avoid double entries in Phi
		for _, v2 := range phiVar.Phi {
			if v == v2 {
				v = nil
				break
			}
		}
		if v == nil {
			continue
		}
		// Add another phi-dependency. At loop start the value of the phi-variable
		// can be the same as the value of `v` at the end of the loop.
		phiVar.Phi = append(phiVar.Phi, v)
		if phiVar.Grouping != nil {
			// If a loop-phi-var has a grouping, it is a phi-grouping/
			// Add the grouping of v as seen at the end of the loop
			phiVarGrouping := phiVar.Grouping.(*Grouping)
			endOfLoopGrouping := loopScope.lookupGrouping(v.Grouping.(*Grouping))
			phiVarGrouping.addInput(endOfLoopGrouping)
			endOfLoopGrouping.addOutput(phiVarGrouping)
			// A real phi-grouping is required. Set the group variable before the loop starts a new iteration
			cmdSet := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{phiVarGrouping.groupVar}, GroupArgs: []ircode.IGrouping{endOfLoopGrouping}, Type: phiVarGrouping.groupVar.Type, Location: c.Location, Scope: c.Scope}
			closeScopeCommand := c.Block[len(c.Block)-1]
			if closeScopeCommand.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			closeScopeCommand.Block = append(closeScopeCommand.Block, cmdSet)
		}
	}
}

// Create phi-vars and phi-groupings resulting from breaks inside the loop.
// This can only be done after the entire loop body has been transformed and
// phi-group-vars have been created (see `createLoopPhiGroupVars`).
func (s *ssaTransformer) mergeVariablesOnBreaks(loopScope *ssaScope) {
	if loopScope.kind != scopeLoop {
		panic("Oooops")
	}
	outerScope := loopScope.parent
	if outerScope == nil {
		panic("Oooops")
	}
	for _, breakInfo := range loopScope.loopBreaks {
		// Find variables that are used in the loop after the break (when following the ir-code top to bottom).
		// Because a loop does loop, the variable might be changed nevertheless when the break executes.
		// Therefore, a phi-variable is required, too.
		for _, loopPhiVar := range loopScope.loopPhis {
			// Ignore variables which are already listed in the map
			if _, ok := breakInfo.vars[loopPhiVar]; ok {
				continue
			}
			breakInfo.vars[loopPhiVar] = loopPhiVar
		}

		for loopPhiVar, vInner := range breakInfo.vars {
			_, vOuter := outerScope.searchVariable(vInner.Original)
			// The variable does not exist in the outer scope? Ignore.
			if vOuter == nil {
				panic("Oooops")
			}
			// Create a phi-var
			breakPhiVar := outerScope.newPhiVariable(vOuter.Original)
			breakPhiVar.Phi = append(breakPhiVar.Phi, vInner, vOuter)
			outerScope.vars[breakPhiVar.Original] = breakPhiVar
			if vInner.Grouping != nil {
				loopPhiGrouping := grouping(loopPhiVar)
				// Create a phi-grouping
				breakPhiGrouping := outerScope.newPhiGrouping()
				breakPhiGrouping.Name += "_breakPhi_" + loopPhiVar.Original.Name
				breakPhiVar.Grouping = breakPhiGrouping
				// Connect the phi-grouping with the groupings used before the loop and before the break
				outerGrouping := loopPhiGrouping.Input[0]
				innerGrouping := grouping(vInner)
				breakPhiGrouping.addInput(outerGrouping)
				breakPhiGrouping.addInput(innerGrouping)
				outerGrouping.addOutput(breakPhiGrouping)
				innerGrouping.addOutput(breakPhiGrouping)
				// Hijack the phi-group-variable from the loopPhiVar
				breakPhiGrouping.groupVar = loopPhiGrouping.groupVar
				// Set the phi-group-variable before the break executes
				cmdSet := &ircode.Command{Op: ircode.OpSetGroupVariable, Dest: []*ircode.Variable{breakPhiGrouping.groupVar}, GroupArgs: []ircode.IGrouping{innerGrouping}, Type: breakPhiGrouping.groupVar.Type, Location: breakInfo.command.Location, Scope: loopScope.block.Scope}
				breakInfo.command.PreBlock = append(breakInfo.command.PreBlock, cmdSet)
			}
		}

		s.mergeScopesIntern(outerScope.groupings, breakInfo.groupings)
	}
}

// mergeVariablesOnBreak generates information that is later (after transforming the entire loop, see `mergeVariablesOnBreaks`)
// used to create phi-vars and phi-group-vars.
// `c` is the `OpBreak` command.
func (s *ssaTransformer) mergeVariablesOnBreak(c *ircode.Command, loopScope *ssaScope, breakScope *ssaScope) {
	if c.Op != ircode.OpBreak {
		panic("Oooops")
	}
	if loopScope.kind != scopeLoop {
		panic("Oooops")
	}
	outerScope := loopScope.parent
	if outerScope == nil {
		panic("Oooops")
	}
	breakVars := make(map[*ircode.Variable]*ircode.Variable)
	for _, loopPhiVar := range loopScope.loopPhis {
		_, vOuter := outerScope.searchVariable(loopPhiVar.Original)
		// The variable does not exist in the outer scope? Ignore.
		if vOuter == nil {
			panic("Oooops")
		}
		// Where in the inner scopes (top to bottom) has this variable been used?
		innerScope := breakScope
		for ; innerScope != loopScope.parent; innerScope = innerScope.parent {
			if vInner, ok := innerScope.vars[loopPhiVar.Original]; ok {
				// Variable `phiVar` has been used in this scope and is there known with version `vInner`
				// The variable has been changed in the loop? If yes, a phi-variable is required
				if vOuter.Name != vInner.Name {
					// Note that the break requires an action for `loopPhiVar` and `vInner`.
					// However, this has to be postponed until after the entire loop has been transformed.
					breakVars[loopPhiVar] = vInner
				}
				break
			}
		}
	}

	breakGroupings := make(map[*Grouping]*Grouping)
	for scope := breakScope; scope != loopScope.parent; scope = scope.parent {
		s.mergeScopesIntern(breakGroupings, scope.groupings)
	}

	loopScope.loopBreaks = append(loopScope.loopBreaks, breakInfo{vars: breakVars, groupings: breakGroupings, command: c})
}

func (s *ssaTransformer) createPhiGrouping(phiVariable, v1, v2 *ircode.Variable, phiScope, scope1, scope2 *ssaScope) *Grouping {
	if phiVariable.Grouping == nil {
		// No grouping, because the variable does not use pointers. Do nothing.
		return nil
	}

	// Determine the grouping of v1 and v2
	grouping1 := grouping(v1)
	grouping2 := grouping(v2)
	if grouping1 == nil {
		panic("Oooops, grouping1")
	}
	if grouping2 == nil {
		panic("Oooops, grouping2")
	}
	grouping1 = scope1.lookupGrouping(grouping1)
	grouping2 = scope2.lookupGrouping(grouping2)
	if grouping1 == nil {
		panic("Oooops, grouping1 after lookup")
	}
	if grouping2 == nil {
		panic("Oooops, grouping2 after lookup")
	}
	// No differences in the grouping? Then do not build a phi-grouping.
	if grouping1 == grouping2 {
		phiVariable.Grouping = grouping1
		return grouping1
	}

	// Create a phi-grouping
	phiGrouping := phiScope.newPhiGrouping()
	// TODO	phiGrouping.usedByVar = phiVariable.Original
	phiGrouping.Name += "_phi_" + phiVariable.Original.Name
	phiVariable.Grouping = phiGrouping
	// Connect the phi-grouping with the groupings of v1 and v2
	phiGrouping.addInput(grouping1)
	phiGrouping.addInput(grouping2)
	grouping1.addOutput(phiGrouping)
	grouping2.addOutput(phiGrouping)

	// Create a phi-group-variable for the phi-grouping
	t := &types.ExprType{Type: &types.PointerType{ElementType: types.PrimitiveTypeUintptr, Mode: types.PtrUnsafe}}
	v := &ircode.Variable{Kind: ircode.VarDefault, Name: phiGrouping.Name, Type: t, Scope: s.f.Body.Scope}
	v.Original = v
	phiGrouping.groupVar = v
	// Add the phi-group-variable to the top-level scope of the function
	s.f.Vars = append(s.f.Vars, v)
	openScope := s.f.Body.Block[0]
	if openScope.Op != ircode.OpOpenScope {
		panic("Oooops")
	}
	cmdVar := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{v}, Type: v.Type, Location: s.f.Body.Location, Scope: s.f.Body.Scope}
	// cmdSet := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{v}, Args: []ircode.Argument{ircode.NewVarArg(gv.In[0].GroupVariable())}, Type: v.Type, Location: block.Location, Scope: block.Scope}
	openScope.Block = append(openScope.Block, cmdVar)

	// println("Out 1. ", grouping1.Name, "->", phiGrouping.Name)
	// println("Out 2. ", grouping2.Name, "->", phiGrouping.Name)
	return phiGrouping
}

// SetGroupVariable sets the group variable on this group and on all statically merged groups as well.
func (s *ssaTransformer) SetGroupVariable(gv *Grouping, v *ircode.Variable, vs *ssaScope) {
	if gv.Original.groupVar == v {
		return
	}
	println("SETGROUPVAR", gv, gv.Name, "to", v.Name, len(gv.Input), len(gv.Output))
	if gv.Original.groupVar != nil {
		panic("Oooops, overwriting group var")
	}
	gv.Original.groupVar = v
	if gv.Kind == StaticMergeGrouping {
		for _, group := range gv.Input {
			if group.Kind == StaticMergeGrouping || group.Kind == DefaultGrouping {
				group = vs.lookupGrouping(group)
				s.SetGroupVariable(group, v, vs)
			}
		}
	}
	for _, group := range gv.Output {
		if group.Kind == StaticMergeGrouping {
			group = vs.lookupGrouping(group)
			s.SetGroupVariable(group, v, vs)
		} else if group.Kind == DynamicMergeGrouping && group.Input[0].Original == gv.Original {
			group = vs.lookupGrouping(group)
			s.SetGroupVariable(group, v, vs)
		}
	}
}

// PropagateGroupVariable ...
func (s *ssaTransformer) PropagateGroupVariable(gv *Grouping, vs *ssaScope) {
	v := gv.GroupVariable()
	if v == nil {
		panic("Oooops, no group var to propagate")
	}
	println("PROPAGATING", gv, gv.Name, "to", v.Name, len(gv.Input), len(gv.Output))
	for _, group := range gv.Output {
		if group.Kind == StaticMergeGrouping {
			group = vs.lookupGrouping(group)
			s.SetGroupVariable(group, v, vs)
		} else if group.Kind == DynamicMergeGrouping && group.Input[0].Original == gv.Original {
			group = vs.lookupGrouping(group)
			s.SetGroupVariable(group, v, vs)
		}
	}
}

// Add code to free groups and add variables for storing group pointers.
// The group pointer variables are added to the top-most scope in which memory belonging to a grouping
// could possibly be used.
func (s *ssaTransformer) transformScopes() {
	for _, scope := range s.scopes {
		for _, gv := range scope.groupings {
			// Ignore groups whose original is defined in another scope.
			// This avoid treating a group twice.
			if gv.scope != scope {
				continue
			}
			s.transformGrouping(scope, gv)
		}
	}

	for _, su := range s.stackUnwindings {
		scope := su.commandScope
		for ; scope != su.topScope.parent; scope = scope.parent {
			// Do not free variables in the loop scope in case of 'continue'.
			// The loop's iter-expression will take care of that and `continue` will jump there.
			if su.command.Op == ircode.OpContinue && scope == su.topScope {
				break
			}
			closeScopeCommand := scope.block.Block[len(scope.block.Block)-1]
			if closeScopeCommand.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			for _, c := range closeScopeCommand.Block {
				if c.Op == ircode.OpFree {
					su.command.PreBlock = append(su.command.PreBlock, c)
				}
			}
		}
	}
}

func (s *ssaTransformer) transformGrouping(scope *ssaScope, gv *Grouping) {
	if gv.Kind == PhiGrouping {
		s.PropagateGroupVariable(gv, scope)
		return
	}
	if (gv.Kind != DefaultGrouping && gv.Kind != ForeignGrouping) || gv.staticMergePoint == nil {
		return
	}
	groupVar := gv.Original.groupVar
	// Already treated?
	if groupVar != nil {
		return
	}
	vs := findTerminatingScope(gv, scope)
	openScope := vs.block.Block[0]
	if openScope.Op != ircode.OpOpenScope {
		panic("Oooops")
	}
	t := &types.ExprType{Type: types.PrimitiveTypeUintptr}
	groupVar = &ircode.Variable{Kind: ircode.VarDefault, Name: gv.Name, Type: t, Scope: vs.block.Scope}
	groupVar.Original = groupVar
	s.f.Vars = append(s.f.Vars, groupVar)
	c := &ircode.Command{Op: ircode.OpDefVariable, Dest: []*ircode.Variable{groupVar}, Type: groupVar.Type, Location: vs.block.Location, Scope: vs.block.Scope}
	c2 := &ircode.Command{Op: ircode.OpSetVariable, Dest: []*ircode.Variable{groupVar}, Args: []ircode.Argument{ircode.NewIntArg(0)}, Type: groupVar.Type, Location: vs.block.Location, Scope: vs.block.Scope}
	println("------>SETGV UNBOUND", gv.Name, groupVar.Name)
	s.SetGroupVariable(gv, groupVar, scope)

	// Add the variable definition and its assignment to openScope of the block
	openScope.Block = append(openScope.Block, c, c2)

	if !gv.IsDefinitelyUnavailable() {
		c := &ircode.Command{Op: ircode.OpFree, Args: []ircode.Argument{ircode.NewVarArg(groupVar)}, Location: vs.block.Location, Scope: vs.block.Scope}
		closeScope := vs.block.Block[len(vs.block.Block)-1]
		if closeScope.Op != ircode.OpCloseScope {
			panic("Oooops")
		}
		closeScope.Block = append(closeScope.Block, c)
	}
}

// Searches the top-most scope in which a group variable (or one of its dependent groups) is used.
// This is the scope where a group can be savely free'd.
func findTerminatingScope(grouping *Grouping, vs *ssaScope) *ssaScope {
	if len(grouping.Output) == 0 || grouping.marked {
		return vs
	}
	grouping.marked = true
	p := vs
	for _, out := range grouping.Output {
		outScope := findTerminatingScope(out, out.scope)
		if p.hasParent(outScope) {
			p = outScope
		}
	}
	grouping.marked = false
	return p
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
		paramGrouping := s.topLevelScope.newGroupingFromSpecifier(g)
		paramGrouping.groupVar = v
	}
	// Mark all input parameters as initialized.
	for _, v := range f.InVars {
		// Parameters are always initialized upon function invcation.
		v.IsInitialized = true
	}
	// Add a grouping to the function scope of the ircode.
	f.Body.Scope.Grouping = s.topLevelScope.newScopedGrouping(f.Body.Scope)
	s.transformBlock(&f.Body, s.topLevelScope)
	s.transformScopes()
}
