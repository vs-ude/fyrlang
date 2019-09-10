package types

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupState ...
type GroupState int

const (
	// GroupNotComputed ...
	GroupNotComputed GroupState = iota
	// GroupComputing ...
	GroupComputing
	// GroupComputed ...
	GroupComputed
)

// GroupKind ...
type GroupKind int

const (
	// GroupInvalid ...
	GroupInvalid GroupKind = iota
	// GroupFree ...
	GroupFree
	// GroupScoped ...
	GroupScoped
	// GroupNamed ...
	GroupNamed
	// GroupPhi ...
	GroupPhi
	// GroupGamma ...
	GroupGamma
	// GroupIsolate ...
	GroupIsolate
)

// GroupScope is used to express that a group of memory objects is located
// on the stack and alive while the respective scope is alive.
type GroupScope interface {
	HasParent(g GroupScope) bool
}

// Group ...
type Group struct {
	id   int
	Kind GroupKind
	// Used for Phi and Gamma groups
	Groups []*Group
	// Optional. Not used by heap-groups.
	// Groups of two scopes can be merged if the scopes are in a parent child relationship.
	Scope GroupScope
	// Optional. Function parameters can make use of named groups.
	// Two named groups with different names cannot be merged.
	Name  string
	State GroupState
	// Area in the source code which demands that this group can be computed (in case of a gamma-group)
	Location errlog.LocationRange
}

var groupIDCount = 1

// NewFreeGroup ...
func NewFreeGroup(loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, State: GroupComputed, Kind: GroupFree, Location: loc}
}

// NewNamedGroup ...
func NewNamedGroup(name string, loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, Name: name, State: GroupComputed, Kind: GroupNamed, Location: loc}
}

// NewUniquelyNamedGroup ...
func NewUniquelyNamedGroup(loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, Name: "!UNI!" + strconv.Itoa(groupIDCount), State: GroupComputed, Kind: GroupNamed, Location: loc}
}

// NewScopedGroup ...
func NewScopedGroup(scope GroupScope, loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, Scope: scope, State: GroupComputed, Kind: GroupScoped, Location: loc}
}

// NewPhiGroup ...
func NewPhiGroup(groups []*Group, loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, Groups: groups, Kind: GroupPhi, Location: loc}
}

// NewGammaGroup ...
func NewGammaGroup(groups []*Group, loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: groupIDCount, Groups: groups, Kind: GroupGamma, Location: loc}
}

// NewInvalidGroup ...
func NewInvalidGroup(loc errlog.LocationRange) *Group {
	groupIDCount++
	return &Group{id: 2, Name: "!INV!", State: GroupComputed, Kind: GroupInvalid, Location: loc}
}

func (g *Group) makeUniquelyNamed() {
	g.Name = "!UNI!" + strconv.Itoa(g.id)
	g.State = GroupComputed
	g.Kind = GroupNamed
}

func (g *Group) makeFree() {
	g.Name = ""
	g.State = GroupComputed
	g.Kind = GroupFree
}

// Compute ...
func (g *Group) Compute(log *errlog.ErrorLog) {
	g.compute(log, g)
}

func (g *Group) compute(log *errlog.ErrorLog, computeFor *Group) {
	switch g.State {
	case GroupNotComputed:
		g.State = GroupComputing
		for _, phi := range g.Groups {
			if phi == g {
				continue
			}
			phi.compute(log, g)
		}
		g.unify(log)
		g.State = GroupComputed
	case GroupComputing:
		if computeFor == g {
			panic("Must not happen")
		}
		for _, phi := range g.Groups {
			if phi == computeFor {
				continue
			}
			phi.compute(log, g)
		}
		g.unify(log)
	case GroupComputed:
		return
	}
}

func (g *Group) unify(log *errlog.ErrorLog) {
	kind := g.Kind
	if g.Kind != GroupPhi && g.Kind != GroupGamma {
		panic("Unify on non-gamma and non-phi group")
	}
	// Start with the first group
	g.Name = g.Groups[0].Name
	g.Scope = g.Groups[0].Scope
	g.Kind = g.Groups[0].Kind
	if g.Kind == GroupInvalid {
		return
	}
	// Try to merge in all the remaining groups
	for _, m := range g.Groups[1:] {
		// If the current group and the group of m are equivalent, ok ...
		if g == m {
			continue
		}
		if m.Kind == GroupPhi || m.Kind == GroupGamma {
			panic("Should have been unified")
		}
		if m.Kind == GroupInvalid {
			g.Kind = GroupInvalid
			g.Name = ""
			g.Scope = nil
			return
		}
		if m.Kind == GroupFree {
			continue
		}
		if g.Kind == GroupFree {
			g.Name = m.Name
			g.Scope = m.Scope
			g.Kind = m.Kind
			continue
		}
		// Groups of the same name can of course be unified
		if g.Kind == GroupNamed && m.Kind == GroupNamed && g.Name == m.Name {
			continue
		}
		// Groups of two different names cannot be unified unless one of them is free.
		if g.Kind == GroupNamed || m.Kind == GroupNamed {
			if kind == GroupGamma {
				log.AddError(errlog.ErrorNamedGroupMerge, g.Location)
				// Make `g` a free group to avoid a cascade of followup errors
				g.makeFree()
				return
			}
			g.makeUniquelyNamed()
			return
		}
		if m.Kind != GroupScoped || g.Kind != GroupScoped {
			panic("Should be scoped")
		}
		// Both groups must be scoped
		if g.Scope.HasParent(m.Scope) {
			// Do nothing
		} else if m.Scope.HasParent(g.Scope) {
			g.Scope = m.Scope
		} else {
			if kind == GroupGamma {
				log.AddError(errlog.ErrorScopedGroupMerge, g.Location)
				// Make `g` a free group to avoid a cascade of followup errors
				g.makeFree()
				return
			}
			g.makeUniquelyNamed()
			return
		}
	}
}

// ToString ...
func (g *Group) ToString() string {
	switch g.Kind {
	case GroupIsolate:
		return "I"
	case GroupInvalid:
		return "V"
	case GroupNamed:
		return g.Name
	case GroupPhi:
		return "P" + strconv.Itoa(g.id)
	case GroupGamma:
		return "G" + strconv.Itoa(g.id)
	case GroupScoped:
		return "S"
	case GroupFree:
		return "F"
	}
	panic("Should not be here")
}
