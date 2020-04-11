package ssa

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

type scopeKind int

const (
	scopeIf scopeKind = 1 + iota
	scopeLoop
	scopeFunc
)

type breakInfo struct {
	vars map[*ircode.Variable]*ircode.Variable
	// groupings map[*Grouping]*Grouping
	command *ircode.Command
}

type ssaScope struct {
	s      *ssaTransformer
	block  *ircode.Command
	parent *ssaScope
	// A lexical scope can be decomposed into multiple ssaScopes, e.g. in
	// ```
	// if a { foo(); return;} bar()
	// ```
	// the lexical scope containing `if` and `bar` is split into two ssaScops.
	// The rational is that control flows through all instructions of a scope once the scope has been entered.
	// This is not the case in the example above.
	// The first ssaScope is the canonical ssaScope.
	canonicalSiblingScope *ssaScope
	// See `canonicalSignal`. Pointer to the next ssaScope of the same lexical scope pr nil.
	nextSiblingScope *ssaScope
	// Maps the `Original` of a groupings to version if the grouping that is used in this scope.
	groupings map[*Grouping]*Grouping
	// Maps the original variable to the latest variable version used in this scope.
	// All variables used or modified in this scope are listed here.
	// Variables that are just used but not modified in this scope have the same version (and hence Name)
	// as the corresponding ircode.Variable in the parent scope.
	vars map[*ircode.Variable]*ircode.Variable
	// Only used for scopes where `kind == scopeLoop`.
	// List of variables which are imported from outside the loop and used/changed inside the loop.
	loopPhis []*ircode.Variable
	// Only used for scopes where `kind == scopeLoop`.
	// Each `breakInfo` results from an OpBreak. It tells which original variables must be updated via a phi-variable
	// because of the break.
	loopBreaks []breakInfo
	// continueCount int
	breakCount int
	// Used by scopes that unconditionally end with `OpBreak` or `OpReturn`.
	// In this case the variable lists the scopes which are left by the break or return.
	// In case the control flow can enter different `OpBreak` or `OpReturn` instructions, e.g.
	// ```
	// if a { break loop1; } else { break loop2; }
	// ```
	// the variable lists those scopes that are left in all possible control flows.
	// The `exitScope` defines from which point in the scope this exit will be taken.
	// For example in
	// ```
	// if a { break innerLoop; } foo(); return
	// ```
	// the scope breaks at least `innerLoop`. Once control flow reaches the function call to `foo`,
	// it will exit all scopes.
	exitScopes []*ssaScope
	kind       scopeKind
	// Used to generate error messages that point to the loop
	loopLocation errlog.LocationRange
	// In the case of
	// ```
	// if a { foo(); return; } bar()
	// ```
	// the scope calling `foo` is an alternative to this scope from the step on where `bar` is being called.
	// In other words: Only one of both an happen, either `foo(); return` or `bar()`.
	alternativeScopes []*ssaScope
	marker            int
}

func newScope(s *ssaTransformer, block *ircode.Command, parent *ssaScope) *ssaScope {
	if parent != nil {
		parent = parent.canonicalSiblingScope
	}
	scope := &ssaScope{block: block, s: s, groupings: make(map[*Grouping]*Grouping), vars: make(map[*ircode.Variable]*ircode.Variable), parent: parent}
	s.scopes = append(s.scopes, scope)
	scope.canonicalSiblingScope = scope
	return scope
}

func newSiblingScope(sibling *ssaScope) *ssaScope {
	if sibling.nextSiblingScope != nil {
		panic("Oooops")
	}
	scope := &ssaScope{kind: sibling.kind, loopLocation: sibling.loopLocation, block: sibling.block, s: sibling.s, groupings: sibling.groupings, vars: sibling.vars, parent: sibling.parent, canonicalSiblingScope: sibling.canonicalSiblingScope}
	sibling.nextSiblingScope = scope
	return scope
}

func (vs *ssaScope) lastSibling() *ssaScope {
	for ; vs.nextSiblingScope != nil; vs = vs.nextSiblingScope {
		// Do nothing by intention
	}
	return vs
}

func (vs *ssaScope) lookupGrouping(gv *Grouping) *Grouping {
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.groupings[gv.Original]; ok {
			return gv2
		}
	}
	panic("Oooops")
}

func (vs *ssaScope) searchVariable(v *ircode.Variable) (*ssaScope, *ircode.Variable) {
	if v2, ok := vs.vars[v.Original]; ok {
		return vs, v2
	}
	for p := vs; p != nil; p = p.parent {
		if v2, ok := p.vars[v.Original]; ok {
			return p, v2
		}
	}
	return nil, nil
}

func (vs *ssaScope) lookupVariable(v *ircode.Variable) (*ssaScope, *ircode.Variable) {
	if v2, ok := vs.vars[v.Original]; ok {
		return vs, v2
	}
	if vs.parent != nil {
		vs2, v2 := vs.parent.lookupVariable(v)
		if v2 != nil {
			if vs.kind == scopeLoop {
				loopPhiVar := vs.newPhiVariable(v2)
				vs.createLoopPhiGrouping(loopPhiVar, v2, vs.loopLocation)
				vs.canonicalSiblingScope.loopPhis = append(vs.canonicalSiblingScope.loopPhis, loopPhiVar)
				return vs, loopPhiVar
			} else if vs.kind == scopeIf {
				v2 = vs.newVariableUsageVersion(v2)
				vs.vars[v.Original] = v2
			}
			return vs2, v2
		}
	}
	return nil, nil
}

func (vs *ssaScope) newVariableVersion(v *ircode.Variable) *ircode.Variable {
	vo := v.Original
	vo.VersionCount++
	name := vo.Name + "." + strconv.Itoa(vo.VersionCount)
	v2 := &ircode.Variable{Kind: vo.Kind, Name: name, Type: vo.Type, Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, Grouping: v.Grouping}
	vs.vars[vo] = v2
	// TODO: Remove constant values from the type
	return v2
}

func (vs *ssaScope) newVariableUsageVersion(v *ircode.Variable) *ircode.Variable {
	vo := v.Original
	v2 := &ircode.Variable{Kind: vo.Kind, Name: v.Name, Type: vo.Type, Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, Grouping: v.Grouping}
	vs.vars[vo] = v2
	return v2
}

func (vs *ssaScope) newPhiVariable(v *ircode.Variable) *ircode.Variable {
	vo := v.Original
	vo.VersionCount++
	name := vo.Name + ".phi" + strconv.Itoa(vo.VersionCount)
	v2 := &ircode.Variable{Kind: ircode.VarPhi, Name: name, Phi: []*ircode.Variable{v}, Type: types.CloneExprType(vo.Type), Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, Grouping: v.Grouping}
	vs.vars[vo] = v2
	return v2
}

func (vs *ssaScope) defineVariable(v *ircode.Variable) {
	vs.vars[v] = v
}

func (vs *ssaScope) createDestinationVariable(c *ircode.Command) *ircode.Variable {
	if len(c.Dest) == 0 || c.Dest[0] == nil {
		return nil
	}
	if len(c.Dest) == 0 {
		panic("Ooooops")
	}
	// Create a new version of the destination variable when required
	var v = c.Dest[0]
	if _, v2 := vs.lookupVariable(v.Original); v2 != nil {
		// If the variable has been defined or assigned so far, create a new version of it.
		v = vs.newVariableVersion(v.Original)
		c.Dest[0] = v
	} else {
		vs.defineVariable(v)
	}
	return v
}

func (vs *ssaScope) createDestinationVariableByIndex(c *ircode.Command, index int) *ircode.Variable {
	if c.Dest[index] == nil {
		return nil
	}
	// Create a new version of the destination variable when required
	var v = c.Dest[index]
	if _, v2 := vs.lookupVariable(v.Original); v2 != nil {
		// If the variable has been defined or assigned so far, create a new version of it.
		v = vs.newVariableVersion(v.Original)
		c.Dest[index] = v
	} else {
		vs.defineVariable(v)
	}
	return v
}

func (vs *ssaScope) funcScope() *ssaScope {
	for {
		if vs.kind == scopeFunc {
			return vs
		}
		vs = vs.parent
	}
}

var groupCounter = 0

func (vs *ssaScope) newDefaultGrouping(loc errlog.LocationRange) *Grouping {
	gname := "g_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: DefaultGrouping, Name: gname, staticMergePoint: &groupingAllocationPoint{scope: vs, step: vs.s.step}, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newConstantGrouping(loc errlog.LocationRange) *Grouping {
	gname := "gc_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: ConstantGrouping, Name: gname, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newGroupingFromSpecifier(groupSpec *types.GroupSpecifier) *Grouping {
	if grouping, ok := vs.s.parameterGroupings[groupSpec]; ok {
		return grouping
	}
	gname := "gn_" + groupSpec.Name
	groupCounter++
	var grouping *Grouping
	if groupSpec.Kind == types.GroupSpecifierIsolate {
		grouping = &Grouping{Kind: DefaultGrouping, Name: gname, scope: vs.funcScope()}
	} else {
		grouping = &Grouping{Kind: ParameterGrouping, Name: gname, scope: vs.funcScope()}
		grouping.Name = groupSpec.Name
		grouping.Constraint.NamedGroup = groupSpec.Name
	}
	grouping.Original = grouping
	grouping.Location = groupSpec.Location
	vs.s.parameterGroupings[groupSpec] = grouping
	vs.groupings[grouping] = grouping
	return grouping
}

func (vs *ssaScope) newScopedGrouping(lexicalScope *ircode.CommandScope, loc errlog.LocationRange) *Grouping {
	if gv, ok := vs.s.scopedGroupings[lexicalScope]; ok {
		return gv
	}
	gname := "gs_" + strconv.Itoa(lexicalScope.ID)
	gv := &Grouping{Kind: ScopedGrouping, Name: gname, lexicalScope: lexicalScope, scope: vs}
	gv.Original = gv
	gv.Location = loc
	gv.Constraint.Scope = lexicalScope
	vs.groupings[gv] = gv
	vs.s.scopedGroupings[lexicalScope] = gv
	return gv
}

func (vs *ssaScope) newViaGrouping(via *Grouping, loc errlog.LocationRange) *Grouping {
	gname := "g_" + strconv.Itoa(groupCounter) + "_via_" + via.GroupingName()
	groupCounter++
	gv := &Grouping{Kind: ForeignGrouping, Name: gname, scope: vs}
	gv.Input = append(gv.Input, via)
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newBranchPhiGrouping(loc errlog.LocationRange) *Grouping {
	gname := "gpb_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: BranchPhiGrouping, Name: gname, phiAllocationPoint: &groupingAllocationPoint{scope: vs, step: vs.s.step}, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newLoopPhiGrouping(loc errlog.LocationRange) *Grouping {
	gname := "gpl_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: LoopPhiGrouping, Name: gname, phiAllocationPoint: &groupingAllocationPoint{scope: vs, step: vs.s.step}, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newStaticMergeGrouping(loc errlog.LocationRange) *Grouping {
	gname := "gsm_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: StaticMergeGrouping, Name: gname, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newDynamicMergeGrouping(loc errlog.LocationRange) *Grouping {
	gname := "gdm_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Kind: DynamicMergeGrouping, Name: gname, scope: vs}
	gv.Original = gv
	gv.Location = loc
	vs.groupings[gv] = gv
	return gv
}

func (vs *ssaScope) newGroupingVersion(original *Grouping) *Grouping {
	gv := &Grouping{Kind: original.Kind, Name: original.Name, Allocations: original.Allocations, Original: original.Original, groupVar: original.groupVar, unavailable: original.unavailable, lexicalScope: original.lexicalScope, staticMergePoint: original.staticMergePoint, phiAllocationPoint: original.phiAllocationPoint, scope: vs, Location: original.Location}
	l := len(original.Input)
	if l > 0 {
		gv.Input = make([]*Grouping, l, l+1)
		copy(gv.Input, original.Input)
	}
	l = len(original.Output)
	if l > 0 {
		gv.Output = make([]*Grouping, l, l+1)
		copy(gv.Output, original.Output)
	}
	vs.groupings[original] = gv
	return gv
}

func (vs *ssaScope) newUnavailableGroupingVersion(original *Grouping) *Grouping {
	group := vs.newGroupingVersion(original)
	group.unavailable = true
	return group
}

func (vs *ssaScope) hasParent(p *ssaScope) bool {
	p = p.canonicalSiblingScope
	for x := vs.parent; x != nil; x = x.parent {
		if p == x {
			return true
		}
	}
	return false
}

// func (vs *ssaScope) mergeVariablesOnContinue(c *ircode.Command, continueScope *ssaScope, log *errlog.ErrorLog) {
//	panic("TODO")
/*
	if vs.kind != scopeLoop {
		panic("Oooops")
	}
	// Iterate over all variables that are defined in an outer scope, but used/changed inside the loop
	for _, phi := range vs.loopPhis {
		var phiGroup *Grouping
		if phi.Grouping != nil {
			phiGroup = vs.lookupGrouping(phi.Grouping.(*Grouping))
		}
		// Iterate over all scopes starting at the scope of the continue down to (and including)
		// the scope of the loop.
		vs2 := continueScope
		for ; vs2 != vs.parent; vs2 = vs2.parent {
			// The phi-variable is used in this scope?
			v, ok := vs2.vars[phi.Original]
			if ok {
				// Avoid double entries in Phi
				for _, v2 := range phi.Phi {
					if v == v2 {
						v = nil
						break
					}
				}
				if v != nil {
					phi.Phi = append(phi.Phi, v)
					if phiGroup != nil {
						g := vs2.lookupGrouping(v.Grouping.(*Grouping))
						newGroup, doMerge := vs.merge(phiGroup, g, nil, c, log)
						// println("CONT MERGE", newGroup.GroupingName(), "=", phiGroup.GroupingName(), g.GroupingName())
						if doMerge {
							cmdMerge := &ircode.Command{Op: ircode.OpMerge, GroupArgs: []ircode.IGrouping{phiGroup, g}, Type: &types.ExprType{Type: types.PrimitiveTypeVoid}, Location: c.Location, Scope: c.Scope}
							c.PreBlock = append(c.PreBlock, cmdMerge)
						}
						// phi.Grouping = newGroup
						phiGroup = newGroup
					}
				}
				break
			}
		}
	} */
// }

func (vs *ssaScope) mergeVariables(childScope *ssaScope) {
	for vo, v := range childScope.vars {
		vs.vars[vo] = v
	}
}

func (vs *ssaScope) createLoopPhiGrouping(loopPhiVar, outerVar *ircode.Variable, loc errlog.LocationRange) *Grouping {
	loopScope := vs
	if loopScope.kind != scopeLoop {
		panic("Oooops")
	}
	outerScope := loopScope.parent
	if outerScope == nil {
		panic("Oooops")
	}

	if loopPhiVar.Grouping == nil {
		// No grouping, because the variable does not use pointers. Do nothing.
		return nil
	}

	// Determine the grouping of v1 and v2
	outerGrouping := grouping(outerVar)
	if outerGrouping == nil {
		panic("Oooops, outerGrouping")
	}
	outerGrouping = outerScope.lookupGrouping(outerGrouping)
	if outerGrouping == nil {
		panic("Oooops, grouping2 after lookup")
	}

	// Create a phi-grouping
	phiGrouping := outerScope.newLoopPhiGrouping(loc)
	phiGrouping.loopScope = loopScope
	phiGrouping.Name += "_loopPhi_" + loopPhiVar.Original.Name
	loopPhiVar.Grouping = phiGrouping
	// Connect the phi-grouping with the groupings of v1 and v2
	phiGrouping.addInput(outerGrouping)
	outerGrouping.addOutput(phiGrouping)
	return phiGrouping
}

func scopeListIntersection(l1, l2 []*ssaScope) []*ssaScope {
	var result []*ssaScope
	for _, s1 := range l1 {
		for _, s2 := range l2 {
			if s1 == s2 {
				result = append(result, s1)
				break
			}
		}
	}
	return result
}

func sameLexicalScope(s1, s2 *ssaScope) bool {
	return s1.canonicalSiblingScope == s2.canonicalSiblingScope
}
