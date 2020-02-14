package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupKind ...
type GroupKind int

const (
	// GroupNamed is a group-specifier with a defined name, e.g. `-grp`.
	GroupNamed GroupKind = iota
	// GroupIsolate is a group-specifier that implies ownership of a group, e.g `->`.
	GroupIsolate
)

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
