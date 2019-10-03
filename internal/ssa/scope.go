package ssa

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
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
			return p, gv2
		}
	}
	return nil, nil
}

func (vs *ssaScope) lookupVariable(v *ircode.Variable) (*ssaScope, *ircode.Variable) {
	for p := vs; p != nil; p = p.parent {
		if v2, ok := p.vars[v]; ok {
			return p, v2
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

func (vs *ssaScope) defineVariable(v *ircode.Variable) {
	vs.vars[v] = v
}

func (vs *ssaScope) createDestinationVariable(c *ircode.Command) *ircode.Variable {
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
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newNamedGroupVariable(name string) *GroupVariable {
	gname := "g_" + name
	groupCounter++
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{NamedGroup: name}, scope: vs.funcScope()}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newScopedGroupVariable(scope *ircode.CommandScope) *GroupVariable {
	gname := "gs_" + strconv.Itoa(scope.ID)
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Constraints: GroupResult{Scope: scope}, scope: vs}
	vs.groups[gv] = gv
	return gv
}

func (vs *ssaScope) newViaGroupVariable(via *GroupVariable) *GroupVariable {
	gname := "g_" + strconv.Itoa(groupCounter) + "_via_" + via.GroupVariableName()
	groupCounter++
	gv := &GroupVariable{Name: gname, In: make(map[*GroupVariable]bool), Merged: make(map[*GroupVariable]bool), Via: via, Constraints: GroupResult{NamedGroup: "->" + gname}, scope: vs}
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
		for m := range gv2.Merged {
			gv.Merged[m] = true
			vs.groups[m] = gv
		}
		for x, v := range gv2.In {
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
		_, out = vs.lookupGroup(out)
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
