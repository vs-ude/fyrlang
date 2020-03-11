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
	vars          map[*ircode.Variable]*ircode.Variable
	loopPhis      []*ircode.Variable
	loopBreaks    []map[*ircode.Variable]*ircode.Variable
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
				vPhi := vs.newPhiVariable(v2)
				// println("------>PHI", v2.Name, vPhi.Name)
				return vs, vPhi
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
	vs.loopPhis = append(vs.loopPhis, v2)
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
	// Iterate over all variables that are defined in an outer scope, but used inside the loop
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

func (vs *ssaScope) mergeVariablesOnBreak(breakScope *ssaScope) {
	if vs.kind != scopeLoop {
		panic("Oooops")
	}
	br := make(map[*ircode.Variable]*ircode.Variable)
	for _, phi := range vs.loopPhis {
		vs2 := breakScope
		for ; vs2 != vs.parent; vs2 = vs2.parent {
			v, ok := vs2.vars[phi.Original]
			if ok {
				br[phi.Original] = v
				break
			}
		}
	}
	vs.loopBreaks = append(vs.loopBreaks, br)
}

// `vs` is the scope of the loop
func (vs *ssaScope) mergeVariablesOnBreaks() {
	if vs.kind != scopeLoop {
		panic("Oooops")
	}
	phis := make([][]*ircode.Variable, len(vs.loopPhis))
	phiGroups := make([][]*Grouping, len(vs.loopPhis))
	// Iterate over all variables which are imported into this loop-scope
	for i, loopPhi := range vs.loopPhis {
		// For each break determine the variable version for `loopPhi`
		for _, bs := range vs.loopBreaks {
			// A version of this variable was set upon break? If not ...
			v, ok := bs[loopPhi.Original]
			if !ok {
				// ... use the variable version at the beginning of the loop
				v, ok = vs.vars[loopPhi.Original]
				if !ok {
					panic("Ooooops")
				}
			}
			if !phisContain(phis[i], v) {
				phis[i] = append(phis[i], v)
			}
			if v.Grouping != nil {
				g := grouping(v)
				if !phiGroupsContain(phiGroups[i], g) {
					phiGroups[i] = append(phiGroups[i], g)
				}
			}
		}
	}
	for i, loopPhi := range vs.loopPhis {
		var v *ircode.Variable
		if len(phis[i]) == 1 {
			v = vs.parent.newVariableVersion(phis[i][0])
		} else {
			v = vs.parent.newPhiVariable(loopPhi)
			v.Phi = phis[i]
		}
		vs.parent.vars[loopPhi.Original] = v
		if len(phiGroups[i]) > 0 {
			gvNew := vs.parent.newGrouping()
			gvNew.InPhi = phiGroups[i]
			// println("------>BREAK", v.Name, gvNew.GroupingName(), "= ...")
			// TODO			gvNew.usedByVar = v.Original
			v.Grouping = gvNew
			// TODO v.Original.HasPhiGrouping = true
			for _, g := range phiGroups[i] {
				g = g.scope.searchGrouping(g)
				// println("     ...", g.GroupingName())
				gvNew.addPhiInput(g)
				g.addOutput(gvNew)
			}
		}
	}
}

func phisContain(phis []*ircode.Variable, test *ircode.Variable) bool {
	for _, p := range phis {
		if p == test {
			return true
		}
	}
	return false
}

func phiGroupsContain(phiGroups []*Grouping, test *Grouping) bool {
	for _, p := range phiGroups {
		if p == test {
			return true
		}
	}
	return false
}

/*
// mergeVariablesOnIf generates phi-variables and phi-groups.
// `vs` is the parent scope and `ifScope` is the scope of the if-clause.
func (vs *ssaScope) mergeVariablesOnIf(ifScope *ssaScope) {
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
			phi := vs.newPhiVariable(vo)
			phi.Phi = append(phi.Phi, v1, v2)
			vs.vars[vo] = phi
			// println("----> PHI ", phi.Name)
			createPhiGrouping(phi, v1, v2, vs, ifScope, vs)
		} else if v1.Grouping != nil {
			// println(vo.Name, grouping(v1).Name)
			// The value of the variable has not changed, but the grouping perhaps?
			grp1 := ifScope.lookupGrouping(grouping(v1))
			grp2 := vs.lookupGrouping(grouping(v2))
			if grp1 != grp2 {
				phi := vs.newPhiVariable(vo)
				phi.Phi = append(phi.Phi, v1, v2)
				vs.vars[vo] = phi
				// println("----> PHI GRP", phi.Name)
				createPhiGrouping(phi, v1, v2, vs, ifScope, vs)
			}
		}
	}
}
*/

func (vs *ssaScope) mergeVariables(childScope *ssaScope) {
	for vo, v := range childScope.vars {
		vs.vars[vo] = v
	}
}

/*
func (vs *ssaScope) mergeVariablesOnIfElse(ifScope *ssaScope, elseScope *ssaScope) {
	for vo, v1 := range ifScope.vars {
		_, v2 := vs.lookupVariable(vo)
		if v2 == nil {
			continue
		}
		if v3, ok := elseScope.vars[vo]; ok {
			// Variable appears in ifScope and elseScope
			phi := vs.newPhiVariable(vo)
			phi.Phi = append(phi.Phi, v1, v3)
			vs.vars[vo] = phi
			createPhiGrouping(phi, v1, v3, vs, ifScope, elseScope)
		} else {
			// Variable appears in ifScope, but not in elseScope
			phi := vs.newPhiVariable(vo)
			phi.Phi = append(phi.Phi, v1, v2)
			vs.vars[vo] = phi
			createPhiGrouping(phi, v1, v2, vs, ifScope, vs)
		}
	}
	for vo, v1 := range elseScope.vars {
		if _, ok := ifScope.vars[vo]; ok {
			continue
		}
		_, v3 := vs.lookupVariable(vo)
		if v3 == nil {
			continue
		}
		// Variable appears in elseScope, but not in ifScope
		phi := vs.newPhiVariable(vo)
		phi.Phi = append(phi.Phi, v1, v3)
		vs.vars[vo] = phi
		createPhiGrouping(phi, v1, v3, vs, elseScope, vs)
	}
}
*/

/*
func createPhiGrouping(phiVariable, v1, v2 *ircode.Variable, phiScope, scope1, scope2 *ssaScope) {
	if phiVariable.Grouping == nil {
		// No grouping, because the variable does not use pointers. Do nothing.
		return
	}
	// Create a phi-grouping
	phiGrouping := phiScope.newGrouping()
	// TODO	phiGrouping.usedByVar = phiVariable.Original
	phiGrouping.Name += "_phi_" + phiVariable.Name
	// Mark the phiVariable as a variable with phi-grouping
	// phiVariable.Grouping = phiGrouping
	// TODO phiVariable.Original.HasPhiGrouping = true
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
	// Connect the phi-grouping with the groupings of v1 and v2
	phiGrouping.addPhiInput(grouping1)
	phiGrouping.addPhiInput(grouping2)
	grouping1.addOutput(phiGrouping)
	grouping2.addOutput(phiGrouping)
	phiScope.staticGroupings[phiGrouping] = phiGrouping

	// println("Out 1. ", grouping1.Name, "->", phiGrouping.Name)
	// println("Out 2. ", grouping2.Name, "->", phiGrouping.Name)
}
*/
