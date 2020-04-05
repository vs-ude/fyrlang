package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
)

// GroupingConstraint ...
type GroupingConstraint struct {
	Scope           *ircode.CommandScope
	NamedGroup      string
	OverConstrained bool
	Error           bool
}

// Equals compares two constraints.
func (gc *GroupingConstraint) Equals(gc2 *GroupingConstraint) bool {
	if gc.Error && gc2.Error {
		return true
	}
	if gc.Error || gc2.Error {
		return false
	}
	if gc.OverConstrained && gc2.OverConstrained {
		return true
	}
	if gc.OverConstrained || gc2.OverConstrained {
		return false
	}
	if gc.Scope != nil && gc.Scope == gc2.Scope {
		return true
	}
	if gc.Scope != nil || gc2.Scope != nil {
		return false
	}
	if gc.NamedGroup != "" && gc.NamedGroup == gc2.NamedGroup {
		return true
	}
	if gc.NamedGroup != "" || gc2.NamedGroup != "" {
		return false
	}
	return true
}

// IsNull returns true if `gc` expresses no constraint.
func (gc *GroupingConstraint) IsNull() bool {
	return !gc.Error && !gc.OverConstrained && gc.Scope == nil && gc.NamedGroup == ""
}

func phiGroupingConstraint(a, b GroupingConstraint, v *ircode.Variable, c *ircode.Command) (result GroupingConstraint) {
	result = mergeGroupingConstraint(a, b, v, c, nil)
	if result.Error {
		result.Error = false
		result.OverConstrained = true
	}
	return result
}

func mergeGroupingConstraint(a, b GroupingConstraint, v *ircode.Variable, c *ircode.Command, log *errlog.ErrorLog) (result GroupingConstraint) {
	if a.Error || b.Error {
		result.Error = true
		return
	}
	var errorKind string
	var errorArgs []string
	if a.OverConstrained || b.OverConstrained {
		result.OverConstrained = true
	}
	if a.OverConstrained && (b.NamedGroup != "" || b.Scope != nil || b.OverConstrained) {
		errorKind = "overconstrained"
		result.Error = true
	} else if b.OverConstrained && (a.NamedGroup != "" || a.Scope != nil || a.OverConstrained) {
		errorKind = "overconstrained"
		result.Error = true
	}
	if a.Scope != nil && b.Scope != nil && a == b {
		result.Scope = a.Scope
	} else if a.Scope != nil && b.Scope != nil && a.Scope.HasParent(b.Scope) {
		result.Scope = a.Scope
	} else if a.Scope != nil && b.Scope != nil && b.Scope.HasParent(a.Scope) {
		result.Scope = b.Scope
	} else if a.Scope != nil && b.Scope == nil {
		result.Scope = a.Scope
	} else if a.Scope == nil && b.Scope != nil {
		result.Scope = b.Scope
	} else if a.Scope == nil && b.Scope == nil {
		result.Scope = nil
	} else {
		errorKind = "both_scoped"
		result.Error = true
	}
	if a.NamedGroup != "" && b.NamedGroup != "" && a.NamedGroup == b.NamedGroup {
		result.NamedGroup = a.NamedGroup
	} else if a.NamedGroup != "" && b.NamedGroup == "" {
		result.NamedGroup = a.NamedGroup
	} else if a.NamedGroup == "" && b.NamedGroup != "" {
		result.NamedGroup = b.NamedGroup
	} else if a.NamedGroup == "" && b.NamedGroup == "" {
		result.NamedGroup = ""
	} else {
		errorKind = "both_named"
		errorArgs = []string{a.NamedGroup, b.NamedGroup}
		result.Error = true
	}
	if result.NamedGroup != "" && result.Scope != nil {
		errorKind = "scoped_and_named"
		errorArgs = []string{result.NamedGroup}
		result.Error = true
	}
	if result.Error && log != nil {
		// println("GROUP ERROR for", v.ToString())
		if v != nil && v.Kind != ircode.VarTemporary {
			errorArgs = append(errorArgs, v.Original.Name)
		}
		errorArgs = append([]string{errorKind}, errorArgs...)
		log.AddError(errlog.ErrorGroupsCannotBeMerged, c.Location, errorArgs...)
	}
	return
}
