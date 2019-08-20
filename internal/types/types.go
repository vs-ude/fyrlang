package types

import (
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
	setName(string)
	Location() errlog.LocationRange
	Check(log *errlog.ErrorLog) error
}

// TypeBase ...
type TypeBase struct {
	location     errlog.LocationRange
	name         string
	typeChecked  bool
	funcsChecked bool
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
	Target   Type
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
	Type Type
}

// GroupType ...
type GroupType struct {
	TypeBase
	GroupName string
	Group     *Group
	Type      Type
}

// GenericType ...
type GenericType struct {
	TypeBase
	Type           parser.Node
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
	// The scope containing the type arguments
	Scope *Scope
}

// Name ...
func (t *TypeBase) Name() string {
	return t.name
}

// setName ...
func (t *TypeBase) setName(name string) {
	t.name = name
}

// Location ...
func (t *TypeBase) Location() errlog.LocationRange {
	return t.location
}

// IsTypedef ...
func (t *TypeBase) IsTypedef() bool {
	return t.name != ""
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
func (t *PointerType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// Check ...
func (t *SliceType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// Check ...
func (t *ArrayType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.ElementType.Check(log)
}

// Check ...
func (t *GroupType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
}

// Check ...
func (t *MutableType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	return t.Type.Check(log)
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
	return nil
}

// Check ...
func (t *GenericInstanceType) Check(log *errlog.ErrorLog) error {
	if t.typeChecked {
		return nil
	}
	t.typeChecked = true
	for _, f := range t.BaseType.Funcs {
		tf, err := declareFunction(f.ast, t.Scope, log)
		if err != nil {
			return err
		}
		tf.Component = f.Component
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

/*************************************************
 *
 * Helper functions
 *
 *************************************************/

func isEqualType(left Type, right Type, mode EqualTypeMode) bool {
	if mode != Strict {
		// Groups can only be checked later
		if l, ok := left.(*GroupType); ok {
			left = l.Type
		}
		if r, ok := right.(*GroupType); ok {
			right = r.Type
		}
	}
	if mode == Comparable {
		if l, ok := left.(*MutableType); ok {
			left = l.Type
		}
		if r, ok := right.(*MutableType); ok {
			right = r.Type
		}
	} else if mode == Assignable {
		_, okl := left.(*MutableType)
		r, okr := right.(*MutableType)
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
		return l.Size != r.Size || isEqualType(l.ElementType, r.ElementType, mode)
	case *StructType:
		return false
	case *MutableType:
		r, ok := right.(*MutableType)
		if !ok {
			return false
		}
		return isEqualType(l.Type, r.Type, mode)
	case *PrimitiveType:
		return false
	default:
		panic("TODO")
	}
}

func checkEqualType(left Type, right Type, mode EqualTypeMode, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if isEqualType(left, right, mode) {
		return nil
	}
	return log.AddError(errlog.ErrorIncompatibleTypes, loc)
}

func isSliceType(t Type) bool {
	switch t2 := t.(type) {
	case *SliceType:
		return true
	case *MutableType:
		return isSliceType(t2.Type)
	case *GroupType:
		return isSliceType(t2.Type)
	}
	return false
}

func isPointerType(t Type) bool {
	switch t2 := t.(type) {
	case *PointerType:
		return true
	case *MutableType:
		return isPointerType(t2.Type)
	case *GroupType:
		return isPointerType(t2.Type)
	}
	return false
}

func isIntegerType(t Type) bool {
	return t == integerType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type || t == uintType || t == uint8Type || t == uint16Type || t == uint32Type || t == uint64Type
}

func isSignedIntegerType(t Type) bool {
	return t == integerType || t == intType || t == int8Type || t == int16Type || t == int32Type || t == int64Type
}

func isFloatType(t Type) bool {
	return t == floatType || t == float32Type || t == float64Type
}

func getArrayType(t Type) (*ArrayType, bool) {
	a, ok := t.(*ArrayType)
	return a, ok
}
