package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupKind ...
type GroupKind int

const (
	// GroupNamed ...
	GroupNamed GroupKind = iota
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
	Kind GroupKind
	// Optional. Function parameters can make use of named groups.
	// Two named groups with different names cannot be merged.
	Name string
	// Area in the source code which demands that this group can be computed (in case of a gamma-group)
	Location errlog.LocationRange
}

// NewNamedGroup ...
func NewNamedGroup(name string, loc errlog.LocationRange) *Group {
	return &Group{Name: name, Kind: GroupNamed, Location: loc}
}

// ToString ...
func (g *Group) ToString() string {
	switch g.Kind {
	case GroupIsolate:
		return "->"
	case GroupNamed:
		return g.Name
	}
	panic("Should not be here")
}
