package ssa

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

// GroupingKind specified the kind of Grouping.
type GroupingKind int

const (
	// DefaultGrouping is one that has no `Input` groupings
	DefaultGrouping GroupingKind = iota
	// BranchPhiGrouping results from conditional control flow and means that the phi-grouping is the
	// same as one of its `Input` groupings. However, which one is known at runtime only.
	BranchPhiGrouping
	// LoopPhiGrouping ...
	LoopPhiGrouping
	// StaticMergeGrouping means that the grouping is a union of its `Input` groupings.
	// A static merge-grouping means that this union is created at compile time.
	StaticMergeGrouping
	// DynamicMergeGrouping means that the grouping is a union of its `Input` groupings.
	// A static merge-grouping means that this union is created at compile time.
	DynamicMergeGrouping
	// ConstantGrouping represents immutable data that persists throughout the entire lifetime of the program.
	// Such constant groups will never be merged.
	ConstantGrouping
	// ParameterGrouping is passed as parameter via a function call.
	ParameterGrouping
	// ScopedGrouping represents a group that resides on the stack and its lifetime is therefore
	// bound to a `lexicalScope`.
	ScopedGrouping
	// ForeignGrouping repesents a grouping of heap-allocated memory for which no stack-based pointers exist.
	// Therefore the compiler cannot verify the lifetime of the group since in general the compiler cannot
	// determine when such heap-allocated groups are deallocated.
	// ForeinGrouping implies that reference counting is required to fixate such groups in memory.
	ForeignGrouping
)

type groupingAllocationPoint struct {
	scope *ssaScope
	step  int
}

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
	Kind GroupingKind
	// Name of the group variable (or at least the basis to generate such a name).
	Name       string
	Input      []*Grouping
	Output     []*Grouping
	Constraint GroupingConstraint
	// The number of allocations done with this grouping.
	Allocations int
	Original    *Grouping
	Location    errlog.LocationRange
	// The Command that requires the existance of this group.
	Command *ircode.Command
	// The ircode variable used to store a pointer to the corresponding group at runtime or nil.
	groupVar *ircode.Variable
	// A grouping can become unavailable, because they have been assigned to some heap data structure
	// or passed to another component. In this case, the group must no longer be used.
	unavailable bool
	// The scope in which this grouping has been created.
	scope *ssaScope
	// Used with `Kind == LoopPhiGrouping`
	loopScope *ssaScope
	// Used with `Kind == ScopedGrouping` only.
	lexicalScope *ircode.CommandScope
	// Used with kind `DefaultGrouping` and kind `StaticMergeGrouping`.
	// A non-zero value means that this grouping is not merged with any parameter-grouping, phi-grouping, scope-grouping, or foreign-grouping.
	// This is used to determine whether a grouping can be statically merged or needs dynamic merging.
	// This value is only set on the `Original`.
	staticMergePoint *groupingAllocationPoint
	// If non-nil, the grouping uses a phi-group-var to determine its group.
	// Consequently, the group can only be determined at run-time after the phi-group-var has been assigned,
	// which happens for example inside an if-clause or loop.
	// The value determines where in the scope this phi-group-var is set.
	// Groups where `phiAllocationPoint` is nil use a group-var that is known at compile time and the group-var can be used everywhere in the function body.
	// This value is only set on the `Original`.
	phiAllocationPoint *groupingAllocationPoint
	marked             bool
	markerNumber       int
	data               verifierData
}

func (p *groupingAllocationPoint) isEarlierThan(p2 *groupingAllocationPoint) bool {
	if p2.scope.hasParent(p.scope) && p.step < p2.step {
		return true
	}
	if sameLexicalScope(p2.scope, p.scope) {
		return p.step < p2.step
	}
	return false
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
/*
func scopeGrouping(s *ircode.CommandScope) *Grouping {
	ec, ok := s.Grouping.(*Grouping)
	if ok {
		return ec
	}
	return nil
}
*/

func valueGrouping(c *ircode.Command, v *ircode.Variable, vs *ssaScope, loc errlog.LocationRange) *Grouping {
	ec, ok := v.Original.ValueGrouping.(*Grouping)
	if ok && ec != nil {
		return ec
	}
	ss, _ := vs.searchVariable(v)
	if ss == nil {
		panic("Oooops")
	}
	grp := ss.newScopedGrouping(c, v, loc)
	v.Original.ValueGrouping = grp
	return grp
}

// GroupingName implements the ircode.IGrouping interface.
func (gv *Grouping) GroupingName() string {
	return gv.Name
}

// GroupVariable implements the ircode.IGrouping interface.
// Variable returns an ircode.Variable that stores the group pointer for
// the given Grouping at runtime.
func (gv *Grouping) GroupVariable() *ircode.Variable {
	if gv.Original.groupVar != nil {
		return gv.Original.groupVar
	}
	println(gv.Name, gv.Kind, gv, gv.Original, gv.Original, gv.Original)
	panic("Oooops, grouping without a group variable")
}

func (gv *Grouping) isPhi() bool {
	return gv.Kind == BranchPhiGrouping || gv.Kind == LoopPhiGrouping
}

// IsParameter is true if the grouping is passed as a parameter to a function.
func (gv *Grouping) IsParameter() bool {
	return gv.Kind == ParameterGrouping
}

// IsConstant is true if the grouping represents immutable constants.
// These groupings always have a null-group-pointer.
func (gv *Grouping) IsConstant() bool {
	return gv.Kind == ConstantGrouping
}

// addInput ...
func (gv *Grouping) addInput(input *Grouping) {
	// Avoid duplicates
	for _, i := range gv.Input {
		if i == input {
			return
		}
	}
	// Put parameter groupings first, such that GroupVariable() takes it.
	if input.IsParameter() && len(gv.Input) > 0 {
		gv.Input = append(gv.Input, gv.Input[0])
		gv.Input[0] = input
	} else {
		gv.Input = append(gv.Input, input)
	}
}

// addOutput ...
func (gv *Grouping) addOutput(output *Grouping) {
	if output == nil {
		panic("Oooops")
	}
	for _, o := range gv.Output {
		if o == output {
			return
		}
	}
	gv.Output = append(gv.Output, output)
}

// IsDefinitelyUnavailable returns true if the group became unavailable disregarding of
// any if-clauses or loops that might or might have not been entered.
func (gv *Grouping) IsDefinitelyUnavailable() bool {
	// TODO
	return false
}

// IsProbablyUnavailable returns true if it is possible that this group became unavailable.
// "Possible" means that it depends on whether some if-clause has been entered or whether a loop has been entered.
func (gv *Grouping) IsProbablyUnavailable() bool {
	// TODO
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
		if arg.Const.ExprType.Type == types.PrimitiveTypeString || arg.Const.ExprType.IsNullValue() {
			gv := vs.newConstantGrouping(c, arg.Location)
			arg.Const.Grouping = gv
			return gv
		}
		gv := vs.newDefaultGrouping(c, arg.Location)
		gv.Allocations++
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
