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
	// Maps GroupVariables to the GroupVariable that merged them.
	// Yet unmerged GroupVariables are listed here, too, and in this case key and value in the map are equal.
	groups map[*GroupVariable]*GroupVariable
	// Maps the original variable to the latest variable version used in this scope.
	vars          map[*ircode.Variable]*ircode.Variable
	loopPhis      []*ircode.Variable
	loopBreaks    []map[*ircode.Variable]*ircode.Variable
	continueCount int
	breakCount    int
	kind          scopeKind
}

func newScope(s *ssaTransformer, block *ircode.Command) *ssaScope {
	scope := &ssaScope{block: block, s: s, groups: make(map[*GroupVariable]*GroupVariable), vars: make(map[*ircode.Variable]*ircode.Variable)}
	s.scopes = append(s.scopes, scope)
	return scope
}

func (vs *ssaScope) lookupGroup(gv *GroupVariable) (*ssaScope, *GroupVariable) {
	//	if gv.IsParameter() {
	//		return nil, gv
	//	}
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.groups[gv]; ok {
			// TODO: Do not do this for parameter groups
			if p != vs {
				if gv2.IsParameter() {
					vs.groups[gv2] = gv2
				} else {
					gv2.Close()
					newGV := vs.newGroupVariable()
					newGV.addInput(gv2)
					for m := range gv2.Merged {
						newGV.Merged[m] = true
						vs.groups[m] = newGV
					}
					newGV.Merged[gv2] = true
					newGV.Constraints = gv2.Constraints
					// newGV.Closed = true
					vs.groups[gv2] = newGV
					gv2.addOutput(newGV)
					println("------>IMPORT", gv2.GroupVariableName(), newGV.GroupVariableName())
					return vs, newGV
				}
			}
			return p, gv2
		}
	}
	return nil, nil
}

func (vs *ssaScope) searchVariable(v *ircode.Variable) (*ssaScope, *ircode.Variable) {
	if v2, ok := vs.vars[v.Original]; ok {
		return vs, v2
	}
	if vs.parent != nil {
		vs2, v2 := vs.parent.searchVariable(v)
		if v2 != nil {
			return vs2, v2
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
				return vs, vs.newPhiVariable(v)
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
	v2 := &ircode.Variable{Kind: vo.Kind, Name: name, Type: vo.Type, Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, GroupInfo: v.GroupInfo}
	vs.vars[vo] = v2
	// TODO: Remove constant values from the type
	return v2
}

func (vs *ssaScope) newVariableUsageVersion(v *ircode.Variable) *ircode.Variable {
	vo := v.Original
	vo.VersionCount++
	name := vo.Name + "." + strconv.Itoa(vo.VersionCount)
	v2 := &ircode.Variable{Kind: vo.Kind, Name: name, Type: vo.Type, Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, GroupInfo: v.GroupInfo}
	vs.vars[vo] = v2
	return v2
}

func (vs *ssaScope) newPhiVariable(v *ircode.Variable) *ircode.Variable {
	vo := v.Original
	vo.VersionCount++
	name := vo.Name + ".phi" + strconv.Itoa(vo.VersionCount)
	v2 := &ircode.Variable{Kind: ircode.VarPhi, Name: name, Phi: []*ircode.Variable{v}, Type: types.CloneExprType(vo.Type), Scope: vo.Scope, Original: vo, IsInitialized: v.IsInitialized, GroupInfo: v.GroupInfo}
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

func (vs *ssaScope) funcScope() *ssaScope {
	for {
		if vs.kind == scopeFunc {
			return vs
		}
		vs = vs.parent
	}
}

var groupCounter = 0

func (vs *ssaScope) newGroupVariable() *GroupVariable {
	gname := "g_" + strconv.Itoa(groupCounter)
	groupCounter++
	gv := &GroupVariable{Name: gname, Merged: make(map[*GroupVariable]bool), scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newNamedGroupVariable(name string) *GroupVariable {
	if gv, ok := vs.s.namedGroupVariables[name]; ok {
		vs.groups[gv] = gv
		return gv
	}
	gname := "g_" + name
	groupCounter++
	gv := &GroupVariable{Name: gname, Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{NamedGroup: name}, scope: vs.funcScope()}
	vs.groups[gv] = gv
	vs.s.namedGroupVariables[name] = gv
	return gv
}

func (vs *ssaScope) newScopedGroupVariable(scope *ircode.CommandScope) *GroupVariable {
	if gv, ok := vs.s.scopedGroupVariables[scope]; ok {
		return gv
	}
	gname := "gs_" + strconv.Itoa(scope.ID)
	gv := &GroupVariable{Name: gname, Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{Scope: scope}, scope: vs}
	vs.groups[gv] = gv
	vs.s.scopedGroupVariables[scope] = gv
	return gv
}

func (vs *ssaScope) newViaGroupVariable(via *GroupVariable) *GroupVariable {
	gname := "g_" + strconv.Itoa(groupCounter) + "_via_" + via.GroupVariableName()
	groupCounter++
	gv := &GroupVariable{Name: gname, Merged: make(map[*GroupVariable]bool), Via: via, Constraints: GroupResult{NamedGroup: "->" + gname}, scope: vs}
	vs.groups[gv] = gv
	return gv
}

func isLeftMergeable(gv *GroupVariable) bool {
	return !gv.Closed
}

func isRightMergeable(gv *GroupVariable) bool {
	return !gv.Closed && !gv.IsParameter() && gv.usedByVar == nil
}

func (vs *ssaScope) merge(gv1 *GroupVariable, gv2 *GroupVariable, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) (*GroupVariable, bool) {
	// Get the latest versions and the scope in which they have been defined
	_, gvA := vs.lookupGroup(gv1)
	_, gvB := vs.lookupGroup(gv2)

	// The trivial case
	if gvA == gvB {
		return gvA, false
	}

	if isLeftMergeable(gvA) && isRightMergeable(gvB) {
		lenIn := len(gvA.In)
		for m := range gvB.Merged {
			gvA.Merged[m] = true
			vs.groups[m] = gvA
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
		vs.groups[gvB] = gvA
		return gvA, lenIn > 0 && len(gvA.In) > lenIn
	}

	if isLeftMergeable(gvB) && isRightMergeable(gvA) {
		lenIn := len(gvB.In)
		for m := range gvA.Merged {
			gvB.Merged[m] = true
			vs.groups[m] = gvB
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
		vs.groups[gvA] = gvB
		return gvB, lenIn > 0 && len(gvB.In) > lenIn
	}

	// Merge `gvA` and `gvB` into a new group
	gv := vs.newGroupVariable()
	gv.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
	if isRightMergeable(gvA) {
		for m := range gvA.Merged {
			gv.Merged[m] = true
			vs.groups[m] = gv
		}
		for _, x := range gvA.In {
			gv.addInput(x)
		}
		for _, x := range gvA.InPhi {
			gv.addPhiInput(x)
		}
		gv.Constraints = gvA.Constraints
		gv.Allocations += gvA.Allocations
	} else {
		gv.addInput(gvA)
		gvA.addOutput(gv)
		gvA.Closed = true
	}
	gv.Merged[gvA] = true
	vs.groups[gvA] = gv
	if isRightMergeable(gvB) {
		for m := range gvB.Merged {
			gv.Merged[m] = true
			vs.groups[m] = gv
		}
		for _, x := range gvB.In {
			gv.addInput(x)
		}
		for _, x := range gvB.InPhi {
			gv.addPhiInput(x)
		}
		gv.Constraints = gvB.Constraints
		gv.Allocations += gvB.Allocations
	} else {
		gv.addInput(gvB)
		gvB.addOutput(gv)
		gvB.Closed = true
	}
	gv.Merged[gvB] = true
	vs.groups[gvB] = gv
	println("----> MERGE", gv.GroupVariableName(), "=", gvA.GroupVariableName(), gvB.GroupVariableName())
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
func (vs *ssaScope) NoAllocations(gv *GroupVariable) bool {
	if gv.marked {
		return true
	}
	if gv.Allocations != 0 {
		return false
	}
	gv.marked = true
	for _, out := range gv.Out {
		_, out = vs.lookupGroup(out)
		if !vs.NoAllocations(out) {
			return false
		}
	}
	gv.marked = false
	return true
}

func (vs *ssaScope) polishBlock(block []*ircode.Command) {
	// Update all groups in the command block such that they reflect the computed group mergers
	for _, c := range block {
		vs.polishBlock(c.PreBlock)
		for _, arg := range c.Args {
			if arg.Var != nil {
				if arg.Var.GroupInfo != nil {
					_, arg.Var.GroupInfo = vs.lookupGroup(arg.Var.GroupInfo.(*GroupVariable))
				}
			} else if arg.Const != nil {
				if arg.Const.GroupInfo != nil {
					_, arg.Const.GroupInfo = vs.lookupGroup(arg.Const.GroupInfo.(*GroupVariable))
				}
			}
		}
		for _, dest := range c.Dest {
			if dest != nil && dest.GroupInfo != nil {
				_, dest.GroupInfo = vs.lookupGroup(dest.GroupInfo.(*GroupVariable))
			}
		}
		//		for i := 0; i < len(c.GroupArgs); i++ {
		//			_, c.GroupArgs[i] = vs.lookupGroup(c.GroupArgs[i].(*GroupVariable))
		//		}
	}
}

func (vs *ssaScope) mergeVariablesOnContinue(continueScope *ssaScope) {
	if vs.kind != scopeLoop {
		panic("Oooops")
	}
	for _, phi := range vs.loopPhis {
		vs2 := continueScope
		for ; vs2 != vs.parent; vs2 = vs2.parent {
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

func (vs *ssaScope) mergeVariablesOnBreaks() {
	if vs.kind != scopeLoop {
		panic("Oooops")
	}
	phis := make([]*ircode.Variable, len(vs.loopPhis))
	// Iterate over all variables which are imported into this loop-scope
	for i, loopPhi := range vs.loopPhis {
		// For each break determine the variable version for `loopPhi`
		for j, bs := range vs.loopBreaks {
			// A version of this variable was set upon break? If not ...
			v, ok := bs[loopPhi.Original]
			if !ok {
				// ... use the variable version at the beginning of the loop
				v, ok = vs.vars[loopPhi.Original]
				if !ok {
					panic("Ooooops")
				}
			}
			if j == 0 {
				phis[i] = v
			} else if j == 1 {
				p := vs.parent.newPhiVariable(loopPhi)
				p.Phi = append(p.Phi, phis[i])
				p.Phi = append(p.Phi, v)
				phis[i] = p
			} else {
				phis[i].Phi = append(phis[i].Phi, v)
			}
		}
	}
	for i, loopPhi := range vs.loopPhis {
		vs.parent.vars[loopPhi.Original] = phis[i]
	}
}

func (vs *ssaScope) mergeVariablesOnIf(ifScope *ssaScope) {
	for vo, v1 := range ifScope.vars {
		_, v2 := vs.lookupVariable(vo)
		if v2 == nil {
			continue
		}
		phi := vs.newPhiVariable(vo)
		phi.Phi = append(phi.Phi, v1, v2)
		vs.vars[vo] = phi
		createPhiGroup(phi, v1, v2, vs, ifScope, vs)
	}
}

func (vs *ssaScope) mergeVariables(childScope *ssaScope) {
	for vo, v := range childScope.vars {
		vs.vars[vo] = v
	}
}

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
			createPhiGroup(phi, v1, v3, vs, ifScope, elseScope)
		} else {
			// Variable appears in ifScope, but not in elseScope
			phi := vs.newPhiVariable(vo)
			phi.Phi = append(phi.Phi, v1, v2)
			vs.vars[vo] = phi
			createPhiGroup(phi, v1, v2, vs, ifScope, vs)
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
		createPhiGroup(phi, v1, v3, vs, elseScope, vs)
	}
}

func createPhiGroup(phiVariable, v1, v2 *ircode.Variable, phiScope, scope1, scope2 *ssaScope) {
	if phiVariable.GroupInfo == nil {
		return
	}
	gvNew := phiScope.newGroupVariable()
	println("PHI", gvNew.Name)
	// gvNew.childScope = ifScope
	gvNew.usedByVar = phiVariable.Original
	phiVariable.GroupInfo = gvNew
	phiVariable.Original.HasPhiGroup = true
	gv1 := scope1.groups[groupVar(v1)]
	gv2 := scope2.groups[groupVar(v2)]
	gvNew.addPhiInput(gv1)
	gvNew.addPhiInput(gv2)
	gv1.addOutput(gvNew)
	gv2.addOutput(gvNew)
	phiScope.groups[gvNew] = gvNew
}
