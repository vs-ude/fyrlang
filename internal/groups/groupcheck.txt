package group

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type groupAnnotater struct {
	f     *ircode.Function
	log   *errlog.ErrorLog
	stack []*groupScope
}

type groupScope struct {
	// A group-update means that a group has been merged with another group, resulting in a new group
	// that should be used from thereon.
	// This map stores the resulting group as value.
	groupUpdates map[*types.Group]*types.Group
	loopBreak    bool
	loopContinue bool
	targetCount  int
}

func newGroupScope() *groupScope {
	return &groupScope{groupUpdates: make(map[*types.Group]*types.Group)}
}

func (ga *groupAnnotater) setGroupUpdate(oldGroup *types.Group, newGroup *types.Group) {
	ga.stack[len(ga.stack)-1].groupUpdates[oldGroup] = newGroup
}

// `search` can be nil. In this case the function searches for the group in the entire scope stack.
func (ga *groupAnnotater) lookupGroupUpdate(g *types.Group, search []*groupScope) *types.Group {
	if search == nil {
		search = ga.stack
	}
	// Search through the stack for group updates
	updated := true
	for updated {
		updated = false
		for i := len(search) - 1; i >= 0; i-- {
			if search[i].loopBreak {
				continue
			}
			if g2, ok := search[i].groupUpdates[g]; ok {
				g = g2
				updated = true
				break
			}
		}
	}
	return g
}

// An optional code flow comes from `src` and enters `dest`.
// For example, in `if (cond) { src }` the code flow can enter `src` optionally, depending on the condition `cond`.
// Therefore mergeOptionalScope updates all groups in `dest` assuming that `src` has perhaps been executed.
func (ga *groupAnnotater) mergeOptionalScope(dest *groupScope, src *groupScope, loc errlog.LocationRange) {
	for g, update := range src.groupUpdates {
		for {
			if update2, ok := src.groupUpdates[update]; ok {
				update = update2
				continue
			}
			break
		}
		dest.groupUpdates[g] = types.NewPhiGroup([]*types.Group{g, update}, loc)
	}
}

func (ga *groupAnnotater) mergeBreakScopes(dest *groupScope, src []*groupScope, loc errlog.LocationRange) {
	if !dest.loopBreak {
		panic("Not a break dest scope")
	}
	var tmp *groupScope
	if dest.targetCount == 0 {
		tmp = dest
	} else {
		tmp = newGroupScope()
	}
	for _, s := range src {
		if s.loopBreak {
			continue
		}
		for g, update := range s.groupUpdates {
			tmp.groupUpdates[g] = ga.lookupGroupUpdate(update, src)
		}
	}
	if tmp != dest {
		ga.mergeOptionalScope(dest, tmp, loc)
	}
}

func (ga *groupAnnotater) createContinueScope(loc errlog.LocationRange) *groupScope {
	s := newGroupScope()
	for _, s := range ga.stack {
		if s.loopBreak {
			continue
		}
		for g, update := range s.groupUpdates {
			s.groupUpdates[g] = types.NewPhiGroup([]*types.Group{ga.lookupGroupUpdate(update, ga.stack)}, loc)
		}
	}
	return s
}

func (ga *groupAnnotater) mergeContinueScopes(dest *groupScope, src []*groupScope, loc errlog.LocationRange) {
	if !dest.loopContinue {
		panic("Not a continue dest scope")
	}
	tmp := newGroupScope()
	for _, s := range src {
		if s.loopBreak {
			continue
		}
		for g, update := range s.groupUpdates {
			tmp.groupUpdates[g] = ga.lookupGroupUpdate(update, src)
		}
	}
	for g, update := range tmp.groupUpdates {
		if phi, ok := dest.groupUpdates[g]; ok {
			phi.Groups = append(phi.Groups, update)
		}
	}
}

func (ga *groupAnnotater) annotateFunction(f *ircode.Function) {
	ga.stack = []*groupScope{newGroupScope()}
	ga.annotateBlock(&f.Body)
}

func (ga *groupAnnotater) annotateBlock(c *ircode.Command) bool {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		if !ga.annotateCommand(c2) {
			return false
		}
	}
	return true
}

// Returns true if there is a code flow that passes through this command.
// A counter example is a loop that does never return, or a break statement,
// because in this case the subsequent commands are not executed.
func (ga *groupAnnotater) annotateCommand(c *ircode.Command) bool {
	switch c.Op {
	case ircode.OpBlock:
		ga.annotateBlock(c)
	case ircode.OpIf:
		ga.annotateArguments(c)
		// Annotate the if-clause
		s := newGroupScope()
		ga.stack = append(ga.stack, s)
		ifCompletes := ga.annotateBlock(c)
		ga.stack = ga.stack[:len(ga.stack)-1]
		// Annotate the else-clause
		if c.Else != nil {
			s2 := newGroupScope()
			ga.stack = append(ga.stack, s2)
			elseCompletes := ga.annotateBlock(c.Else)
			ga.stack = ga.stack[:len(ga.stack)-1]
			if ifCompletes && elseCompletes {
				// TODO ga.mergeAlternativeScopes(ga.stack[len(ga.stack)-1], s, s2, c.Location)
			} else if ifCompletes {
				ga.mergeOptionalScope(ga.stack[len(ga.stack)-1], s, c.Location)
			} else if elseCompletes {
				ga.mergeOptionalScope(ga.stack[len(ga.stack)-1], s2, c.Location)
			}
			return ifCompletes || elseCompletes
		} else if ifCompletes {
			ga.mergeOptionalScope(ga.stack[len(ga.stack)-1], s, c.Location)
		}
	case ircode.OpLoop:
		breakScope := newGroupScope()
		breakScope.loopBreak = true
		ga.stack = append(ga.stack, breakScope)
		continueScope := ga.createContinueScope(c.Location)
		continueScope.loopContinue = true
		ga.stack = append(ga.stack, continueScope)
		s := newGroupScope()
		ga.stack = append(ga.stack, s)
		doesLoop := ga.annotateBlock(c)
		ga.stack = ga.stack[:len(ga.stack)-3]
		if doesLoop || continueScope.targetCount > 0 {
			ga.mergeContinueScopes(continueScope, []*groupScope{s}, c.Location)
		}
		ga.mergeOptionalScope(ga.stack[len(ga.stack)-1], breakScope, c.Location)
		if breakScope.targetCount == 0 {
			return false
		}
		return true
	case ircode.OpBreak:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var i int
		for i = len(ga.stack) - 1; i >= 0; i-- {
			if ga.stack[i].loopBreak {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopDepth != 0 {
			panic("Could not find matching loop")
		}
		ga.stack[i].targetCount++
		ga.mergeBreakScopes(ga.stack[i], ga.stack[i+1:], c.Location)
		return false
	case ircode.OpContinue:
		loopDepth := int(c.Args[0].Const.ExprType.IntegerValue.Uint64()) + 1
		var i int
		for i = len(ga.stack) - 1; i >= 0; i-- {
			if ga.stack[i].loopContinue {
				loopDepth--
				if loopDepth == 0 {
					break
				}
			}
		}
		if loopDepth != 0 {
			panic("Could not find matching continue")
		}
		ga.stack[i].targetCount++
		ga.mergeContinueScopes(ga.stack[i], ga.stack[i+1:], c.Location)
		return false
	case ircode.OpDefVariable:
		if len(c.Dest) != 1 {
			panic("Ooooops")
		}
		c.Dest[0].Var.Group = types.NewScopedGroup(c.Dest[0].Var.Scope, c.Location)
		if c.Dest[0].Var.Kind == ircode.VarParameter {
			// The groups of parameters are defined by the function signature.
			// Thus, we can use them here directly.
			c.Dest[0].Var.PointerDestGroup = c.Dest[0].Var.Type.PointerDestGroup
		} else {
			// The variable is declared here but not initialized.
			// Let's start with the idea that all its pointers do point to a free group.
			// This can change once the variable
			c.Dest[0].Var.PointerDestGroup = types.NewFreeGroup(c.Location)
		}
		c.Dest[0].Group = c.Dest[0].Var.Group
		c.Dest[0].PointerDestGroup = c.Dest[0].Var.PointerDestGroup
	case ircode.OpPrintln,
		ircode.OpAdd,
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
		ircode.OpBitwiseComplement:

		if len(c.Dest) != 1 {
			panic("Oooops")
		}
		ga.annotateArguments(c)
		ga.initializeTempVar(c.Dest[0].Var, c.Location)
		// The result of these operations has no pointers.
		// Hence we can use a free group here.
		ptrDestGroup := types.NewFreeGroup(c.Location)
		ga.setGroupUpdate(c.Dest[0].Var.PointerDestGroup, ptrDestGroup)
		c.Dest[0].Group = c.Dest[0].Var.Group
		c.Dest[0].PointerDestGroup = ptrDestGroup
	case ircode.OpSetVariable:
		if len(c.Dest) != 1 {
			panic("Oooops")
		}
		ga.annotateArguments(c)
		ga.initializeTempVar(c.Dest[0].Var, c.Location)
		// Determine the group to which pointers of the RHS are pointing
		ptrDestGroup := ga.argumentGroup(c.Args[0])
		// Take note that the LHS (pointers stored in the assigned variable) do now point to the same group as the RHS.
		ga.setGroupUpdate(c.Dest[0].Var.PointerDestGroup, ptrDestGroup)
		c.Dest[0].Group = c.Dest[0].Var.Group
		c.Dest[0].PointerDestGroup = ptrDestGroup
	case ircode.OpSet:
		if len(c.Dest) != 1 {
			panic("Oooops")
		}
		ga.annotateArguments(c)
		ga.initializeTempVar(c.Dest[0].Var, c.Location)
		// Get the group to which values of the RHS are pointing
		rhsPtrDestGroup := ga.argumentGroup(c.Args[len(c.Args)-1])
		lhsPtrDestGroup := ga.accessChainGroup(c)
		if isPureValueType(c.Type.Type) {
			// Do nothing
		} else if rhsPtrDestGroup.Kind == types.GroupIsolate {
			// Do nothing, because the groups are still disjoined.
		} else {
			ptrDestGroup := types.NewGammaGroup([]*types.Group{lhsPtrDestGroup, rhsPtrDestGroup}, c.Location)
			ga.setGroupUpdate(lhsPtrDestGroup, ptrDestGroup)
			ga.setGroupUpdate(rhsPtrDestGroup, ptrDestGroup)
			c.Dest[0].PointerDestGroup = ptrDestGroup
		}
		c.Dest[0].Group = c.Dest[0].Var.Group
	case ircode.OpGet:
		if len(c.Dest) != 1 {
			panic("Oooops")
		}
		ga.annotateArguments(c)
		ga.initializeTempVar(c.Dest[0].Var, c.Location)
		ptrDestGroup := ga.accessChainGroup(c)
		ga.setGroupUpdate(c.Dest[0].Var.PointerDestGroup, ptrDestGroup)
		c.Dest[0].Group = c.Dest[0].Var.Group
		c.Dest[0].PointerDestGroup = ptrDestGroup
	case ircode.OpStruct:
		// TODO Merge them all except for the isolates
	case ircode.OpArray:
		if len(c.Dest) != 1 {
			panic("Oooops")
		}
		ga.annotateArguments(c)
		ga.initializeTempVar(c.Dest[0].Var, c.Location)
		// TODO: Merge them all unless the values of the array are isolates.
	default:
		panic("TODO")
	}
	return true
}

func (ga *groupAnnotater) initializeTempVar(v *ircode.Variable, loc errlog.LocationRange) {
	// This is only required for temporary variables, because all others are properly defined before they are assigned.
	if v.Kind == ircode.VarTemporary {
		v.Group = types.NewScopedGroup(v.Scope, loc)
		v.PointerDestGroup = types.NewFreeGroup(loc)
	}
}

func (ga *groupAnnotater) annotateArguments(c *ircode.Command) {
	for i := range c.Args {
		if c.Args[i].Cmd != nil {
			panic("No nested commands")
			//			ga.annotateCommand(c.Args[i].Cmd)
		}
		if c.Args[i].Var.Var != nil {
			c.Args[i].Var.Group = ga.lookupGroupUpdate(c.Args[i].Var.Group, nil)
		}
	}
}

func (ga *groupAnnotater) argumentGroup(a ircode.Argument) *types.Group {
	if a.Var.Var != nil {
		return ga.lookupGroupUpdate(a.Var.Group, nil)
	}
	if a.Const != nil {
		return types.NewFreeGroup(a.Location)
	}
	panic("Argument should not be a command")
}

func (ga *groupAnnotater) accessChainGroup(c *ircode.Command) *types.Group {
	// Shortcut in case the result of the access chain carries no pointers at all.
	if isPureValueType(c.Type.Type) {
		return types.NewFreeGroup(c.Location)
	}
	if len(c.AccessChain) == 0 {
		panic("No access chain")
	}
	if c.Args[0].Var.Var == nil {
		panic("Access chain is not accessing a variable")
	}
	if c.Args[0].Var.PointerDestGroup == nil || c.Args[0].Var.Group == nil {
		panic("Access chain is accessing a variable that has no group")
	}
	// The variable on which this access chain starts is stored as local variable in a scope.
	// Thus, the group of this value is a scoped group.
	valueGroup := c.Args[0].Var.Group
	// The variable on which this access chain starts might have pointers.
	// Determine to group to which these pointers are pointing.
	ptrDestGroup := c.Args[0].Var.PointerDestGroup
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
				return types.NewFreeGroup(ac.Location)
			}
			if ptrDestGroup.Kind == types.GroupIsolate {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value is in turn an isolate pointer. But that is ok, since the type system has this information in form of a GroupType.
				ptrDestGroup = valueGroup
			} else {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into a gamma-group.
				ptrDestGroup = types.NewGammaGroup([]*types.Group{valueGroup, ptrDestGroup}, ac.Location)
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = types.NewScopedGroup(c.Scope, ac.Location)
		case ircode.AccessStruct:
			_, ok := types.GetStructType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = types.NewUniquelyNamedGroup(ac.Location)
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
				return types.NewFreeGroup(ac.Location)
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = types.NewUniquelyNamedGroup(ac.Location)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = types.NewUniquelyNamedGroup(ac.Location)
			}
		case ircode.AccessSliceIndex:
			_, ok := types.GetSliceType(ac.InputType.Type)
			if !ok {
				panic("Not a slice")
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = types.NewUniquelyNamedGroup(ac.Location)
			}
		case ircode.AccessDereferencePointer:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return types.NewFreeGroup(ac.Location)
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = types.NewUniquelyNamedGroup(ac.Location)
			}
		case ircode.AccessSlice:
			// ... TODO
			panic("TODO")
		default:
			panic("TODO")
		}
	}
	return ptrDestGroup
}

func (ga *groupAnnotater) computeBlock(c *ircode.Command) {
	if c.Op != ircode.OpBlock && c.Op != ircode.OpIf && c.Op != ircode.OpLoop {
		panic("Not a block")
	}
	for _, c2 := range c.Block {
		ga.computeCommand(c2)
	}
}

func (ga *groupAnnotater) computeCommand(c *ircode.Command) {
	switch c.Op {
	case ircode.OpBlock:
		ga.computeBlock(c)
	case ircode.OpIf:
		// Compute the condition
		ga.computeArguments(c)
		// Compute the if-clause
		ga.computeBlock(c)
		// Compute the else-clause
		if c.Else != nil {
			ga.computeBlock(c.Else)
		}
	case ircode.OpLoop:
		ga.computeBlock(c)
	case ircode.OpBreak, ircode.OpContinue, ircode.OpDefVariable:
		// Do nothing by intention
	case ircode.OpGet:
		ga.computeArguments(c)
	case ircode.OpSetVariable, ircode.OpSet:
		ga.computeArguments(c)
		ga.computeDests(c)
	case ircode.OpAdd,
		ircode.OpPrintln,
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
		ircode.OpBitwiseComplement:

		// Do nothing by intention
	default:
		panic("TODO")
	}
}

func (ga *groupAnnotater) computeArguments(c *ircode.Command) {
	// Evaluate arguments right to left
	for i := len(c.Args) - 1; i >= 0; i-- {
		if c.Args[i].Cmd != nil {
			panic("Should not happen")
		}
		if c.Args[i].Var.Var != nil {
			if c.Args[i].Var.PointerDestGroup.Kind == types.GroupInvalid {
				ga.log.AddError(errlog.ErrorUninitializedVariable, c.Location, c.Args[i].Var.Var.Original.Name)
				continue
			}
			ga.computeVariableUsage(&c.Args[i].Var)
		}
	}
}

func (ga *groupAnnotater) computeDests(c *ircode.Command) {
	for i := range c.Dest {
		ga.computeVariableUsage(&c.Dest[i])
	}
}

func (ga *groupAnnotater) computeVariableUsage(vu *ircode.VariableUsage) {
	vu.PointerDestGroup.Compute(ga.log)
}

// AnnotateGroups ...
func AnnotateGroups(f *ircode.Function, log *errlog.ErrorLog) {
	ga := &groupAnnotater{f: f, log: log}
	// Annotate all variables usages with groups
	ga.annotateFunction(f)
	// Try to compute all groups.
	// This can lead to compiler errors, which are sent to the ErrorLog.
	//	ga.computeBlock(&f.Body)
}

func isPureValueType(t types.Type) bool {
	if types.IsIntegerType(t) {
		return true
	}
	if types.IsFloatType(t) {
		return true
	}
	if t == types.PrimitiveTypeBool {
		return true
	}
	switch t2 := t.(type) {
	case *types.AliasType:
		return isPureValueType(t2.Alias)
	case *types.MutableType:
		return isPureValueType(t2.Type)
	case *types.GroupType:
		return isPureValueType(t2.Type)
	case *types.StructType:
		for _, f := range t2.Fields {
			if !isPureValueType(f.Type) {
				return false
			}
		}
		return true
	}
	return false
}
