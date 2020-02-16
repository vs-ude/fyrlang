package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupSpecifierKind ...
type GroupSpecifierKind int

const (
	// GroupSpecifierNamed is a group-specifier with a defined name, e.g. `-grp`.
	GroupSpecifierNamed GroupSpecifierKind = iota
	// GroupSpecifierIsolate is a group-specifier that implies ownership of a group, e.g `->`.
	GroupSpecifierIsolate
)

// GroupSpecifier represents a group specification such as `-grp` or `->`.
type GroupSpecifier struct {
	Kind GroupSpecifierKind
	// Optional. Function parameters can make use of named groups.
	// Two named groups with different names cannot be merged.
	Name string
	// Area in the source code which demands that this group can be computed (in case of a gamma-group)
	Location errlog.LocationRange
}

// NewNamedGroupSpecifier ...
func NewNamedGroupSpecifier(name string, loc errlog.LocationRange) *GroupSpecifier {
	return &GroupSpecifier{Name: name, Kind: GroupSpecifierNamed, Location: loc}
}

// ToString ...
func (g *GroupSpecifier) ToString() string {
	switch g.Kind {
	case GroupSpecifierIsolate:
		return "->"
	case GroupSpecifierNamed:
		return "-" + g.Name
	}
	panic("Should not be here")
}
