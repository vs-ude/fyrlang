package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// GroupSpecifier represents a group specification such as `grp or `grp->.
type GroupSpecifier struct {
	// Area in the source code which demands that this group can be computed (in case of a gamma-group)
	Location errlog.LocationRange
	Elements []GroupSpecifierElement
}

type GroupSpecifierElement struct {
	// Two named groups with different names cannot be merged.
	Name  string
	Arrow bool
}

// NewGroupSpecifier ...
func NewGroupSpecifier(name string, loc errlog.LocationRange) *GroupSpecifier {
	return &GroupSpecifier{Location: loc, Elements: []GroupSpecifierElement{{Name: name}}}
}

// ToString ...
func (g *GroupSpecifier) ToString() string {
	var str string
	for i, e := range g.Elements {
		if i != 0 {
			str += "|"
		}
		str += e.Name
		if e.Arrow {
			str += "->"
		}
	}
	return str
}
