package types

import (
	"strconv"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// PointerMode ...
type PointerMode int

const (
	// PtrReference ...
	PtrReference PointerMode = iota
	// PtrOwner ...
	PtrOwner
	// PtrUnsafe ...
	PtrUnsafe
)

// EqualTypeMode ...
type EqualTypeMode int

const (
	// Assignable ...
	Assignable EqualTypeMode = 1 + iota
	// PointerAssignable ...
	PointerAssignable
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
	SetComponent(cmp *ComponentType)
	Scope() *Scope
	SetScope(s *Scope)
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
	// The scope in which the type has been defined.
	// This is either a file scope or a component scope.
	scope *Scope
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
	BaseType     *StructType
	Interfaces   []*InterfaceType
	Fields       []*StructField
	Funcs        []*Func
	IsConcurrent bool
}

// StructField ...
type StructField struct {
	Name string
	Type Type
	// Set to true of the field represents the basetype of a struct
	IsBaseType bool
}

// UnionType ...
type UnionType struct {
	TypeBase
	Fields []*StructField
	Funcs  []*Func
}

// ComponentType ...
type ComponentType struct {
	TypeBase
	// The scope that contains all types and elements defined in the component
	ComponentScope *Scope
	Interfaces     []*InterfaceType
	Fields         []*ComponentField
	Funcs          []*Func
	// Static components have no init func of their own.
	// This func is listed in `Funcs` as well.
	InitFunc *Func
	// All components used by this component
	ComponentsUsed []*ComponentUsage
	IsStatic       bool
}

// ComponentField ...
type ComponentField struct {
	Var        *Variable
	Expression *parser.VarExpressionNode
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
	// Optional
	GroupSpecifier *GroupSpecifier
	Mutable        bool
	Mode           PointerMode
	ElementType    Type
}

// InterfaceType ...
type InterfaceType struct {
	TypeBase
	BaseTypes            []*InterfaceType
	Funcs                []*InterfaceFunc
	IsComponentInterface bool
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
	Target       Type
	IsDestructor bool
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

// QualifiedType ...
type QualifiedType struct {
	TypeBase
	Volatile bool
	Const    bool
	Type     Type
}

// GenericType ...
type GenericType struct {
	TypeBase
	Type           parser.Node
	TypeParameters []*GenericTypeParameter
	Funcs          []*Func
}

// GenericTypeParameter ...
type GenericTypeParameter struct {
	// Set to true in the case of ```MyGeneric<`x>```.
	IsGroupSpecfier bool
	Name            string
}

// GenericInstanceType ...
type GenericInstanceType struct {
	TypeBase
	BaseType                *GenericType
	TypeArguments           map[string]Type
	GroupSpecifierArguments map[string]*GroupSpecifier
	InstanceType            Type
	Funcs                   []*Func
	// The scope containing the type arguments.
	// This scope is a child-scope of the scope in which the BaseType has been defined.
	GenericScope *Scope
	// Multiple equivalent instances of the same generic type can exist.
	// To avoid double code generation in a package, this pointer can link to an equivalent.
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

// SetComponent ...
func (t *TypeBase) SetComponent(c *ComponentType) {
	t.component = c
}

// Scope ...
func (t *TypeBase) Scope() *Scope {
	return t.scope
}

// SetScope ...
func (t *TypeBase) SetScope(s *Scope) {
	t.scope = s
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

// Func ...
func (t *AliasType) Func(name string) *Func {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return f
		}
	}
	return nil
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
	// Nothing else to check here. This happened in `file.go` already.
	return nil
}

// ToString ...
func (t *ComponentType) ToString() string {
	return "component" + t.name
}

// UsesComponent ...
func (t *ComponentType) UsesComponent(c *ComponentType, recursive bool) bool {
	if c == t {
		return true
	}
	for _, u := range t.ComponentsUsed {
		if u.Type == c {
			return true
		}
		if recursive {
			if u.Type.UsesComponent(c, true) {
				return true
			}
		}
	}
	return false
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
	str := ""
	if t.GroupSpecifier != nil {
		str += "`" + t.GroupSpecifier.ToString() + " "
	}
	if t.Mutable {
		str += "mut "
	}
	switch t.Mode {
	case PtrOwner:
		str += "*"
	case PtrReference:
		str += "&"
	case PtrUnsafe:
		str += "#"
	}
	return str + t.ElementType.ToString()
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
func (t *QualifiedType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
}

// ToString ...
func (t *QualifiedType) ToString() string {
	if t.Volatile {
		return "volatile " + t.Type.ToString()
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

// ToString ...
func (t *StructType) ToString() string {
	if t.Name() != "" {
		return t.Name()
	}
	str := "struct {"
	for _, f := range t.Fields {
		str += f.Name + " " + f.Type.ToString() + "; "
	}
	str += "}"
	return str
}

// HasBaseType ...
func (t *StructType) HasBaseType(b *StructType) bool {
	if t.BaseType == b {
		return true
	}
	if t.BaseType != nil {
		return t.BaseType.HasBaseType(b)
	}
	return false
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

// Destructor ...
func (t *StructType) Destructor() *Func {
	for _, f := range t.Funcs {
		if f.Name() == "__dtor__" {
			return f
		}
	}
	return nil
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
	if NeedsDestructor(t) && t.Destructor() == nil {
		// Create a destructor
		target := &PointerType{GroupSpecifier: NewGroupSpecifier("this", t.location), Mutable: true, Mode: PtrOwner, ElementType: t}
		ft := &FuncType{TypeBase: TypeBase{location: t.location, component: t.Component(), pkg: t.Package()}, IsDestructor: true, Target: target, In: &ParameterList{}, Out: &ParameterList{}}
		f := &Func{name: "__dtor__", Component: t.Component(), Type: ft, OuterScope: t.Scope()}
		f.InnerScope = newScope(f.OuterScope, FunctionScope, f.Location)
		f.InnerScope.Func = f
		vthis := &Variable{name: "this", Type: NewExprType(ft.Target)}
		f.InnerScope.AddElement(vthis, t.location, log)
		t.Funcs = append(t.Funcs, f)
		pkg := t.Package()
		pkg.Funcs = append(pkg.Funcs, f)
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

// FieldChain ...
func (t *StructType) FieldChain(name string) []*StructField {
	for _, f := range t.Fields {
		if f.Name == name {
			return []*StructField{f}
		}
	}
	if t.BaseType != nil {
		f := t.BaseType.FieldChain(name)
		if f == nil {
			return nil
		}
		// The first field is always the BaseType field
		return append([]*StructField{t.Fields[0]}, f...)
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

// ToString ...
func (t *UnionType) ToString() string {
	if t.Name() != "" {
		return t.Name()
	}
	str := "union {"
	for _, f := range t.Fields {
		str += f.Name + " " + f.Type.ToString() + "; "
	}
	str += "}"
	return str
}

// HasMember ...
func (t *UnionType) HasMember(name string) bool {
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
func (t *UnionType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	for _, f := range t.Fields {
		if err := f.Type.Check(log); err != nil {
			return err
		}
		if TypeHasPointers(f.Type) {
			return log.AddError(errlog.ErrorPointerInUnion, f.Type.Location())
		}
	}
	for _, f := range t.Funcs {
		if err := f.Type.Check(log); err != nil {
			return err
		}
	}
	return nil
}

// Field ...
func (t *UnionType) Field(name string) *StructField {
	for _, f := range t.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// Func ...
func (t *UnionType) Func(name string) *Func {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

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
	// Expand all functions associated with the generic type
	for _, f := range t.BaseType.Funcs {
		fn := f.Ast.Clone().(*parser.FuncNode)
		tf, err := declareFunction(fn, t.GenericScope, log)
		if err != nil {
			return err
		}
		tf.Component = f.Component
		if tf.DualIsMut {
			t.GenericScope.dualIsMut = -1
			fn := f.Ast.Clone().(*parser.FuncNode)
			tf2, err := declareFunction(fn, t.GenericScope, log)
			t.GenericScope.dualIsMut = 0
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
		if f.DualFunc != nil {
			if err := f.DualFunc.Type.Check(log); err != nil {
				return err
			}
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

// Func ...
func (t *GenericInstanceType) Func(name string) *Func {
	for _, f := range t.Funcs {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

// ToString ...
func (t *GenericInstanceType) ToString() string {
	str := t.Name() + "<"
	for i, p := range t.BaseType.TypeParameters {
		a := t.TypeArguments[p.Name]
		if i > 0 {
			str += ","
		}
		if p.IsGroupSpecfier {
			str += "`" + p.Name
		} else {
			str += a.ToString()
		}
	}
	str += ">"
	return str
}

/*************************************************
 *
 * Helper functions
 *
 *************************************************/

// Determines whether the two types can be assigned or compared.
// No error is written to the error log in case the test result is negative.
func isEqualType(left Type, right Type, mode EqualTypeMode) bool {
	// If the left type does not have a qualifier and the right type
	_, okl := left.(*QualifiedType)
	_, okr := right.(*QualifiedType)
	if !okl && okr {
		left = &QualifiedType{Type: left}
	} else if okl && !okr {
		right = &QualifiedType{Type: right}
	}

	// A pointer to a struct of Type B can be assigned a pointer to a struct of Type A if B has the basetype A.
	if mode == PointerAssignable {
		if lst, ok := left.(*StructType); ok {
			if rst, ok := right.(*StructType); ok {
				if rst.HasBaseType(lst) {
					return true
				}
			}
		}
	}

	// Compare types
	switch l := left.(type) {
	case *PointerType:
		r, ok := right.(*PointerType)
		if !ok {
			return false
		}
		if mode == Assignable {
			mode = PointerAssignable
		}
		if mode == Assignable || mode == PointerAssignable {
			if l.Mutable && !r.Mutable {
				return false
			}
			if r.Mode == PtrUnsafe && l.Mode != PtrUnsafe {
				return false
			}
			/*
				if l.Mode == PtrOwner && r.Mode != PtrOwner {
					return false
				}
				if l.Mode == PtrReference && (r.Mode != PtrOwner && r.Mode != PtrReference) {
					return false
				}
			*/
		} else if mode == Strict {
			if l.Mutable != r.Mutable {
				return false
			}
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
		// Both structs must be the same
		if l != right {
			return false
		}
		// Owning pointers cannot be copied. They must be taken
		if TypeHasOwningPointers(l) {
			return false
		}
		return true
	case *QualifiedType:
		r, ok := right.(*QualifiedType)
		if !ok {
			// Should have been handled above
			panic("Oooops")
		}
		// Cannot assign to a constant value
		if mode == Assignable || mode == PointerAssignable && l.Const {
			return false
		}
		if mode == PointerAssignable {
			if !l.Const && r.Const {
				return false
			}
			if !l.Volatile && r.Volatile {
				return false
			}
		} else if mode == Strict {
			if l.Const != r.Const || l.Volatile != r.Volatile {
				return false
			}
		}
		return isEqualType(l.Type, r.Type, mode)
	case *PrimitiveType:
		// Both primitive types must be the same
		if left != right {
			return false
		}
		return true
	case *AliasType:
		if l != right {
			return false
		}
		return true
	case *GenericInstanceType:
		g, ok := right.(*GenericInstanceType)
		if ok && l.InstanceType == g.InstanceType {
			return true
		}
		return false
	case *FuncType:
		r, ok := right.(*FuncType)
		if !ok || len(r.In.Params) != len(l.In.Params) || len(r.Out.Params) != len(l.Out.Params) || (l.Target == nil) != (r.Target == nil) {
			return false
		}
		for i := range r.In.Params {
			if !isEqualType(l.In.Params[i].Type, r.In.Params[i].Type, Strict) {
				return false
			}
		}
		for i := range r.Out.Params {
			if !isEqualType(l.Out.Params[i].Type, r.Out.Params[i].Type, Strict) {
				return false
			}
		}
		if l.Target != nil {
			if !isEqualType(l.Target, r.Target, Strict) {
				return false
			}
		}
		return true
	default:
		panic("Ooops")
	}
}

// Determines whether the two types can be assigned or compared.
// Logs an error, if the test fails.
func checkEqualType(left Type, right Type, mode EqualTypeMode, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if isEqualType(left, right, mode) {
		return nil
	}
	return log.AddError(errlog.ErrorIncompatibleTypes, loc)
}

// IsSliceType ...
func IsSliceType(t Type) bool {
	t = StripType(t)
	_, ok := t.(*SliceType)
	return ok
}

// IsPointerType ...
func IsPointerType(t Type) bool {
	t = StripType(t)
	_, ok := t.(*PointerType)
	return ok
}

// IsArrayType ...
func IsArrayType(t Type) bool {
	t = StripType(t)
	if t == arrayLiteralType {
		return true
	}
	_, ok := t.(*ArrayType)
	return ok
}

// IsStructType ...
func IsStructType(t Type) bool {
	t = StripType(t)
	if t == structLiteralType {
		return true
	}
	_, ok := t.(*StructType)
	return ok
}

// IsIntegerType ...
func IsIntegerType(t Type) bool {
	t = StripType(t)
	return t == integerType || t == uintptrType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type || t == uintType || t == uint8Type || t == uint16Type || t == uint32Type || t == uint64Type
}

// IsSignedIntegerType ...
func IsSignedIntegerType(t Type) bool {
	t = StripType(t)
	return t == integerType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type
}

// IsUnsignedIntegerType ...
func IsUnsignedIntegerType(t Type) bool {
	t = StripType(t)
	return t == integerType || t == uintptrType || t == uintType || t == uint8Type || t == uint16Type || t == uint32Type || t == uint64Type
}

// IsFloatType ...
func IsFloatType(t Type) bool {
	t = StripType(t)
	return t == floatType || t == float32Type || t == float64Type
}

// IsStringType ...
func IsStringType(t Type) bool {
	t = StripType(t)
	return t == stringType
}

// IsUnsafePointerType ...
func IsUnsafePointerType(t Type) bool {
	t = StripType(t)
	ptr, ok := t.(*PointerType)
	return ok && ptr.Mode == PtrUnsafe
}

// IsMutablePointerType ...
func IsMutablePointerType(t Type) bool {
	t = StripType(t)
	ptr, ok := t.(*PointerType)
	return ok && ptr.Mutable
}

// IsSlicePointerType ...
func IsSlicePointerType(t Type) bool {
	t = StripType(t)
	ptr, ok := t.(*PointerType)
	if !ok {
		return false
	}
	return IsSliceType(ptr.ElementType)
}

// IsFuncType ...
func IsFuncType(t Type) bool {
	t = StripType(t)
	_, ok := t.(*FuncType)
	return ok
}

// GetArrayType ...
func GetArrayType(t Type) (*ArrayType, bool) {
	t = StripType(t)
	t2, ok := t.(*ArrayType)
	return t2, ok
}

// GetSliceType ...
func GetSliceType(t Type) (*SliceType, bool) {
	t = StripType(t)
	t2, ok := t.(*SliceType)
	return t2, ok
}

// GetSlicePointerType ...
func GetSlicePointerType(t Type) (*PointerType, *SliceType, bool) {
	t = StripType(t)
	ptr, ok := t.(*PointerType)
	if !ok {
		return nil, nil, false
	}
	s, ok := GetSliceType(ptr.ElementType)
	return ptr, s, ok
}

// GetStructType ...
func GetStructType(t Type) (*StructType, bool) {
	t = StripType(t)
	t2, ok := t.(*StructType)
	return t2, ok
}

// GetUnionType ...
func GetUnionType(t Type) (*UnionType, bool) {
	t = StripType(t)
	t2, ok := t.(*UnionType)
	return t2, ok
}

// GetPointerType ...
func GetPointerType(t Type) (*PointerType, bool) {
	t = StripType(t)
	t2, ok := t.(*PointerType)
	return t2, ok
}

// GetFuncType ...
func GetFuncType(t Type) (*FuncType, bool) {
	t = StripType(t)
	t2, ok := t.(*FuncType)
	return t2, ok
}

// GetGenericInstanceType ...
func GetGenericInstanceType(t Type) (*GenericInstanceType, bool) {
	switch t2 := t.(type) {
	case *AliasType:
		return GetGenericInstanceType(t2.Alias)
	case *QualifiedType:
		return GetGenericInstanceType(t2.Type)
	case *GenericInstanceType:
		return t2, true
	}
	return nil, false
}

// GetAliasType ...
func GetAliasType(t Type) (*AliasType, bool) {
	switch t2 := t.(type) {
	case *AliasType:
		return t2, true
	case *QualifiedType:
		return GetAliasType(t2.Type)
	}
	return nil, false
}

// TypeHasPointers returns true if the type contains pointers (except unsafe pointers).
// Unsafe pointers and structs/unions containing pointers to other groups are ignored
func TypeHasPointers(t Type) bool {
	switch t2 := t.(type) {
	case *AliasType:
		return TypeHasPointers(t2.Alias)
	case *QualifiedType:
		return TypeHasPointers(t2.Type)
	case *ArrayType:
		return TypeHasPointers(t2.ElementType)
	case *SliceType:
		return TypeHasPointers(t2.ElementType)
	case *PointerType:
		// Unsafe pointers are treated like integers.
		if t2.Mode != PtrUnsafe {
			return true
		}
	case *StructType:
		for _, f := range t2.Fields {
			if TypeHasPointers(f.Type) {
				return true
			}
		}
	case *UnionType:
		for _, f := range t2.Fields {
			if TypeHasPointers(f.Type) {
				return true
			}
		}
	case *PrimitiveType:
		return t2 == PrimitiveTypeString
	case *GenericInstanceType:
		return TypeHasPointers(t2.InstanceType)
	}
	return false
}

// TypeHasPointers returns true if the type contains owning pointers.
func TypeHasOwningPointers(t Type) bool {
	switch t2 := t.(type) {
	case *AliasType:
		return TypeHasOwningPointers(t2.Alias)
	case *QualifiedType:
		return TypeHasOwningPointers(t2.Type)
	case *ArrayType:
		return TypeHasOwningPointers(t2.ElementType)
	case *SliceType:
		return TypeHasOwningPointers(t2.ElementType)
	case *PointerType:
		if t2.Mode == PtrOwner {
			return true
		}
	case *StructType:
		for _, f := range t2.Fields {
			if TypeHasOwningPointers(f.Type) {
				return true
			}
		}
	case *UnionType:
		for _, f := range t2.Fields {
			if TypeHasOwningPointers(f.Type) {
				return true
			}
		}
	case *PrimitiveType:
		return t2 == PrimitiveTypeString
	case *GenericInstanceType:
		return TypeHasOwningPointers(t2.InstanceType)
	}
	return false
}

// StripType ...
func StripType(t Type) Type {
	switch t2 := t.(type) {
	case *AliasType:
		return StripType(t2.Alias)
	case *QualifiedType:
		return StripType(t2.Type)
	case *GenericInstanceType:
		return StripType(t2.InstanceType)
	}
	return t
}

// Destructor ...
func Destructor(t Type) *Func {
	t = StripType(t)
	if st, ok := t.(*StructType); ok {
		return st.Destructor()
	}
	// TODO for array
	return nil
}

// NeedsDestructor ...
func NeedsDestructor(t Type) bool {
	switch t2 := t.(type) {
	case *AliasType:
		return NeedsDestructor(t2.Alias)
	case *QualifiedType:
		return NeedsDestructor(t2.Type)
	case *SliceType:
		return NeedsDestructor(t2.ElementType)
	case *StructType:
		if t2.Destructor() != nil {
			return true
		}
		if t2.BaseType != nil && NeedsDestructor(t2.BaseType) {
			return true
		}
		for _, f := range t2.Fields {
			if NeedsDestructor(f.Type) {
				return true
			}
		}
	case *PointerType:
		if t2.Mode == PtrOwner {
			return true
		}
	case *ArrayType:
		return NeedsDestructor(t2.ElementType)
	case *GenericInstanceType:
		return NeedsDestructor(t2.InstanceType)
	case *InterfaceType:
		return true
	}
	return false
}

func GetGroupSpecifiers(t Type, specifiers map[string]bool) {
	switch t2 := t.(type) {
	case *QualifiedType:
		GetGroupSpecifiers(t2.Type, specifiers)
	case *ArrayType:
		GetGroupSpecifiers(t2.ElementType, specifiers)
	case *SliceType:
		GetGroupSpecifiers(t2.ElementType, specifiers)
	case *PointerType:
		if t2.GroupSpecifier != nil {
			for _, e := range t2.GroupSpecifier.Elements {
				specifiers[e.Name] = true
			}
		}
	case *GenericInstanceType:
		GetGroupSpecifiers(t2.InstanceType, specifiers)
	}
}
