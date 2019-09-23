package ircode

import (
	"fmt"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/types"
)

// Function ...
type Function struct {
	Name string
	Func *types.Func
	// This command is only required to hold a list of commands in its Branch field.
	Body Command
	// Functions that are instantiated from a generic need to be de-duplicated when linking.
	IsGenericInstance bool
	// True if the function should be visible to other packages.
	IsExported     bool
	IsExtern       bool
	Vars           []*Variable
	parameterScope *CommandScope
}

// NewFunction ...
func NewFunction(name string, f *types.Func) *Function {
	fir := &Function{Func: f, Name: name}
	fir.Body.Op = OpBlock
	// The scope used for function parameters
	fir.parameterScope = newScope(nil)
	// The scope used for local variables (unless they are inside a nested scope)
	fir.Body.Scope = newScope(fir.parameterScope)
	return fir
}

// ToString ...
func (f *Function) ToString() string {
	str := fmt.Sprintf("func %v %v { // %v\n", f.Name, f.Func.Type.ToFunctionSignature(), strconv.Itoa(f.Body.Scope.ID))
	str += f.Body.ToString("")
	str += "}\n\n"
	return str
}
