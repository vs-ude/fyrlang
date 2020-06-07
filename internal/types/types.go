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
	Mode        PointerMode
	ElementType Type
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
	Mutable  bool
	Volatile bool
	Type     Type
}

// GroupedType ...
type GroupedType struct {
	TypeBase
	// The group specifier used by this type.
	GroupSpecifier *GroupSpecifier
	Type           Type
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
	GenericScope *Scope
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
func (t *GroupedType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
}

// ToString ...
func (t *GroupedType) ToString() string {
	if t.GroupSpecifier.Kind == GroupSpecifierIsolate {
		return "->" + t.Type.ToString()
	}
	return "-" + t.GroupSpecifier.Name + " " + t.Type.ToString()
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
	// Check group specifiers (or ignore them)
	if mode == Comparable {
		// Groups do not matter for comparison
		if l, ok := left.(*GroupedType); ok {
			left = l.Type
		}
		if r, ok := right.(*GroupedType); ok {
			right = r.Type
		}
	} else {
		// Groups do not matter for comparison
		l, lok := left.(*GroupedType)
		r, rok := right.(*GroupedType)
		if lok != rok {
			return false
		}
		if lok {
			if l.GroupSpecifier.Kind != r.GroupSpecifier.Kind {
				return false
			}
			if l.GroupSpecifier.Kind == GroupSpecifierNamed && l.GroupSpecifier.Name != r.GroupSpecifier.Name {
				return false
			}
			left = l.Type
			right = r.Type
		}
	}

	// Check mutability
	if mode == Comparable {
		// Mutability and volatile do not matter
		if l, ok := left.(*MutableType); ok {
			left = l.Type
		}
		if r, ok := right.(*MutableType); ok {
			right = r.Type
		}
	} else if mode == Assignable {
		_, okl := left.(*MutableType)
		r, okr := right.(*MutableType)
		if !okl && okr && !r.Volatile {
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
		if mode == Assignable {
			if !ok && l.Mutable {
				return false
			} else if !ok {
				return isEqualType(l.Type, right, mode)
			}
			if l.Mutable && !r.Mutable {
				return false
			}
			if !l.Volatile && r.Volatile {
				return false
			}
		} else {
			if !ok || l.Mutable != r.Mutable || l.Volatile != r.Volatile {
				return false
			}
		}
		return isEqualType(l.Type, r.Type, mode)
	case *PrimitiveType:
		return false
	case *AliasType:
		return false
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
	case *GroupedType:
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
	case *GroupedType:
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
	case *GroupedType:
		return IsArrayType(t2.Type)
	}
	return false
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
	switch t2 := t.(type) {
	case *PointerType:
		return t2.Mode == PtrUnsafe
	case *AliasType:
		return IsUnsafePointerType(t2.Alias)
	case *MutableType:
		return IsUnsafePointerType(t2.Type)
	case *GroupedType:
		return IsUnsafePointerType(t2.Type)
	}
	return false
}

// IsFuncType ...
func IsFuncType(t Type) bool {
	switch t2 := t.(type) {
	case *FuncType:
		return true
	case *AliasType:
		return IsUnsafePointerType(t2.Alias)
	case *MutableType:
		return IsUnsafePointerType(t2.Type)
	case *GroupedType:
		return IsUnsafePointerType(t2.Type)
	}
	return false
}

// GetArrayType ...
func GetArrayType(t Type) (*ArrayType, bool) {
	switch t2 := t.(type) {
	case *ArrayType:
		return t2, true
	case *AliasType:
		return GetArrayType(t2.Alias)
	case *MutableType:
		return GetArrayType(t2.Type)
	case *GroupedType:
		return GetArrayType(t2.Type)
	case *GenericInstanceType:
		return GetArrayType(t2.InstanceType)
	}
	return nil, false
}

// GetSliceType ...
func GetSliceType(t Type) (*SliceType, bool) {
	switch t2 := t.(type) {
	case *SliceType:
		return t2, true
	case *AliasType:
		return GetSliceType(t2.Alias)
	case *MutableType:
		return GetSliceType(t2.Type)
	case *GroupedType:
		return GetSliceType(t2.Type)
	case *GenericInstanceType:
		return GetSliceType(t2.InstanceType)
	}
	return nil, false
}

// GetStructType ...
func GetStructType(t Type) (*StructType, bool) {
	switch t2 := t.(type) {
	case *StructType:
		return t2, true
	case *AliasType:
		return GetStructType(t2.Alias)
	case *MutableType:
		return GetStructType(t2.Type)
	case *GroupedType:
		return GetStructType(t2.Type)
	case *GenericInstanceType:
		return GetStructType(t2.InstanceType)
	}
	return nil, false
}

// GetUnionType ...
func GetUnionType(t Type) (*UnionType, bool) {
	switch t2 := t.(type) {
	case *UnionType:
		return t2, true
	case *AliasType:
		return GetUnionType(t2.Alias)
	case *MutableType:
		return GetUnionType(t2.Type)
	case *GroupedType:
		return GetUnionType(t2.Type)
	case *GenericInstanceType:
		return GetUnionType(t2.InstanceType)
	}
	return nil, false
}

// GetPointerType ...
func GetPointerType(t Type) (*PointerType, bool) {
	switch t2 := t.(type) {
	case *PointerType:
		return t2, true
	case *AliasType:
		return GetPointerType(t2.Alias)
	case *MutableType:
		return GetPointerType(t2.Type)
	case *GroupedType:
		return GetPointerType(t2.Type)
	case *GenericInstanceType:
		return GetPointerType(t2.InstanceType)
	}
	return nil, false
}

// GetFuncType ...
func GetFuncType(t Type) (*FuncType, bool) {
	switch t2 := t.(type) {
	case *AliasType:
		return GetFuncType(t2.Alias)
	case *FuncType:
		return t2, true
	case *MutableType:
		return GetFuncType(t2.Type)
	case *GroupedType:
		return GetFuncType(t2.Type)
	case *GenericInstanceType:
		return GetFuncType(t2.InstanceType)
	}
	return nil, false
}

// GetGenericInstanceType ...
func GetGenericInstanceType(t Type) (*GenericInstanceType, bool) {
	switch t2 := t.(type) {
	case *MutableType:
		return GetGenericInstanceType(t2.Type)
	case *GroupedType:
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
	case *MutableType:
		return GetAliasType(t2.Type)
	case *GroupedType:
		return GetAliasType(t2.Type)
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
	case *GroupedType:
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

// RemoveGroup returns the same type, but without a toplevel GroupType component (if there is any).
func RemoveGroup(t Type) Type {
	switch t2 := t.(type) {
	case *GroupedType:
		return t2.Type
	}
	return t
}

// StripType ...
func StripType(t Type) Type {
	switch t2 := t.(type) {
	case *AliasType:
		return StripType(t2.Alias)
	case *MutableType:
		return StripType(t2.Type)
	case *GroupedType:
		return StripType(t2.Type)
	}
	return t
}
