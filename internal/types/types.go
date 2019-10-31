package types

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
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

// EqualTypeMode ...
type EqualTypeMode int

const (
	// Assignable ...
	Assignable EqualTypeMode = 1 + iota
	// Strict ...
	Strict
	// Comparable ...
	Comparable
)

// Type ...
type Type interface {
	Name() string
	SetName(string)
	Location() errlog.LocationRange
	Check(log *errlog.ErrorLog) error
	ToString() string
	Package() *Package
	Component() *ComponentType
}

// TypeBase ...
type TypeBase struct {
	location     errlog.LocationRange
	name         string
	typeChecked  bool
	funcsChecked bool
	// The package in which the type has been defined
	pkg       *Package
	component *ComponentType
}

// PrimitiveType ...
type PrimitiveType struct {
	TypeBase
}

// AliasType ...
type AliasType struct {
	TypeBase
	Alias Type
	Funcs []*Func
}

// StructType ...
type StructType struct {
	TypeBase
	BaseType   *StructType
	Interfaces []*InterfaceType
	Fields     []*StructField
	Funcs      []*Func
}

// StructField ...
type StructField struct {
	Name string
	Type Type
}

// ComponentType ...
type ComponentType struct {
	TypeBase
	Scope *Scope
}

// ComponentField ...
type ComponentField struct {
	Name string
	Type Type
}

// SliceType ...
type SliceType struct {
	TypeBase
	ElementType Type
}

// ArrayType ...
type ArrayType struct {
	TypeBase
	Size        uint64
	ElementType Type
}

// PointerType ...
type PointerType struct {
	TypeBase
	Mode        PointerMode
	ElementType Type
}

// InterfaceType ...
type InterfaceType struct {
	TypeBase
	BaseTypes []*InterfaceType
	Funcs     []*InterfaceFunc
}

// InterfaceFunc ...
type InterfaceFunc struct {
	// TODO: Why not use the name of FuncType?
	Name     string
	FuncType *FuncType
}

// ClosureType ...
type ClosureType struct {
	TypeBase
	FuncType *FuncType
}

// FuncType ...
type FuncType struct {
	TypeBase
	In  *ParameterList
	Out *ParameterList
	// Optional
	Target Type
	// Computed value
	returnType Type
}

// ParameterList ...
type ParameterList struct {
	Params []*Parameter
}

// Parameter ...
type Parameter struct {
	Location errlog.LocationRange
	Name     string
	Type     Type
}

// MutableType ...
type MutableType struct {
	TypeBase
	Mutable bool
	Type    Type
}

// GroupType ...
type GroupType struct {
	TypeBase
	// The group to which
	Group *Group
	Type  Type
}

// GenericType ...
type GenericType struct {
	TypeBase
	Type parser.Node
	// The scope in which this type has been defined.
	Scope          *Scope
	TypeParameters []*GenericTypeParameter
	Funcs          []*Func
}

// GenericTypeParameter ...
type GenericTypeParameter struct {
	Name string
}

// GenericInstanceType ...
type GenericInstanceType struct {
	TypeBase
	BaseType      *GenericType
	TypeArguments map[string]Type
	InstanceType  Type
	Funcs         []*Func
	// The scope containing the type arguments.
	// This scope is a child-scope of the scope in which the BaseType has been defined.
	Scope *Scope
	// Multiple equivalent instances of the same generic type can exist.
	// To avoid double code generation in a package, this pointer links to an equivalent.
	equivalent *GenericInstanceType
}

var intType = newPrimitiveType("int")
var int8Type = newPrimitiveType("int8")
var int16Type = newPrimitiveType("int16")
var int32Type = newPrimitiveType("int32")
var int64Type = newPrimitiveType("int64")
var uintType = newPrimitiveType("uint")
var uint8Type = newPrimitiveType("uint8")
var uint16Type = newPrimitiveType("uint16")
var uint32Type = newPrimitiveType("uint32")
var uint64Type = newPrimitiveType("uint64")
var uintptrType = newPrimitiveType("uintptr")
var float32Type = newPrimitiveType("float32")
var float64Type = newPrimitiveType("float64")
var boolType = newPrimitiveType("bool")
var byteType = uint8Type
var runeType = newPrimitiveType("rune")
var nullType = newPrimitiveType("null")
var stringType = newPrimitiveType("string")

var integerType = newPrimitiveType("integer_number")
var floatType = newPrimitiveType("float_number")
var voidType = newPrimitiveType("void")
var arrayLiteralType = newPrimitiveType("array_literal")
var structLiteralType = newPrimitiveType("struct_literal")
var namespaceType = newPrimitiveType("namespace")

var (
	// PrimitiveTypeInt ...
	PrimitiveTypeInt = intType
	// PrimitiveTypeInt8 ...
	PrimitiveTypeInt8 = int8Type
	// PrimitiveTypeInt16 ...
	PrimitiveTypeInt16 = int16Type
	// PrimitiveTypeInt32 ...
	PrimitiveTypeInt32 = int32Type
	// PrimitiveTypeInt64 ...
	PrimitiveTypeInt64 = int64Type
	// PrimitiveTypeUint ...
	PrimitiveTypeUint = uintType
	// PrimitiveTypeUint8 ...
	PrimitiveTypeUint8 = uint8Type
	// PrimitiveTypeUint16 ...
	PrimitiveTypeUint16 = uint16Type
	// PrimitiveTypeUint32 ...
	PrimitiveTypeUint32 = uint32Type
	// PrimitiveTypeUint64 ...
	PrimitiveTypeUint64 = uint64Type
	// PrimitiveTypeUintptr ...
	PrimitiveTypeUintptr = uintptrType
	// PrimitiveTypeFloat32 ...
	PrimitiveTypeFloat32 = float32Type
	// PrimitiveTypeFloat64 ...
	PrimitiveTypeFloat64 = float64Type
	// PrimitiveTypeBool ...
	PrimitiveTypeBool = boolType
	// PrimitiveTypeByte ...
	PrimitiveTypeByte = byteType
	// PrimitiveTypeRune ...
	PrimitiveTypeRune = runeType
	// PrimitiveTypeNull ...
	PrimitiveTypeNull = nullType
	// PrimitiveTypeString ...
	PrimitiveTypeString = stringType
	// PrimitiveTypeIntLiteral ...
	PrimitiveTypeIntLiteral = integerType
	// PrimitiveTypeFloatLiteral ...
	PrimitiveTypeFloatLiteral = floatType
	// PrimitiveTypeArrayLiteral ...
	PrimitiveTypeArrayLiteral = arrayLiteralType
	// PrimitiveTypeVoid ...
	PrimitiveTypeVoid = voidType
	// PrimitiveTypeNamespace ...
	PrimitiveTypeNamespace = namespaceType
)

// Name ...
func (t *TypeBase) Name() string {
	return t.name
}

// SetName ...
func (t *TypeBase) SetName(name string) {
	t.name = name
}

// Location ...
// Returns zero for global types such as int, byte, etc.
func (t *TypeBase) Location() errlog.LocationRange {
	return t.location
}

// IsTypedef ...
func (t *TypeBase) IsTypedef() bool {
	return t.name != ""
}

// ToString ...
func (t *TypeBase) ToString() string {
	return t.name
}

// Package ...
// Returns nil for global types such as int, byte, etc.
func (t *TypeBase) Package() *Package {
	return t.pkg
}

// Component ...
func (t *TypeBase) Component() *ComponentType {
	return t.component
}

func newPrimitiveType(name string) *PrimitiveType {
	return &PrimitiveType{TypeBase: TypeBase{name: name}}
}

// Check ...
func (t *PrimitiveType) Check(log *errlog.ErrorLog) error {
	t.typeChecked = true
	return nil
}

// HasMember ...
func (t *AliasType) HasMember(name string) bool {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return true
		}
	}
	return false
}

// Check ...
func (t *AliasType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	t2 := t.Alias
	for {
		t3, ok := t2.(*AliasType)
		if !ok {
			break
		}
		if t3 == t {
			return log.AddError(errlog.ErrorCyclicTypeDefinition, t.Location())
		}
		t2 = t3.Alias
	}
	t.Alias = t2

	for _, f := range t.Funcs {
		if err := f.Type.Check(log); err != nil {
			return err
		}
	}
	return t.Alias.Check(log)
}

// Check ...
func (t *ComponentType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	// TODO Check everything in the scope ...
	return nil
}

// Check ...
func (t *PointerType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// ToString ...
func (t *PointerType) ToString() string {
	return "*" + t.ElementType.ToString()
}

// Check ...
func (t *SliceType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// ToString ...
func (t *SliceType) ToString() string {
	return "[]" + t.ElementType.ToString()
}

// Check ...
func (t *ArrayType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// ToString ...
func (t *ArrayType) ToString() string {
	return "[" + strconv.FormatUint(t.Size, 10) + "]" + t.ElementType.ToString()
}

// Check ...
func (t *GroupType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
}

// ToString ...
func (t *GroupType) ToString() string {
	if t.Group.Kind == GroupIsolate {
		return "->" + t.Type.ToString()
	}
	return "-" + t.Group.Name + " " + t.Type.ToString()
}

// Check ...
func (t *MutableType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
}

// ToString ...
func (t *MutableType) ToString() string {
	if t.Mutable {
		return "mut " + t.Type.ToString()
	}
	return t.Type.ToString()
}

// HasMember ...
func (t *GenericType) HasMember(name string) bool {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return true
		}
	}
	return false
}

// Check ...
func (t *GenericType) Check(log *errlog.ErrorLog) error {
	t.typeChecked = true
	// Do nothing
	return nil
}

// HasMember ...
func (t *StructType) HasMember(name string) bool {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return true
		}
	}
	for _, f := range t.Fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// Check ...
func (t *StructType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	if t.BaseType != nil {
		if err := t.BaseType.Check(log); err != nil {
			return err
		}
	}
	for _, f := range t.Fields {
		if err := f.Type.Check(log); err != nil {
			return err
		}
	}
	for _, iface := range t.Interfaces {
		if err := iface.Check(log); err != nil {
			return err
		}
	}
	for _, f := range t.Funcs {
		if err := f.Type.Check(log); err != nil {
			return err
		}
	}
	return nil
}

// Func ...
func (t *StructType) Func(name string) *Func {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return f
		}
	}
	if t.BaseType != nil {
		return t.BaseType.Func(name)
	}
	return nil
}

// Field ...
func (t *StructType) Field(name string) *StructField {
	for _, f := range t.Fields {
		if f.Name == name {
			return f
		}
	}
	if t.BaseType != nil {
		return t.BaseType.Field(name)
	}
	return nil
}

/*
// FieldIndex ...
// If the field could not be found, FieldIndex returns a value < 0.
// Currently, it returns the negative number of fields of the struct minus 1
func (t *StructType) FieldIndex(f *StructField) int {
	index := -1
	if t.BaseType != nil {
		index = t.BaseType.FieldIndex(f)
		if index >= 0 {
			return index
		}
	}
	for i, f2 := range t.Fields {
		if f == f2 {
			return -index + 1 + i
		}
	}
	return index - len(t.Fields)
}
*/

// Check ...
func (t *InterfaceType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	for _, b := range t.BaseTypes {
		if err := b.Check(log); err != nil {
			return err
		}
	}
	for _, f := range t.Funcs {
		if err := f.FuncType.Check(log); err != nil {
			return err
		}
	}
	return nil
}

// Check ...
func (t *ClosureType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.FuncType.Check(log)
}

// Check ...
func (t *FuncType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	for _, p := range t.In.Params {
		if err := p.Type.Check(log); err != nil {
			return err
		}
	}
	for _, p := range t.Out.Params {
		if err := p.Type.Check(log); err != nil {
			return err
		}
	}
	if len(t.Out.Params) == 0 {
		t.returnType = PrimitiveTypeVoid
	} else if len(t.Out.Params) == 1 {
		t.returnType = t.Out.Params[0].Type
	} else {
		st := &StructType{TypeBase: TypeBase{location: t.TypeBase.location, pkg: t.TypeBase.pkg, component: t.TypeBase.component}}
		for i, p := range t.Out.Params {
			f := &StructField{Name: "f" + strconv.Itoa(i), Type: p.Type}
			st.Fields = append(st.Fields, f)
		}
		t.returnType = st
	}
	return nil
}

// ToFunctionSignature ...
func (t *FuncType) ToFunctionSignature() string {
	str := "("
	for i, p := range t.In.Params {
		if i > 0 {
			str += ", "
		}
		str += p.Name + " "
		str += p.Type.ToString()
	}
	str += ")"
	if len(t.Out.Params) > 0 {
		str += " ("
		for i, p := range t.Out.Params {
			if i > 0 {
				str += ", "
			}
			if p.Name != "" {
				str += p.Name + " "
			}
			str += p.Type.ToString()
		}
		str += ")"
	}
	return str
}

// ReturnType ...
func (t *FuncType) ReturnType() Type {
	return t.returnType
}

// HasNamedReturnVariables ...
func (t *FuncType) HasNamedReturnVariables() bool {
	if len(t.Out.Params) == 0 {
		return false
	}
	return t.Out.Params[0].Name != ""
}

// Check ...
func (t *GenericInstanceType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	// The generic has been instantiated before? Just copy over the functions
	if t.equivalent != nil {
		if err := t.equivalent.Check(log); err != nil {
			return err
		}
		t.Funcs = t.equivalent.Funcs
		return nil
	}
	for _, f := range t.BaseType.Funcs {
		tf, err := declareFunction(f.Ast, t.Scope, log)
		if err != nil {
			return err
		}
		tf.Component = f.Component
		if tf.DualIsMut {
			t.Scope.dualIsMut = -1
			tf2, err := declareFunction(f.Ast, t.Scope, log)
			t.Scope.dualIsMut = 0
			if err != nil {
				return err
			}
			tf2.Component = f.Component
			tf.DualFunc = tf2
		}
	}
	if t.InstanceType != nil {
		if err := t.InstanceType.Check(log); err != nil {
			return err
		}
	}
	for _, f := range t.Funcs {
		if err := f.Type.Check(log); err != nil {
			return err
		}
	}
	return nil
}

// HasMember ...
func (t *GenericInstanceType) HasMember(name string) bool {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return true
		}
	}
	return false
}

// ToString ...
func (t *GenericInstanceType) ToString() string {
	str := t.Name() + "<"
	for i, p := range t.BaseType.TypeParameters {
		a := t.TypeArguments[p.Name]
		if i > 0 {
			str += ","
		}
		str += a.ToString()
	}
	str += ">"
	return str
}

/*************************************************
 *
 * Helper functions
 *
 *************************************************/

func isEqualType(left Type, right Type, mode EqualTypeMode) bool {
	// Check groups (or ignore them)
	if mode == Comparable {
		// Groups do not matter for comparison
		if l, ok := left.(*GroupType); ok {
			left = l.Type
		}
		if r, ok := right.(*GroupType); ok {
			right = r.Type
		}
	} else {
		// Groups do not matter for comparison
		l, lok := left.(*GroupType)
		r, rok := right.(*GroupType)
		if lok != rok {
			return false
		}
		if lok {
			if l.Group.Kind != r.Group.Kind {
				return false
			}
			if l.Group.Kind == GroupNamed && l.Group.Name != r.Group.Name {
				return false
			}
			left = l.Type
			right = r.Type
		}
	}

	// Check mutability
	if mode == Comparable {
		if l, ok := left.(*MutableType); ok {
			left = l.Type
		}
		if r, ok := right.(*MutableType); ok {
			right = r.Type
		}
	} else if mode == Assignable {
		l, okl := left.(*MutableType)
		okl = okl && l.Mutable
		r, okr := right.(*MutableType)
		okr = okr && r.Mutable
		if okr && !okl {
			right = r.Type
		}
	}

	// Trivial case
	if left == right {
		return true
	}
	// Compare types
	switch l := left.(type) {
	case *PointerType:
		r, ok := right.(*PointerType)
		if !ok {
			return false
		}
		return isEqualType(l.ElementType, r.ElementType, mode)
	case *SliceType:
		r, ok := right.(*SliceType)
		if !ok {
			return false
		}
		return isEqualType(l.ElementType, r.ElementType, mode)
	case *ArrayType:
		r, ok := right.(*ArrayType)
		if !ok {
			return false
		}
		return l.Size == r.Size && isEqualType(l.ElementType, r.ElementType, mode)
	case *StructType:
		return false
	case *MutableType:
		r, ok := right.(*MutableType)
		if !ok || l.Mutable != r.Mutable {
			return false
		}
		return isEqualType(l.Type, r.Type, mode)
	case *PrimitiveType:
		return false
	default:
		panic("Ooops")
	}
}

func checkEqualType(left Type, right Type, mode EqualTypeMode, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if isEqualType(left, right, mode) {
		return nil
	}
	return log.AddError(errlog.ErrorIncompatibleTypes, loc)
}

// IsSliceType ...
func IsSliceType(t Type) bool {
	switch t2 := t.(type) {
	case *SliceType:
		return true
	case *MutableType:
		return IsSliceType(t2.Type)
	case *GroupType:
		return IsSliceType(t2.Type)
	}
	return false
}

// IsPointerType ...
func IsPointerType(t Type) bool {
	switch t2 := t.(type) {
	case *PointerType:
		return true
	case *MutableType:
		return IsPointerType(t2.Type)
	case *GroupType:
		return IsPointerType(t2.Type)
	}
	return false
}

// IsArrayType ...
func IsArrayType(t Type) bool {
	if t == arrayLiteralType {
		return true
	}
	switch t2 := t.(type) {
	case *ArrayType:
		return true
	case *MutableType:
		return IsArrayType(t2.Type)
	case *GroupType:
		return IsArrayType(t2.Type)
	}
	return false
}

// IsIntegerType ...
func IsIntegerType(t Type) bool {
	return t == integerType || t == uintptrType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type || t == uintType || t == uint8Type || t == uint16Type || t == uint32Type || t == uint64Type
}

// IsSignedIntegerType ...
func IsSignedIntegerType(t Type) bool {
	return t == integerType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type
}

// IsUnsignedIntegerType ...
func IsUnsignedIntegerType(t Type) bool {
	return t == integerType || t == uintptrType || t == uintType || t == uint8Type || t == uint16Type || t == uint32Type || t == uint64Type
}

// IsFloatType ...
func IsFloatType(t Type) bool {
	return t == floatType || t == float32Type || t == float64Type
}

// IsUnsafePointerType ...
func IsUnsafePointerType(t Type) bool {
	switch t2 := t.(type) {
	case *PointerType:
		return t2.Mode == PtrUnsafe
	case *MutableType:
		return IsUnsafePointerType(t2.Type)
	case *GroupType:
		return IsUnsafePointerType(t2.Type)
	}
	return false
}

// GetArrayType ...
func GetArrayType(t Type) (*ArrayType, bool) {
	switch t2 := t.(type) {
	case *ArrayType:
		return t2, true
	case *MutableType:
		return GetArrayType(t2.Type)
	case *GroupType:
		return GetArrayType(t2.Type)
	}
	return nil, false
}

// GetSliceType ...
func GetSliceType(t Type) (*SliceType, bool) {
	switch t2 := t.(type) {
	case *SliceType:
		return t2, true
	case *MutableType:
		return GetSliceType(t2.Type)
	case *GroupType:
		return GetSliceType(t2.Type)
	}
	return nil, false
}

// GetStructType ...
func GetStructType(t Type) (*StructType, bool) {
	switch t2 := t.(type) {
	case *StructType:
		return t2, true
	case *MutableType:
		return GetStructType(t2.Type)
	case *GroupType:
		return GetStructType(t2.Type)
	}
	return nil, false
}

// GetPointerType ...
func GetPointerType(t Type) (*PointerType, bool) {
	switch t2 := t.(type) {
	case *PointerType:
		return t2, true
	case *MutableType:
		return GetPointerType(t2.Type)
	case *GroupType:
		return GetPointerType(t2.Type)
	}
	return nil, false
}

// GetFuncType ...
func GetFuncType(t Type) (*FuncType, bool) {
	switch t2 := t.(type) {
	case *FuncType:
		return t2, true
	case *MutableType:
		return GetFuncType(t2.Type)
	case *GroupType:
		return GetFuncType(t2.Type)
	}
	return nil, false
}

// TypeHasPointers ...
func TypeHasPointers(t Type) bool {
	switch t2 := t.(type) {
	case *AliasType:
		return TypeHasPointers(t2.Alias)
	case *MutableType:
		return TypeHasPointers(t2.Type)
	case *GroupType:
		return TypeHasPointers(t2.Type)
	case *ArrayType:
		return TypeHasPointers(t2.ElementType)
	case *PointerType:
		// Unsafe pointers are treated like integers.
		if t2.Mode != PtrUnsafe {
			return true
		}
	case *SliceType:
		return true
	case *StructType:
		for _, f := range t2.Fields {
			if TypeHasPointers(f.Type) {
				return true
			}
		}
	}
	return false
}
