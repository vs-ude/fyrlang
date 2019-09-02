package ircode

import (
	"fmt"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/types"
)

// Function ...
type Function struct {
	Name string
	Type *types.FuncType
	// This command is only required to hold a list of commands in its Branch field.
	Body Command
	// Functions that are instantiated from a generic need to be de-duplicated when linking.
	IsGenericInstance bool
	// True if the function should be visible to other packages.
	IsExported     bool
	vars           []*Variable
	parameterScope *CommandScope
}

// NewFunction ...
func NewFunction(name string, t *types.FuncType) *Function {
	f := &Function{Type: t, Name: name}
	f.Body.Op = OpBlock
	// The scope used for function parameters
	f.parameterScope = newScope(nil)
	// The scope used for local variables (unless they are inside a nested scope)
	f.Body.Scope = newScope(f.parameterScope)
	return f
}

// ToString ...
func (f *Function) ToString() string {
	str := fmt.Sprintf("func %v %v { // %v\n", f.Name, f.Type.ToFunctionSignature(), strconv.Itoa(f.Body.Scope.ID))
	str += f.Body.ToString("")
	str += "}\n\n"
	return str
}
