package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Grouping is an artefact of program analysis.
// Each ircode variable is assigned to a grouping.
// Multiple variables can belong to the same Grouping which implies
// that the lifetime of these variables becomes the same.
// Furthermore, groupings can be merged or can be the result of changing
// an input grouping.
// A Grouping can be closed, which makes it immutable.
// An immutable Grouping can be "changed" by creating a new grouping that takes
// the immutable grouping as input.
type Grouping struct {
	// Name of the group variable.
	Name   string
	In     []*Grouping
	InPhi  []*Grouping
	Out    []*Grouping
	Merged map[*Grouping]bool
	// No groupings or inputs may be added to an equivalence class that is closed.
	// Furthermore, no other equivalence class can join with a closed equivalence class.
	// However, other equivalence classes can use a closed one as their input.
	Closed      bool
	Constraints GroupResult
	Via         *Grouping
	// The number of allocations done with this group.
	Allocations int
	// The ircode variable used to store a pointer to the group at runtime.
	Var *ircode.Variable
	// Closed groupings can be unavailable, because they have been assigned to some heap data structure
	// or passed to another component. In this case, the group must no longer be used.
	unavailable bool
	// The scope in which this group has been defined.
	scope *ssaScope
	// For phi-groups this identifies the variable for which this phi-group has been created.
	usedByVar   *ircode.Variable
	isParameter bool
	isConstant  bool
	marked      bool
}

// Returns the Grouping associated with some ircode variable.
func grouping(v *ircode.Variable) *Grouping {
	ec, ok := v.Grouping.(*Grouping)
	if ok {
		return ec
	}
	return nil
}

// Returns the Grouping associated with all stack-based ircode variables that exist in the given scope.
func scopeGrouping(s *ircode.CommandScope) *Grouping {
	ec, ok := s.Grouping.(*Grouping)
	if ok {
		return ec
	}
	return nil
}

// GroupingName implements the ircode.IGrouping interface.
func (gv *Grouping) GroupingName() string {
	return gv.Name
}

// GroupVariable implements the ircode.IGrouping interface.
// Variable returns an ircode.Variable that stores the group pointer for
// the given Grouping at runtime.
func (gv *Grouping) GroupVariable() *ircode.Variable {
	if gv.Var != nil {
		return gv.Var
	}
	// Phi-groups are used by a single variable and this variable has a phi-group-pointer variable.
	if gv.usedByVar != nil {
		return gv.usedByVar.PhiGroupVariable
	}
	if len(gv.In) != 0 {
		// Search for the first input that is not a phi-group
		for _, gv2 := range gv.In {
			if !gv2.isPhi() {
				return gv2.GroupVariable()
			}
		}
		return gv.In[0].GroupVariable()
	}
	return nil
}

func (gv *Grouping) isPhi() bool {
	return gv.usedByVar != nil
}

// IsParameter is true if the grouping is passed as a parameter to a function.
func (gv *Grouping) IsParameter() bool {
	return gv.isParameter
}

// IsConstant is true if the grouping represents immutable constants.
// These groupings always have a null-group-pointer.
func (gv *Grouping) IsConstant() bool {
	return gv.isConstant
}

// Close ...
func (gv *Grouping) Close() {
	gv.Closed = true
}

// addInput ...
func (gv *Grouping) addInput(input *Grouping) {
	// Avoid duplicates
	for _, i := range gv.In {
		if i == input {
			return
		}
	}
	// Put parameter groupings first, such taht GroupVariable() takes it.
	if input.IsParameter() && len(gv.In) > 0 {
		gv.In = append(gv.In, gv.In[0])
		gv.In[0] = input
	} else {
		gv.In = append(gv.In, input)
	}
}

// addPhiInput ...
func (gv *Grouping) addPhiInput(input *Grouping) {
	for _, i := range gv.InPhi {
		if i == input {
			return
		}
	}
	gv.InPhi = append(gv.InPhi, input)
}

// addOutput ...
func (gv *Grouping) addOutput(output *Grouping) {
	if output == nil {
		panic("Oooops")
	}
	if gv == nil {
		panic("Oooops nil")
	}
	for _, o := range gv.Out {
		if o == output {
			return
		}
	}
	gv.Out = append(gv.Out, output)
}

func (gv *Grouping) makeUnavailable() {
	gv.Closed = true
	gv.unavailable = true
}

// IsDefinitelyUnavailable ...
func (gv *Grouping) IsDefinitelyUnavailable() bool {
	return gv.unavailable
}

// IsProbablyUnavailable ...
func (gv *Grouping) IsProbablyUnavailable() bool {
	if gv.unavailable {
		return true
	}
	for _, gphi := range gv.InPhi {
		if gphi.IsProbablyUnavailable() {
			return true
		}
	}
	for _, gin := range gv.In {
		if gin.IsProbablyUnavailable() {
			return true
		}
	}
	return false
}

func argumentGrouping(c *ircode.Command, arg ircode.Argument, vs *ssaScope, loc errlog.LocationRange) *Grouping {
	if arg.Var != nil {
		return grouping(arg.Var)
	}
	if arg.Const.Grouping != nil {
		return arg.Const.Grouping.(*Grouping)
	}
	// If the const contains heap allocated data, attach a group variable
	if types.TypeHasPointers(arg.Const.ExprType.Type) {
		gv := vs.newGrouping()
		if arg.Const.ExprType.Type == types.PrimitiveTypeString {
			gv.isConstant = true
		}
		if arg.Const.ExprType.IsNullValue() {
			gv.Allocations++
		}
		/*
			// Check whether the constant contains any memory allocations
			if arg.Const.HasMemoryAllocations() {
				gv.Allocations++
			}
		*/
		vs.staticGroupings[gv] = gv
		arg.Const.Grouping = gv
		return gv
	}
	return nil
}

// Go sucks
func setGrouping(v *ircode.Variable, gv *Grouping) {
	if gv == nil {
		v.Grouping = nil
	} else {
		v.Grouping = gv
	}
}
