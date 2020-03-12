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
	vars    map[*ircode.Variable]*ircode.Variable
	command *ircode.Command
}

type ssaScope struct {
	s      *ssaTransformer
	block  *ircode.Command
	parent *ssaScope
	// Maps Groupings to the Grouping that merged them.
	// Yet unmerged Groupings are listed here, too, and in this case key and value in the map are equal.
	staticGroupings  map[*Grouping]*Grouping
	dynamicGroupings map[*Grouping]*Grouping
	// Maps the original variable to the latest variable version used in this scope.
	// All variables used or modified in this scope are listed here.
	// Variables that are just used but not modified in this scope have the same version (and hence Name)
	// as the corresponding ircode.Variable in the parent scope.
	vars map[*ircode.Variable]*ircode.Variable
	// Only used for scopes where `kind == scopeLoop`.
	// List of variables which are imported from outside the loop and used/changed inside the loop.
	loopPhis []*ircode.Variable
	// Only used for scopes where `kind == scopeLoop`.
	// Each map results from an OpBreak. It tells which original variables must be updated via a phi-variable
	// because of the break.
	loopBreaks    []breakInfo
	continueCount int
	breakCount    int
	kind          scopeKind
}

func newScope(s *ssaTransformer, block *ircode.Command) *ssaScope {
	scope := &ssaScope{block: block, s: s, staticGroupings: make(map[*Grouping]*Grouping), dynamicGroupings: make(map[*Grouping]*Grouping), vars: make(map[*ircode.Variable]*ircode.Variable)}
	s.scopes = append(s.scopes, scope)
	return scope
}

func (vs *ssaScope) lookupGrouping(gv *Grouping) *Grouping {
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.dynamicGroupings[gv]; ok {
			if p != vs {
				gv2.Close()
			}
			return gv2
		}
		if gv2, ok := p.staticGroupings[gv]; ok {
			if p != vs {
				gv2.Close()
			}
			return gv2
		}
	}
	panic("Oooops")
}

func (vs *ssaScope) searchGrouping(gv *Grouping) *Grouping {
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.dynamicGroupings[gv]; ok {
			return gv2
		}
		if gv2, ok := p.staticGroupings[gv]; ok {
			return gv2
		}
	}
	panic("Oooops")
}

func (vs *ssaScope) searchPolishedGrouping(gv *Grouping) *Grouping {
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.staticGroupings[gv]; ok {
			return gv2
		}
	}
	return gv
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
				vs.createLoopPhiGrouping(loopPhiVar, v2)
				vs.loopPhis = append(vs.loopPhis, loopPhiVar)
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
	if len(c.Dest) != 1 {
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

func (vs *ssaScope) newGrouping() *Grouping {
	gname := "g_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &Grouping{Name: gname, Merged: make(map[*Grouping]bool), scope: vs}
	vs.staticGroupings[gv] = gv
	return gv
}

func (vs *ssaScope) newNamedGrouping(groupSpec *types.GroupSpecifier) *Grouping {
	if grouping, ok := vs.s.parameterGroupings[groupSpec]; ok {
		vs.staticGroupings[grouping] = grouping
		return grouping
	}
	gname := "gn_" + groupSpec.Name
	groupCounter++
	var grouping *Grouping
	if groupSpec.Kind == types.GroupSpecifierIsolate {
		grouping = &Grouping{Name: gname, Merged: make(map[*Grouping]bool), Constraints: GroupResult{}, scope: vs.funcScope(), isParameter: true}
	} else {
		grouping = &Grouping{Name: gname, Merged: make(map[*Grouping]bool), Constraints: GroupResult{NamedGroup: groupSpec.Name}, scope: vs.funcScope(), isParameter: true}
	}
	vs.staticGroupings[grouping] = grouping
	vs.s.parameterGroupings[groupSpec] = grouping
	return grouping
}

func (vs *ssaScope) newScopedGrouping(scope *ircode.CommandScope) *Grouping {
	if gv, ok := vs.s.scopedGroupings[scope]; ok {
		return gv
	}
	gname := "gs_" + strconv.Itoa(scope.ID)
	gv := &Grouping{Name: gname, Merged: make(map[*Grouping]bool), Constraints: GroupResult{Scope: scope}, scope: vs}
	vs.staticGroupings[gv] = gv
	vs.s.scopedGroupings[scope] = gv
	return gv
}

func (vs *ssaScope) newViaGrouping(via *Grouping) *Grouping {
	gname := "g_" + strconv.Itoa(groupCounter) + "_via_" + via.GroupingName()
	groupCounter++
	gv := &Grouping{Name: gname, Merged: make(map[*Grouping]bool), Via: via, Constraints: GroupResult{NamedGroup: "->" + gname}, scope: vs}
	vs.staticGroupings[gv] = gv
	return gv
}

func isLeftMergeable(gv *Grouping) bool {
	// return true // !gv.Closed // && !gv.isPhi()
	return !gv.isPhi()
}

func isRightMergeable(gv *Grouping) bool {
	return !gv.Closed && !gv.IsParameter() && !gv.isPhi()
}

func (vs *ssaScope) merge(gv1 *Grouping, gv2 *Grouping, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) (*Grouping, bool) {
	// Get the latest versions and the scope in which they have been defined
	gvA := vs.lookupGrouping(gv1)
	gvB := vs.lookupGrouping(gv2)

	// The trivial case
	if gvA == gvB {
		return gvA, false
	}

	// Never merge constant groupings with any other groupings
	if gvA.IsConstant() {
		return gvB, false
	}
	if gvB.IsConstant() {
		return gvA, false
	}

	if isLeftMergeable(gvA) && isRightMergeable(gvB) {
		lenIn := len(gvA.In)
		for m := range gvB.Merged {
			gvA.Merged[m] = true
			vs.staticGroupings[m] = gvA
		}
		gvA.Merged[gvB] = true
		gvA.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
		for _, x := range gvB.In {
			gvA.addInput(x)
		}
		for _, x := range gvB.InPhi {
			gvA.addPhiInput(x)
		}
		gvA.Allocations += gvB.Allocations
		gvB.Allocations = 0
		vs.staticGroupings[gvB] = gvA
		return gvA, lenIn > 0 && len(gvA.In) > lenIn
	}

	if isLeftMergeable(gvB) && isRightMergeable(gvA) {
		lenIn := len(gvB.In)
		for m := range gvA.Merged {
			gvB.Merged[m] = true
			vs.staticGroupings[m] = gvB
		}
		gvB.Merged[gvA] = true
		gvB.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
		for _, x := range gvA.In {
			gvB.addInput(x)
		}
		for _, x := range gvA.InPhi {
			gvB.addPhiInput(x)
		}
		gvB.Allocations += gvA.Allocations
		gvA.Allocations = 0
		vs.staticGroupings[gvA] = gvB
		return gvB, lenIn > 0 && len(gvB.In) > lenIn
	}

	// Merge `gvA` and `gvB` into a new group
	gv := vs.newGrouping()
	gv.Closed = true
	gv.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
	// if isRightMergeable(gvA) {
	for m := range gvA.Merged {
		gv.Merged[m] = true
		vs.dynamicGroupings[m] = gv
		// delete(vs.staticGroupings, m)
	}
	/*
		for _, x := range gvA.In {
			gv.addInput(x)
		}
		for _, x := range gvA.InPhi {
			gv.addPhiInput(x)
		}
		// gv.Constraints = gvA.Constraints
		gv.Allocations += gvA.Allocations
	} else {*/
	gv.addInput(gvA)
	gvA.addOutput(gv)
	gvA.Close()
	gv.Merged[gvA] = true
	vs.dynamicGroupings[gvA] = gv
	// delete(vs.staticGroupings, gvA)
	//if isRightMergeable(gvB) {
	for m := range gvB.Merged {
		gv.Merged[m] = true
		vs.dynamicGroupings[m] = gv
		// delete(vs.staticGroupings, m)
	}
	/*
			for _, x := range gvB.In {
				gv.addInput(x)
			}
			for _, x := range gvB.InPhi {
				gv.addPhiInput(x)
			}
			// gv.Constraints = gvB.Constraints
			gv.Allocations += gvB.Allocations
		} else {
	*/
	gv.addInput(gvB)
	gvB.addOutput(gv)
	gvB.Close()
	// gvB.Closed = true
	// }
	gv.Merged[gvB] = true
	vs.dynamicGroupings[gvB] = gv
	// delete(vs.staticGroupings, gvB)

	println("----> MERGE", gv.GroupingName(), "=", gvA.GroupingName(), gvB.GroupingName())
	println("           ", gv1.Name, gv2.Name)
	// println("           ", gvA.Closed, gvA.isPhi(), gvB.Closed, gvB.isPhi())
	return gv, true
}

func (vs *ssaScope) hasParent(p *ssaScope) bool {
	for x := vs.parent; x != nil; x = x.parent {
		if p == x {
			return true
		}
	}
	return false
}

// NoAllocations ...
func (vs *ssaScope) NoAllocations(gv *Grouping) bool {
	if gv.marked {
		return true
	}
	if gv.Allocations != 0 {
		return false
	}
	gv.marked = true
	for _, out := range gv.Out {
		out2 := vs.lookupGrouping(out)
		if out2 == nil {
			panic("Unknown group " + out.Name)
		}
		if !vs.NoAllocations(out2) {
			return false
		}
	}
	gv.marked = false
	return true
}

// Update all groups in the command block such that they reflect the computed group mergers
func (vs *ssaScope) polishBlock(block []*ircode.Command) {
	for _, c := range block {
		vs.polishBlock(c.PreBlock)
		// if c.Op == ircode.OpSetGroupVariable {
		// continue
		// }
		for _, arg := range c.Args {
			if arg.Var != nil {
				if arg.Var.Grouping != nil {
					arg.Var.Grouping = vs.searchPolishedGrouping(arg.Var.Grouping.(*Grouping))
				}
			} else if arg.Const != nil {
				if arg.Const.Grouping != nil {
					arg.Const.Grouping = vs.searchPolishedGrouping(arg.Const.Grouping.(*Grouping))
				}
			}
		}
		for _, dest := range c.Dest {
			if dest != nil && dest.Grouping != nil {
				dest.Grouping = vs.searchPolishedGrouping(dest.Grouping.(*Grouping))
			}
		}
		for i := 0; i < len(c.GroupArgs); i++ {
			c.GroupArgs[i] = vs.searchPolishedGrouping(c.GroupArgs[i].(*Grouping))
		}
	}
}

func (vs *ssaScope) mergeVariablesOnContinue(c *ircode.Command, continueScope *ssaScope, log *errlog.ErrorLog) {
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
	}
}

func (vs *ssaScope) mergeVariables(childScope *ssaScope) {
	for vo, v := range childScope.vars {
		vs.vars[vo] = v
	}
}

func (vs *ssaScope) createLoopPhiGrouping(loopPhiVar, outerVar *ircode.Variable) *Grouping {
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
	phiGrouping := outerScope.newGrouping()
	// TODO	phiGrouping.usedByVar = phiVariable.Original
	phiGrouping.Name += "_loopPhi_" + loopPhiVar.Original.Name
	loopPhiVar.Grouping = phiGrouping
	// Connect the phi-grouping with the groupings of v1 and v2
	phiGrouping.addPhiInput(outerGrouping)
	outerGrouping.addOutput(phiGrouping)
	outerScope.staticGroupings[phiGrouping] = phiGrouping
	return phiGrouping
}
