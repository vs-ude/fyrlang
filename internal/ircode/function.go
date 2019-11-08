package ircode

import (
	"fmt"
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
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
	IsExported bool
	IsExtern   bool
	// A list of all variables used in the function, including parameters.
	// This list is populated by the `Builder`.
	Vars           []*Variable
	parameterScope *CommandScope
	functionType   *FunctionType
}

// FunctionType is the IR-code representation of a types.FunctionType.
type FunctionType struct {
	In  []*FunctionParameter
	Out []*FunctionParameter
	// Names of the group parameters that are passed along with a function
	GroupParameters []*types.Group
	// Computed value
	returnType types.Type
	funcType   *types.FuncType
}

// FunctionParameter ...
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
	if ft.Target != nil {
		// et := types.NewExprType(ft.Target)
		fp := &FunctionParameter{Name: "this", Type: ft.Target, Location: ft.Target.Location()}
		t.In = append(t.In, fp)
		t.addGroupParameter(fp, 0)
		/*
			if et.PointerDestMutable {
				groupType := &types.PointerType{ElementType: types.PrimitiveTypeUintptr, Mode: types.PtrUnsafe}
				fp = &FunctionParameter{Name: "g_this", Type: groupType, Location: ft.Target.Location()}
				t.In = append(t.In, fp)
			}
		*/
	}
	for _, p := range ft.In.Params {
		fp := &FunctionParameter{Name: p.Name, Type: p.Type, Location: p.Location}
		t.In = append(t.In, fp)
		t.addGroupParameter(fp, len(t.In))
	}
	for i, p := range ft.Out.Params {
		fp := &FunctionParameter{Name: p.Name, Type: p.Type, Location: p.Location}
		t.Out = append(t.Out, fp)
		t.addGroupParameter(fp, i)
	}
	return t
}

// Function parameters with pointers require an additional parameter (a group parameter)
// to pass information about the group of this parameter to a function.
func (t *FunctionType) addGroupParameter(p *FunctionParameter, pos int) {
	if types.TypeHasPointers(p.Type) {
		et := types.NewExprType(p.Type)
		if et.PointerDestGroup != nil {
			if et.PointerDestGroup.Kind == types.GroupIsolate {
				if p.Name == "" {
					// Return parameters can have no name. Construct one.
					et.PointerDestGroup.Name = strconv.Itoa(pos) + "ret"
				} else {
					et.PointerDestGroup.Name = p.Name
				}
				t.GroupParameters = append(t.GroupParameters, et.PointerDestGroup)
			} else {
				found := false
				for _, g := range t.GroupParameters {
					if g == et.PointerDestGroup {
						found = true
						break
					}
				}
				if !found {
					t.GroupParameters = append(t.GroupParameters, et.PointerDestGroup)
				}
			}
		} else {
			g := &types.Group{Kind: types.GroupNamed, Name: p.Name, Location: p.Location}
			if p.Name == "" {
				// Return parameters can have no name. Construct one.
				g.Name = strconv.Itoa(pos) + "ret"
			}
			p.Type = &types.GroupType{Group: g, Type: p.Type}
			t.GroupParameters = append(t.GroupParameters, g)
		}
	}
}

// ReturnType ...
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
			f := &types.StructField{Name: "f" + strconv.Itoa(i), Type: p.Type}
			st.Fields = append(st.Fields, f)
		}
		t.returnType = st
	}
	return t.returnType
}
