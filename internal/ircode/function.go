package ircode

import (
	"fmt"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
)

// Function ...
type Function struct {
	Name string
	Type *FunctionType
	// This command is only required to hold a list of commands in its Branch field.
	Body           Command
	vars           []*Variable
	parameterScope *CommandScope
}

// NewFunction ...
func NewFunction(name string) *Function {
	f := &Function{Type: &FunctionType{Type: Type{Name: name}}, Name: name}
	f.Body.Type = VoidType
	f.Body.Op = OpBlock
	// The scope used for function parameters
	f.parameterScope = newScope(nil)
	// The scope used for local variables (unless they are inside a nested scope)
	f.Body.Scope = newScope(f.parameterScope)
	return f
}

// AddParameter ...
func (f *Function) AddParameter(name string, t IType, ptrGroupName string, loc errlog.LocationRange) *Variable {
	v := &Variable{Type: t, Name: name, Scope: f.parameterScope}
	v.Original = v
	vu := VariableUsage{Var: v}
	/*
		if IsPureValueType(t) {
			vu.Group.Pointer = NewFreeGroup(loc)
		} else if ptrGroupName != "" {
			vu.Group.Pointer = NewNamedGroup(ptrGroupName, loc)
		} else {
			vu.Group.Pointer = NewScopedGroup(f.parameterScope, loc)
		}
		if IsBorrowedType(t) {
			vu.Group.Borrow = NewScopedGroup(f.parameterScope, loc)
		}
	*/
	f.Type.Params = append(f.Type.Params, vu)
	return v
}

// ToString ...
func (f *Function) ToString() string {
	str := fmt.Sprintf("func %v %v { // %v\n", f.Name, f.Type.ToFunctionSignature(), strconv.Itoa(f.Body.Scope.ID))
	str += f.Body.ToString("")
	str += "}\n\n"
	return str
}
