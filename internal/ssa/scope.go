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

func newScope() *ssaScope {
	return &ssaScope{groups: make(map[*GroupVariable]*GroupVariable), vars: make(map[*ircode.Variable]*ircode.Variable)}
}

func (vs *ssaScope) lookupGroup(gv *GroupVariable) (*ssaScope, *GroupVariable) {
	for p := vs; p != nil; p = p.parent {
		if gv2, ok := p.groups[gv]; ok {
			if p != vs {
				gv2.Close()
				newGV := vs.newGroupVariable()
				newGV.In[gv2] = true
				for m := range gv2.Merged {
					newGV.Merged[m] = true
					vs.groups[m] = newGV
				}
				newGV.Merged[gv2] = true
				newGV.Constraints = gv2.Constraints
				vs.groups[gv2] = newGV
				return vs, newGV
			}
			return p, gv2
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
				return vs, vs.newLoopPhiVariable(v)
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

func (vs *ssaScope) newLoopPhiVariable(v *ircode.Variable) *ircode.Variable {
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
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Out: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newNamedGroupVariable(name string) *GroupVariable {
	gname := "g_" + name
	groupCounter++
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Out: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{NamedGroup: name}, scope: vs.funcScope()}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newScopedGroupVariable(scope *ircode.CommandScope) *GroupVariable {
	gname := "gs_" + strconv.Itoa(scope.ID)
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Out: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{Scope: scope}, scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newViaGroupVariable(via *GroupVariable) *GroupVariable {
	gname := "g_" + strconv.Itoa(groupCounter) + "_via_" + via.GroupVariableName()
	groupCounter++
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Out: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Via: via, Constraints: GroupResult{NamedGroup: "->" + gname}, scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) merge(gv1 *GroupVariable, gv2 *GroupVariable, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) *GroupVariable {
	// Get the latest versions and the scope in which they have been defined
	scopeA, gvA := vs.lookupGroup(gv1)
	scopeB, gvB := vs.lookupGroup(gv2)

	// The trivial case
	if gvA == gvB {
		return gvA
	}

	// If `gvA` is a named group, then merge everything else into this named group.
	if gvA.Constraints.NamedGroup != "" {
		if scopeB == vs && !gvB.Closed {
			for m := range gvB.Merged {
				gvA.Merged[m] = true
				vs.groups[m] = gvA
			}
			gvA.Merged[gvB] = true
			gvA.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
			for x, v := range gvB.In {
				gvA.In[x] = v
			}
			gvA.Allocations += gvB.Allocations
			gvB.Allocations = 0
			vs.groups[gvB] = gvA
		} else {
			gvA.In[gvB] = true
			gvB.Closed = true
		}
		return gvA
	}

	// If `gvB` is a named group, then merge everything else into this named group.
	if gvB.Constraints.NamedGroup != "" {
		if scopeA == vs && !gvA.Closed {
			for m := range gvA.Merged {
				gvB.Merged[m] = true
				vs.groups[m] = gvB
			}
			gvB.Merged[gvA] = true
			gvB.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
			for x, v := range gvA.In {
				gvB.In[x] = v
			}
			gvB.Allocations += gvA.Allocations
			gvA.Allocations = 0
			vs.groups[gvA] = gvB
		} else {
			gvB.In[gvA] = true
			gvA.Closed = true
		}
		return gvB
	}

	// Both groups are in the same scope. So simply merge one into the other
	if scopeA == vs && scopeB == vs && !gvA.Closed && !gvB.Closed {
		for m := range gvB.Merged {
			gvA.Merged[m] = true
			vs.groups[m] = gvA
		}
		gvA.Merged[gvB] = true
		gvA.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
		for x, v := range gvB.In {
			gvA.In[x] = v
		}
		gvA.Allocations += gvB.Allocations
		gvB.Allocations = 0
		vs.groups[gvA] = gvA
		vs.groups[gvB] = gvA
		return gvA
	}

	// Merge `gvA` and `gvB` into a new group
	gv := vs.newGroupVariable()
	gv.Constraints = mergeGroupResult(gvA.Constraints, gvB.Constraints, v, c, log)
	if scopeA == vs && !gvA.Closed {
		for m := range gvA.Merged {
			gv.Merged[m] = true
			vs.groups[m] = gv
		}
		gv.Merged[gvA] = true
		for x, v := range gvA.In {
			gv.In[x] = v
		}
		gv.Constraints = gvA.Constraints
		gv.Allocations += gvA.Allocations
		vs.groups[gvA] = gv
	} else {
		gv.In[gvA] = true
		gvA.Closed = true
	}
	gvA.Out[gv] = true
	if scopeB == vs && !gvB.Closed {
		for m := range gvB.Merged {
			gv.Merged[m] = true
			vs.groups[m] = gv
		}
		gv.Merged[gvB] = true
		for x, v := range gvB.In {
			gv.In[x] = v
		}
		gv.Constraints = gvB.Constraints
		gv.Allocations += gvB.Allocations
		vs.groups[gvB] = gv
	} else {
		gv.In[gvB] = true
		gvB.Closed = true
	}
	gvB.Out[gv] = true
	return gv
}

func (vs *ssaScope) hasParent(p *ssaScope) bool {
	for x := vs.parent; x != nil; x = x.parent {
		if p == x {
			return true
		}
	}
	return false
}

func (vs *ssaScope) groupVariableMergesOuterScope(gv *GroupVariable) bool {
	if gv.marked {
		return false
	}
	if vs.hasParent(gv.scope) {
		return true
	}
	if len(gv.Out) == 0 {
		return false
	}
	gv.marked = true
	for out := range gv.Out {
		_, out2 := vs.lookupGroup(out)
		if out != nil && out2 == nil {
			panic("Shit")
		}
		if vs.groupVariableMergesOuterScope(out) {
			return true
		}
	}
	gv.marked = false
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
	for out := range gv.Out {
		_, out = vs.lookupGroup(out)
		if !vs.NoAllocations(out) {
			return false
		}
	}
	gv.marked = false
	return true
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
				p := vs.parent.newLoopPhiVariable(loopPhi)
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
		phi := vs.newLoopPhiVariable(vo)
		phi.Phi = append(phi.Phi, v1, v2)
		vs.vars[vo] = phi
	}
}

func (vs *ssaScope) mergeVariables(childScope *ssaScope) {
	for vo, v := range childScope.vars {
		vs.vars[vo] = v
	}
}

func (vs *ssaScope) mergeVariablesOnIfElse(ifScope *ssaScope, elseScope *ssaScope) {
	for vo, v1 := range ifScope.vars {
		var ok bool
		var v2 *ircode.Variable
		if v2, ok = elseScope.vars[vo]; !ok {
			_, v2 = vs.lookupVariable(vo)
			if v2 == nil {
				continue
			}
		}
		phi := vs.newLoopPhiVariable(vo)
		phi.Phi = append(phi.Phi, v1, v2)
		vs.vars[vo] = phi
	}
	for vo, v1 := range elseScope.vars {
		if _, ok := ifScope.vars[vo]; ok {
			continue
		}
		_, v2 := vs.lookupVariable(vo)
		if v2 == nil {
			continue
		}
		phi := vs.newLoopPhiVariable(vo)
		phi.Phi = append(phi.Phi, v1, v2)
		vs.vars[vo] = phi
	}
}
