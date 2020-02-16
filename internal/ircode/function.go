package ircode

import (
	"fmt"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Function is an ircode representation of a type-checked function.
type Function struct {
	// Name of the function as seen by the linker.
	// Thus, this name is potentially mangled.
	Name string
	// The type-checked function from which this ircode function has been generated.
	Func *types.Func
	// This command is only required to hold a list of commands in its Branch field.
	Body Command
	// Functions that are instantiated from a generic need to be de-duplicated when linking.
	IsGenericInstance bool
	// True if the function should be visible to other packages.
	IsExported bool
	// True if the function has been exported from some external package.
	IsExtern bool
	// A list of all variables used in the function, including parameters.
	// This list is populated by the `Builder`.
	Vars []*Variable
	// A list of all ircode variables which correspond to function parameters.
	InVars []*Variable
	// A list of all ircode variables corresponding to named return parameters.
	OutVars        []*Variable
	parameterScope *CommandScope
	functionType   *FunctionType
}

// FunctionType is the IR-code representation of a types.FunctionType.
// Member functions are no longer treated special at thus point.
// Their `this` pointer becomes the firs input parameter.
type FunctionType struct {
	In  []*FunctionParameter
	Out []*FunctionParameter
	// Group specifiers that are passed along with a function call.
	GroupSpecifiers []*types.GroupSpecifier
	// Computed value
	returnType types.Type
	funcType   *types.FuncType
}

// FunctionParameter is the ircode representation of a types.Parameter.
type FunctionParameter struct {
	Location errlog.LocationRange
	Name     string
	Type     types.Type
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

// Type ...
func (f *Function) Type() *FunctionType {
	if f.functionType == nil {
		f.functionType = NewFunctionType(f.Func.Type)
	}
	return f.functionType
}

// NewFunctionType ...
func NewFunctionType(ft *types.FuncType) *FunctionType {
	t := &FunctionType{funcType: ft}
	// Create an ircode representation of the function's `this` parameter (if the function is a method)
	if ft.Target != nil {
		// Turn 'this' into the first parameter expected by the function
		fp := &FunctionParameter{Name: "this", Type: ft.Target, Location: ft.Target.Location()}
		t.In = append(t.In, fp)
		t.addGroupSpecifier(fp, 0)
	}
	// Create an ircode representation of the function's in and out parameters (and group specifiers)
	for _, p := range ft.In.Params {
		fp := &FunctionParameter{Name: p.Name, Type: p.Type, Location: p.Location}
		t.In = append(t.In, fp)
		t.addGroupSpecifier(fp, len(t.In))
	}
	for i, p := range ft.Out.Params {
		fp := &FunctionParameter{Name: p.Name, Type: p.Type, Location: p.Location}
		t.Out = append(t.Out, fp)
		t.addGroupSpecifier(fp, i)
	}
	return t
}

func (t *FunctionType) addGroupSpecifier(p *FunctionParameter, pos int) {
	if types.TypeHasPointers(p.Type) {
		et := types.NewExprType(p.Type)
		if et.PointerDestGroupSpecifier == nil {
			panic("Oooops")
		}
		// Avoid duplicates
		for _, g := range t.GroupSpecifiers {
			if g == et.PointerDestGroupSpecifier {
				return
			}
		}
		t.GroupSpecifiers = append(t.GroupSpecifiers, et.PointerDestGroupSpecifier)
	}
}

// ReturnType returns the effective return type used in the ircode.
// In ircode, a function can return only one value.
// Hence, multiple return values are represented as one return value of type struct.
func (t *FunctionType) ReturnType() types.Type {
	if t.returnType != nil {
		return t.returnType
	}
	if len(t.Out) == 0 {
		t.returnType = types.PrimitiveTypeVoid
	} else if len(t.Out) == 1 {
		t.returnType = t.Out[0].Type
	} else {
		st := &types.StructType{TypeBase: t.funcType.TypeBase}
		st.SetName("")
		for i, p := range t.Out {
			f := &types.StructField{Name: "f" + strconv.Itoa(i), Type: types.RemoveGroup(p.Type)}
			st.Fields = append(st.Fields, f)
		}
		t.returnType = st
	}
	return t.returnType
}
