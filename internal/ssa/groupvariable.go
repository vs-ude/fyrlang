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
	In map[*GroupVariable]bool
	// All groups that (potentially) merge this group are keys in this map.
	// A value of true means that these other groups do definitly list this group.
	// A value of false means that at most one of these other groups will merge this one at runtime,
	// but at compile it is unknown which one.
	Out    map[*GroupVariable]bool
	Merged map[*GroupVariable]bool
	// No group variables or inputs may be added to an equivalence class that is closed.
	// Furthermore, no other equivalence class can join with a closed equivalence class.
	// However, other equivalence classes can use a closed one as their input.
	Closed      bool
	Constraints GroupResult
	Via         *GroupVariable
	// The number of allocations done with this group.
	Allocations int
	// The top-most scope to which this group variables has some ties (e.g. by using a group variable as input)
	scope  *ssaScope
	marked bool
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

// GroupVariableName ...
func (gv *GroupVariable) GroupVariableName() string {
	return gv.Name
}

// Close ...
func (gv *GroupVariable) Close() {
	gv.Closed = true
}

// AddInput ...
func (gv *GroupVariable) AddInput(input *GroupVariable) {
	if _, ok := gv.In[input]; ok {
		return
	}
	gv.In[input] = true
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
		// Check whether the constant contains any memory allocations
		if arg.Const.HasMemoryAllocations() {
			gv.Allocations++
		}
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

func accessChainGroupVariable(c *ircode.Command, vs *ssaScope, log *errlog.ErrorLog) *GroupVariable {
	// Shortcut in case the result of the access chain carries no pointers at all.
	if !types.TypeHasPointers(c.Type.Type) {
		return nil
	}
	if len(c.AccessChain) == 0 {
		panic("No access chain")
	}
	if c.Args[0].Var == nil {
		panic("Access chain is not accessing a variable")
	}
	// The variable on which this access chain starts is stored as local variable in a scope.
	// Thus, the group of this value is a scoped group.
	valueGroup := scopeGroupVar(c.Args[0].Var.Scope)
	// The variable on which this access chain starts might have pointers.
	// Determine to group to which these pointers are pointing.
	ptrDestGroup := groupVar(c.Args[0].Var)
	if ptrDestGroup == nil {
		// The variable has no pointers. In this case the only possible operation is to take the address of take a slice.
		ptrDestGroup = valueGroup
	}
	for _, ac := range c.AccessChain {
		switch ac.Kind {
		case ircode.AccessAddressOf:
			// The result of `&expr` must be a pointer.
			pt, ok := types.GetPointerType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a pointer")
			}
			// The resulting pointer is assigned to an unsafe pointer? -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			if ac.InputType.PointerDestGroup != nil && ac.InputType.PointerDestGroup.Kind == types.GroupIsolate {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value is in turn an isolate pointer. But that is ok, since the type system has this information in form of a GroupType.
				ptrDestGroup = valueGroup
			} else {
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into one group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = vs.merge(valueGroup, ptrDestGroup, nil, c, log)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = scopeGroupVar(c.Scope)
		case ircode.AccessSlice:
			// The result of `expr[a:b]` must be a slice.
			_, ok := types.GetSliceType(ac.OutputType.Type)
			if !ok {
				panic("Output is not a slice")
			}
			if _, ok := types.GetSliceType(ac.InputType.Type); ok {
				// Do nothing by intention. A slice of a slice points to the same group as the original slice.
			} else {
				_, ok := types.GetArrayType(ac.InputType.Type)
				if !ok {
					panic("Input is not a slice and not an array")
				}
				// The resulting pointer does now point to the group of the value of which the address has been taken (valueGroup).
				// This value may contain further pointers to a group stored in `ptrDestGroup`.
				// Pointers and all pointers from there on must point to the same group (unless it is an isolate pointer).
				// Therefore, the `valueGroup` and `ptrDestGroup` must be merged into one group.
				if valueGroup != ptrDestGroup {
					ptrDestGroup = vs.merge(valueGroup, ptrDestGroup, nil, c, log)
				}
			}
			// The value is now a temporary variable on the stack.
			// Therefore its group is a scoped group
			valueGroup = scopeGroupVar(c.Scope)
		case ircode.AccessStruct:
			_, ok := types.GetStructType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessPointerToStruct:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			_, ok = types.GetStructType(pt.ElementType)
			if !ok {
				panic("Not a struct")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessArrayIndex:
			_, ok := types.GetArrayType(ac.InputType.Type)
			if !ok {
				panic("Not a struct")
			}
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessSliceIndex:
			_, ok := types.GetSliceType(ac.InputType.Type)
			if !ok {
				panic("Not a slice")
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		case ircode.AccessDereferencePointer:
			pt, ok := types.GetPointerType(ac.InputType.Type)
			if !ok {
				panic("Not a pointer")
			}
			// Following an unsafe pointer -> give up
			if pt.Mode == types.PtrUnsafe {
				return nil
			}
			valueGroup = ptrDestGroup
			if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupIsolate {
				ptrDestGroup = vs.newViaGroupVariable(valueGroup)
			} else if ac.OutputType.PointerDestGroup != nil && ac.OutputType.PointerDestGroup.Kind == types.GroupNamed {
				ptrDestGroup = vs.newNamedGroupVariable(ac.OutputType.PointerDestGroup.Name)
			}
		default:
			panic("Oooops")
		}
	}
	return ptrDestGroup
}
