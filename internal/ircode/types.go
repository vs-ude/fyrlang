package ircode

/*
import (
	"fmt"
)

// PointerMode ...
type PointerMode int

const (
	// PtrOwner ...
	PtrOwner PointerMode = 1 + iota
	// PtrWeakRef ...
	PtrWeakRef
	// PtrStrongRef ...
	PtrStrongRef
	// PtrBorrow ...
	PtrBorrow
	// PtrUnsafe ...
	PtrUnsafe
	// PtrIsolatedGroup points to isolated group on the heap
	PtrIsolatedGroup
)

// IType ...
type IType interface {
	ToString() string
}

// Type ...
type Type struct {
	Name string
}

// BasicType ...
type BasicType struct {
	Type
}

// StructField ...
type StructField struct {
	Name string
	Type IType
}

// StructType ...
type StructType struct {
	Type
	Fields []*StructField
}

// FunctionType ...
type FunctionType struct {
	Type
	// Return type of the function.
	Return IType
	Params []VariableUsage
}

// PointerType ...
type PointerType struct {
	Type
	Element IType
	Mode    PointerMode
}

// SliceType ...
type SliceType struct {
	Type
	Array *ArrayType
	Mode  PointerMode
}

// ArrayType ...
type ArrayType struct {
	Type
	Element IType
	// -1 means that the size of the array is not known, because it was dynamically allocated on the heap.
	Size int
}

// InterfacePointerType ...
type InterfacePointerType struct {
	Type
	Mode PointerMode
}

// VoidType ...
var VoidType = &BasicType{Type: Type{Name: "void"}}

// IntType ...
var IntType = &BasicType{Type: Type{Name: "int"}}

// Float64Type ...
var Float64Type = &BasicType{Type: Type{Name: "float64"}}

// BoolType ...
var BoolType = &BasicType{Type: Type{Name: "bool"}}

// StringType ...
var StringType = &BasicType{Type: Type{Name: "string"}}

// AnyType ...
var AnyType = &BasicType{Type: Type{Name: "any"}}

// NewFunctionType ...
func NewFunctionType() *FunctionType {
	return &FunctionType{}
}

// ToString ...
func (t *BasicType) ToString() string {
	return t.Name
}

// ToString ...
func (t *PointerType) ToString() string {
	return t.Mode.ToString() + " " + t.Element.ToString()
}

// ToString ...
func (t *SliceType) ToString() string {
	return t.Mode.ToString() + " " + t.Array.ToString()
}

// ToString ...
func (t *ArrayType) ToString() string {
	if t.Size == -1 {
		return "[]" + t.Element.ToString()
	}
	return fmt.Sprintf("[%v]%v", t.Size, t.Element.ToString())
}

// ToString ...
func (t *InterfacePointerType) ToString() string {
	if t.Name == "" {
		return t.Mode.ToString() + "interface{}"
	}
	return t.Mode.ToString() + "interface{" + t.Name + "}"
}

// ToString ...
func (t *FunctionType) ToString() string {
	return "func " + t.ToFunctionSignature()
}

// ToFunctionSignature ...
func (t *FunctionType) ToFunctionSignature() string {
	str := "("
	for i, p := range t.Params {
		if i != 0 {
			str += ", "
		}
		if p.Group != nil {
			str += "`" + p.Group.Name + " "
		}
		str += p.Var.Name + " " + p.Var.Type.ToString()
	}
	str += ")"
	if t.Return != nil {
		str += " " + t.Return.ToString()
	}
	return str
}

// NewPointerType ...
func NewPointerType(element IType, mode PointerMode) *PointerType {
	return &PointerType{Element: element, Mode: mode}
}

// NewSliceType ...
func NewSliceType(element IType, size int, mode PointerMode) *SliceType {
	return &SliceType{Array: &ArrayType{Element: element, Size: size}, Mode: mode}
}

// NewArrayType ...
func NewArrayType(element IType, size int) *ArrayType {
	return &ArrayType{Element: element, Size: size}
}

// NewStructType ...
func NewStructType(name string) *StructType {
	return &StructType{Type: Type{Name: name}}
}

// AddField ...
func (t *StructType) AddField(name string, fieldType IType) *StructType {
	t.Fields = append(t.Fields, &StructField{Name: name, Type: fieldType})
	return t
}

// ToString ...
func (t *StructType) ToString() string {
	if t.Type.Name[0] == '%' {
		return t.Type.Name
	}
	str := "struct {"
	for i, f := range t.Fields {
		if i > 0 {
			str += ", "
		}
		str += f.Type.ToString()
	}
	return str
}

// FieldIndex ...
func (t *StructType) FieldIndex(f *StructField) int {
	for i, f2 := range t.Fields {
		if f == f2 {
			return i
		}
	}
	panic("Unknown field")
}

// CompareTypes ...
func CompareTypes(t1, t2 IType) bool {
	if t1 == t2 {
		return true
	}
	switch p1 := t1.(type) {
	case *PointerType:
		if p2, ok := t2.(*PointerType); ok {
			return CompareTypes(p1.Element, p2.Element)
		}
		break
	case *InterfacePointerType:
		if _, ok := t2.(*InterfacePointerType); ok {
			return true
		}
		break
	case *ArrayType:
		if p2, ok := t2.(*ArrayType); ok && p1.Size == p2.Size {
			return CompareTypes(p1.Element, p2.Element)
		}
		break
	case *SliceType:
		if p2, ok := t2.(*SliceType); ok {
			return CompareTypes(p1.Array, p2.Array)
		}
		break
	}
	return false
}

// ToString ...
func (m PointerMode) ToString() string {
	var str string
	switch m {
	case PtrOwner:
		str = "*"
		break
	case PtrWeakRef:
		str = "~"
		break
	case PtrStrongRef:
		str = "$"
		break
	case PtrBorrow:
		str = "&"
		break
	case PtrUnsafe:
		str = "#"
		break
	}
	return str
}

// IsBorrowedType ...
func IsBorrowedType(t IType) bool {
	switch r := t.(type) {
	case *PointerType:
		return r.Mode == PtrBorrow
	case *SliceType:
		return r.Mode == PtrBorrow
	case *InterfacePointerType:
		return r.Mode == PtrBorrow
	}
	return false
}

// IsIsolatedGroupType returns true if the type is a pointer or pointer-like type
// and it points to elements in another group.
func IsIsolatedGroupType(t IType) bool {
	switch r := t.(type) {
	case *PointerType:
		return r.Mode == PtrIsolatedGroup
	case *SliceType:
		return r.Mode == PtrIsolatedGroup
	case *InterfacePointerType:
		return r.Mode == PtrIsolatedGroup
	}
	return false
}

// IsPureValueType ...
func IsPureValueType(t IType) bool {
	switch r := t.(type) {
	case *PointerType:
		return r.Mode == PtrUnsafe
	case *SliceType:
		return r.Mode == PtrUnsafe
	case *InterfacePointerType:
		return r.Mode == PtrUnsafe
	case *FunctionType:
		return false
	case *ArrayType:
		return IsPureValueType(r.Element)
	case *StructType:
		for _, f := range r.Fields {
			if !IsPureValueType(f.Type) {
				return false
			}
		}
		return true
	}
	return true
}
*/
