package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

/*
// GroupInKind ...
type GroupInKind int

const (
	mergeGroupFromOuterScope GroupInKind = 1 + iota
	mergeGroupFromSameOrChildScope
	phiGroupFromOuterScope
	phiGroupFromSameOrChildScope
)
*/

// GroupVariable ...
type GroupVariable struct {
	Name string
	// All groups mentioned as keys in the map are (potentially) merged with this group.
	// If the value in the map is true, the other group is merged.
	// For all groups with a value of false, at most one of them will be merged, but it is unknown
	// at compile time which ones.
	In    []*GroupVariable
	InPhi []*GroupVariable
	// All groups that (potentially) merge this group are keys in this map.
	// A value of true means that these other groups do definitly list this group.
	// A value of false means that at most one of these other groups will merge this one at runtime,
	// but at compile it is unknown which one.
	Out    []*GroupVariable
	Merged map[*GroupVariable]bool
	// No group variables or inputs may be added to an equivalence class that is closed.
	// Furthermore, no other equivalence class can join with a closed equivalence class.
	// However, other equivalence classes can use a closed one as their input.
	Closed bool
	// Closed groups can be unavailable, because they have been assigned to some heap data structure
	// or passed to another component. In this case, the group must no longer be used.
	Unavailable bool
	Constraints GroupResult
	Via         *GroupVariable
	// The number of allocations done with this group.
	Allocations int
	// The variable used to store a pointer to the group.
	Var *ircode.Variable
	// The top-most scope to which this group variables has some ties (e.g. by using a group variable as input)
	scope *ssaScope
	// The `childScope` that made this group necessary, or nil.
	// This is for example an `OpIf` of `OpLoop`.
	// The `childScope` is a child of `scope`.
	// childScope *ssaScope
	// For phi-groups this identifies the variable for which this phi-group has been created.
	usedByVar *ircode.Variable
	marked    bool
}

func groupVar(v *ircode.Variable) *GroupVariable {
	ec, ok := v.GroupInfo.(*GroupVariable)
	if ok {
		return ec
	}
	return nil
}

func scopeGroupVar(s *ircode.CommandScope) *GroupVariable {
	ec, ok := s.GroupInfo.(*GroupVariable)
	if ok {
		return ec
	}
	return nil
}

// GroupVariableName implements ircode.IGroupVariable
func (gv *GroupVariable) GroupVariableName() string {
	if gv == nil {
		return "FUCK"
	}
	return gv.Name
}

// Variable implements ircode.IGroupVariable
func (gv *GroupVariable) Variable() *ircode.Variable {
	if gv.Var != nil {
		return gv.Var
	}
	if gv.usedByVar != nil {
		return gv.usedByVar.PhiGroupVariable
	}
	if len(gv.In) != 0 {
		return gv.In[0].Variable()
	}
	return nil
}

// IsParameter ...
func (gv *GroupVariable) IsParameter() bool {
	return gv.Constraints.NamedGroup != "" && gv.Via == nil && len(gv.In) == 0 && len(gv.InPhi) == 0
}

// Close ...
func (gv *GroupVariable) Close() {
	gv.Closed = true
}

// addInput ...
func (gv *GroupVariable) addInput(input *GroupVariable) {
	for _, i := range gv.In {
		if i == input {
			return
		}
	}
	if input.IsParameter() && len(gv.In) > 0 {
		gv.In = append(gv.In, gv.In[0])
		gv.In[0] = input
	}
	gv.In = append(gv.In, input)
}

// addPhiInput ...
func (gv *GroupVariable) addPhiInput(input *GroupVariable) {
	for _, i := range gv.InPhi {
		if i == input {
			return
		}
	}
	gv.InPhi = append(gv.InPhi, input)
}

// addOutput ...
func (gv *GroupVariable) addOutput(output *GroupVariable) {
	for _, o := range gv.Out {
		if o == output {
			return
		}
	}
	gv.Out = append(gv.Out, output)
}

func (gv *GroupVariable) makeUnavailable() {
	gv.Closed = true
	gv.Unavailable = true
}

func argumentGroupVariable(c *ircode.Command, arg ircode.Argument, vs *ssaScope, loc errlog.LocationRange) *GroupVariable {
	if arg.Var != nil {
		return groupVar(arg.Var)
	}
	if arg.Const.GroupInfo != nil {
		return arg.Const.GroupInfo.(*GroupVariable)
	}
	// If the const contains heap allocated data, attach a group variable
	if types.TypeHasPointers(arg.Const.ExprType.Type) {
		gv := vs.newGroupVariable()
		/*
			// Check whether the constant contains any memory allocations
			if arg.Const.HasMemoryAllocations() {
				gv.Allocations++
			}
		*/
		vs.groups[gv] = gv
		arg.Const.GroupInfo = gv
		return gv
	}
	return nil
}

// Go sucks
func setGroupVariable(v *ircode.Variable, gv *GroupVariable) {
	if gv == nil {
		v.GroupInfo = nil
	} else {
		v.GroupInfo = gv
	}
}
