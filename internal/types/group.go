package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupSpecifierKind ...
type GroupSpecifierKind int

const (
	// GroupSpecifierNamed is a group-specifier with a defined name only, e.g. "`foo".
	GroupSpecifierNamed GroupSpecifierKind = iota
	// GroupSpecifierNew is a group-specifier that implies ownership of a group, e.g "new" or "new `foo".
	GroupSpecifierNew
	// GroupSpecifierShared is a group-specifier that implies that the group is on the heap and that
	// it is reference counted. However, the group cannot be reference counted in another concurrent component.
	GroupSpecifierShared
	// GroupSpecifierConst is a group-specifier that implies that the group is on the heap and that
	// it is reference counted. Even other concurrent components can perform reference counting on
	// this group. However, types marked with this specifier must not be mutable.
	GroupSpecifierConst
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

// NewGroupSpecifier ...
func NewGroupSpecifier(name string, kind GroupSpecifierKind, loc errlog.LocationRange) *GroupSpecifier {
	return &GroupSpecifier{Name: name, Kind: kind, Location: loc}
}

// ToString ...
func (g *GroupSpecifier) ToString() string {
	var str string
	switch g.Kind {
	case GroupSpecifierNew:
		str = "new"
	case GroupSpecifierShared:
		str = "->"
	case GroupSpecifierConst:
		str = "const"
	}
	if g.Name != "" {
		if str != "" {
			str += " `" + g.Name
		} else {
			str = "-" + g.Name
		}
	}
	return str
}
