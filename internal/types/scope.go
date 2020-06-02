package types

import (
	"unicode"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// ScopeKind ...
type ScopeKind int

const (
	// RootScope ...
	RootScope ScopeKind = iota
	// PackageScope ...
	PackageScope
	// FileScope ...
	FileScope
	// ComponentScope ...
	ComponentScope
	// ComponentFileScope ...
	ComponentFileScope
	// FunctionScope ...
	FunctionScope
	// GenericTypeScope ...
	GenericTypeScope
	// IfScope ...
	IfScope
	// ForScope ...
	ForScope
)

// Scope ...
type Scope struct {
	Kind   ScopeKind
	Parent *Scope
	// For scopes of kind PackageScope this points to the respective package.
	Package *Package
	// For scoped of kine ComponentScope this points to the component type owning the scope.
	Component *ComponentType
	Types     map[string]Type
	Elements  map[string]ScopeElement
	// Used with FunctionScope only.
	// The field stores all named group specifiers used by the function's parameters.
	// This is required to ensure that all group-specifiers of the same name are mapped
	// to the same GroupSpecifier instance.
	GroupSpecifiers map[string]*GroupSpecifier
	// Used with FunctionScope only
	Func     *Func
	Location errlog.LocationRange
	// Used for debugging
	ID int
	// dualIsMut is 1 if dual is mut, it is 0 if this scope carries no information about this
	// and -1 if dual is not-mut.
	dualIsMut int
}

// ScopeElement ...
type ScopeElement interface {
	Name() string
}

// Func ...
// Implements ScopeElement.
type Func struct {
	name string
	// May be null
	Component *ComponentType
	Type      *FuncType
	Ast       *parser.FuncNode
	// The scope in which this function is declared.
	OuterScope *Scope
	// The scope that contains the function parameters etc.
	// This is null for extern functions.
	InnerScope *Scope
	// May be null
	GenericFunc *GenericFunc
	// May be null
	TypeArguments map[string]Type
	// Functions in the `extern "C" { ... }` section are labeled as extern
	IsExtern bool
	// Set by the attribute `[export]`
	IsExported bool
	// Set by the attribute `[isr]`
	IsInterruptServiceRoutine bool
	// Set by the attribute `[nomangle]`
	NoNameMangling bool
	// Functions with the dual keyword are parsed twice, once with this flag set to true
	// and once set to false.
	DualIsMut bool
	// The dual func requires a non-mutable target
	DualFunc *Func
	Location errlog.LocationRange
}

// GenericFunc ...
// Implements ScopeElement.
type GenericFunc struct {
	name string
	// May be null
	Component      *ComponentType
	TypeParameters []*GenericTypeParameter
	ast            parser.Node
}

// Namespace ...
// Implements ScopeElement.
type Namespace struct {
	name  string
	Scope *Scope
}

// Variable ...
// Implements ScopeElement.
type Variable struct {
	name string
	// May be null
	Component *ComponentType
	Type      *ExprType
	// Set by the attribute `[export]`
	IsExported bool
}

// ComponentUsage corresponds to a `use` statement
// Implements ScopeElement.
type ComponentUsage struct {
	// Might be empty
	name     string
	Type     *ComponentType
	Location errlog.LocationRange
}

// Name ...
func (c *ComponentUsage) Name() string {
	return c.name
}

// Name ...
func (f *Func) Name() string {
	return f.name
}

// IsGenericMemberFunc returns true if the `f.Type` has a `Target` of type `GenericType`.
// These functions are not instantiated. Use IsGenericInstanceMemberFunc to check
// whether a function belongs to the instantiation of a generic type.
func (f *Func) IsGenericMemberFunc() bool {
	t := f.Type.Target
	if t == nil {
		return false
	}
	if g, ok := t.(*GroupedType); ok {
		t = g.Type
	}
	if m, ok := t.(*MutableType); ok {
		t = m.Type
	}
	if ptr, ok := t.(*PointerType); ok {
		t = ptr.ElementType
	}
	_, ok := t.(*GenericType)
	return ok
}

// IsGenericInstanceMemberFunc returns true if the Func has a Target of type `GenericTypeInstance`.
func (f *Func) IsGenericInstanceMemberFunc() bool {
	t := f.Type.Target
	if t == nil {
		return false
	}
	if g, ok := t.(*GroupedType); ok {
		t = g.Type
	}
	if m, ok := t.(*MutableType); ok {
		t = m.Type
	}
	if ptr, ok := t.(*PointerType); ok {
		t = ptr.ElementType
	}
	_, ok := t.(*GenericInstanceType)
	return ok
}

// redefinesElement checks if the function is a deviating redefinition of an existing element
// TODO: currently only compares if types match _exactly_
func (f *Func) redefinesElement(e ScopeElement) bool {
	// existing is also an external function?
	element, ok := e.(*Func)
	if !ok {
		return true
	}
	if !element.IsExtern {
		return true
	}
	// compare parameter types
	if len(f.Type.In.Params) != len(element.Type.In.Params) {
		return true
	}
	for i := 0; i < len(f.Type.In.Params); i++ {
		if f.Type.In.Params[i].Type.ToString() != element.Type.In.Params[i].Type.ToString() {
			return true
		}
	}
	if len(f.Type.Out.Params) != len(element.Type.Out.Params) {
		return true
	}
	for i := 0; i < len(f.Type.Out.Params); i++ {
		if f.Type.Out.Params[i].Type.ToString() != element.Type.Out.Params[i].Type.ToString() {
			return true
		}
	}
	return false
}

// IsGenericInstanceFunc ...
func (f *Func) IsGenericInstanceFunc() bool {
	return f.GenericFunc != nil
}

// Name ...
func (f *GenericFunc) Name() string {
	return f.name
}

// Name ...
func (f *Namespace) Name() string {
	return f.name
}

// Name ...
func (f *Variable) Name() string {
	return f.name
}

var scopeIds = 1

func newScope(parent *Scope, kind ScopeKind, loc errlog.LocationRange) *Scope {
	s := &Scope{Types: make(map[string]Type), Elements: make(map[string]ScopeElement), Parent: parent, Kind: kind, ID: scopeIds}
	scopeIds++
	if kind == FunctionScope {
		s.GroupSpecifiers = make(map[string]*GroupSpecifier)
	}
	//	s.Group = NewScopedGroup(s, s.Location)
	return s
}

// NewRootScope ...
func NewRootScope() *Scope {
	s := newScope(nil, RootScope, errlog.EncodeLocationRange(0, 0, 0, 0, 0))
	s.AddType(intType, nil)
	s.AddType(int8Type, nil)
	s.AddType(int16Type, nil)
	s.AddType(int32Type, nil)
	s.AddType(int64Type, nil)
	s.AddType(uintType, nil)
	s.AddType(uint8Type, nil)
	s.AddType(uint16Type, nil)
	s.AddType(uint32Type, nil)
	s.AddType(uint64Type, nil)
	s.AddType(uintptrType, nil)
	s.AddType(float32Type, nil)
	s.AddType(float64Type, nil)
	s.AddType(boolType, nil)
	s.AddTypeByName(byteType, "byte", nil)
	s.AddType(runeType, nil)
	s.AddType(nullType, nil)
	s.AddType(stringType, nil)
	return s
}

// Root ...
func (s *Scope) Root() *Scope {
	for ; s.Parent != nil; s = s.Parent {
	}
	return s
}

// HasParent ...
func (s *Scope) HasParent(p *Scope) bool {
	if s.Parent == p {
		return true
	}
	if s.Parent != nil {
		return s.Parent.HasParent(p)
	}
	return false
}

// PackageScope ...
func (s *Scope) PackageScope() *Scope {
	for ; s != nil; s = s.Parent {
		if s.Kind == PackageScope {
			return s
		}
	}
	panic("No package")
}

// InstantiatingPackage ...
func (s *Scope) InstantiatingPackage() *Package {
	for ; s != nil; s = s.Parent {
		if s.Kind == GenericTypeScope {
			return s.Package
		}
		if s.Kind == PackageScope {
			return s.Package
		}
	}
	panic("No package")
}

// ComponentScope ...
// Returns nil, if the scope does not belong to a component.
func (s *Scope) ComponentScope() *Scope {
	for ; s != nil; s = s.Parent {
		if s.Kind == ComponentScope {
			return s
		}
	}
	return nil
}

// ForScope ...
func (s *Scope) ForScope() *Scope {
	for ; s != nil; s = s.Parent {
		if s.Kind == FunctionScope {
			return nil
		}
		if s.Kind == ForScope {
			return s
		}
	}
	return nil
}

// FunctionScope ...
func (s *Scope) FunctionScope() *Scope {
	for ; s != nil; s = s.Parent {
		if s.Kind == FunctionScope {
			return s
		}
	}
	return nil
}

// ComponentTypes ...
func (s *Scope) ComponentTypes() (result []*ComponentType) {
	for _, t := range s.Types {
		if c, ok := t.(*ComponentType); ok {
			result = append(result, c)
		}
	}
	return
}

// DualIsMut ...
func (s *Scope) DualIsMut() int {
	for ; s != nil; s = s.Parent {
		if s.dualIsMut != 0 {
			return s.dualIsMut
		}
	}
	return 0
}

// AddType ...
func (s *Scope) AddType(t Type, log *errlog.ErrorLog) error {
	if s.Kind == FileScope {
		return s.Parent.AddType(t, log)
	}
	if _, ok := s.Types[t.Name()]; ok {
		return log.AddError(errlog.ErrorDuplicateTypeName, t.Location(), t.Name())
	}
	s.Types[t.Name()] = t
	if s.Kind == ComponentFileScope {
		return s.Component.ComponentScope.AddType(t, log)
	}
	return nil
}

// AddTypeByName ...
func (s *Scope) AddTypeByName(t Type, name string, log *errlog.ErrorLog) error {
	if s.Kind == FileScope {
		return s.Parent.AddTypeByName(t, name, log)
	}
	if _, ok := s.Types[name]; ok {
		return log.AddError(errlog.ErrorDuplicateTypeName, t.Location(), name)
	}
	s.Types[name] = t
	if s.Kind == ComponentFileScope {
		return s.Component.ComponentScope.AddTypeByName(t, name, log)
	}
	return nil
}

// AddElement ...
func (s *Scope) AddElement(element ScopeElement, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if s.Kind == FileScope {
		return s.Parent.AddElement(element, loc, log)
	}
	name := element.Name()
	if existing, ok := s.Elements[name]; ok {
		if f, ok2 := element.(*Func); ok2 && f.IsExtern && !f.redefinesElement(existing) {
			// noop
		} else {
			return log.AddError(errlog.ErrorDuplicateScopeName, loc, name)
		}
	} else {
		s.Elements[name] = element
	}
	if s.Kind == ComponentFileScope {
		return s.Component.ComponentScope.AddElement(element, loc, log)
	}
	return nil
}

// AddNamespace ...
func (s *Scope) AddNamespace(ns *Namespace, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	name := ns.Name()
	if _, ok := s.Elements[name]; ok {
		return log.AddError(errlog.ErrorDuplicateScopeName, loc, name)
	}
	s.Elements[name] = ns
	return nil
}

// lookupType ...
func (s *Scope) lookupType(name string) (*Scope, Type) {
	if t, ok := s.Types[name]; ok {
		return s, t
	}
	if s.Parent != nil {
		return s.Parent.lookupType(name)
	}
	return nil, nil
}

// LookupType ...
func (s *Scope) LookupType(name string, loc errlog.LocationRange, log *errlog.ErrorLog) (Type, error) {
	if t, ok := s.Types[name]; ok {
		return t, nil
	}
	if s.Parent != nil {
		return s.Parent.LookupType(name, loc, log)
	}
	return nil, log.AddError(errlog.ErrorUnknownType, loc, name)
}

// HasElement does not log an error if the element does not exist
func (s *Scope) HasElement(name string) ScopeElement {
	if e, ok := s.Elements[name]; ok {
		return e
	}
	if s.Parent != nil {
		return s.Parent.HasElement(name)
	}
	return nil
}

// GetElement does not log an error if the element does not exist.
// Instead, it panics.
func (s *Scope) GetElement(name string) ScopeElement {
	if e, ok := s.Elements[name]; ok {
		return e
	}
	if s.Parent != nil {
		return s.Parent.GetElement(name)
	}
	panic("element does not exist " + name)
}

// GetVariable does not log an error if the element does not exist
func (s *Scope) GetVariable(name string) *Variable {
	if e, ok := s.Elements[name]; ok {
		if v, ok := e.(*Variable); ok {
			return v
		}
		panic("var does not exist")
	}
	if s.Parent != nil {
		return s.Parent.GetVariable(name)
	}
	panic("var does not exist " + name)
}

func (s *Scope) lookupElement(name string, loc errlog.LocationRange, log *errlog.ErrorLog) (*Scope, ScopeElement) {
	if e, ok := s.Elements[name]; ok {
		return s, e
	}
	if s.Parent != nil {
		return s.Parent.lookupElement(name, loc, log)
	}
	return nil, nil
}

// LookupElement ...
func (s *Scope) LookupElement(name string, loc errlog.LocationRange, log *errlog.ErrorLog) (ScopeElement, error) {
	if e, ok := s.Elements[name]; ok {
		return e, nil
	}
	if s.Parent != nil {
		return s.Parent.LookupElement(name, loc, log)
	}
	return nil, log.AddError(errlog.ErrorUnknownIdentifier, loc, name)
}

// LookupNamespace ...
func (s *Scope) LookupNamespace(name string, loc errlog.LocationRange, log *errlog.ErrorLog) (*Namespace, error) {
	if s2, ok := s.Elements[name]; ok {
		ns, ok := s2.(*Namespace)
		if !ok {
			return nil, log.AddError(errlog.ErrorNotANamespace, loc, name)
		}
		return ns, nil
	}
	if s.Parent != nil {
		return s.Parent.LookupNamespace(name, loc, log)
	}
	return nil, log.AddError(errlog.ErrorUnknownNamespace, loc, name)
}

// LookupNamedType ...
func (s *Scope) LookupNamedType(n *parser.NamedTypeNode, log *errlog.ErrorLog) (Type, error) {
	s2 := s
	if n.Namespace != nil {
		for _, r := range n.NameToken.StringValue {
			if !unicode.IsUpper(r) {
				return nil, log.AddError(errlog.ErrorNameNotExported, n.NameToken.Location, n.NameToken.StringValue)
			}
			break
		}
		var err error
		s2, err = s.lookupNamedScope(n.Namespace, log)
		if err != nil {
			return nil, err
		}
	}
	return s2.LookupType(n.NameToken.StringValue, n.NameToken.Location, log)
}

// lookupNamedScope ...
func (s *Scope) lookupNamedScope(n *parser.NamedTypeNode, log *errlog.ErrorLog) (*Scope, error) {
	s2 := s
	if n.Namespace != nil {
		var err error
		s2, err = s.lookupNamedScope(n.Namespace, log)
		if err != nil {
			return nil, err
		}
	}
	ns, err := s2.LookupNamespace(n.NameToken.StringValue, n.NameToken.Location, log)
	if err != nil {
		return nil, err
	}
	return ns.Scope, nil
}

// LookupOrCreateGroupSpecifier ...
func (s *Scope) LookupOrCreateGroupSpecifier(name string, loc errlog.LocationRange) *GroupSpecifier {
	g := s.lookupGroupSpecifier(name)
	if g == nil {
		g = NewNamedGroupSpecifier(name, loc)
		// Register the group in the function scope (if inside a function)
		for s != nil {
			if s.Kind == FunctionScope {
				s.GroupSpecifiers[name] = g
				break
			}
			s = s.Parent
		}
	}
	return g
}

func (s *Scope) lookupGroupSpecifier(name string) *GroupSpecifier {
	if g, ok := s.GroupSpecifiers[name]; ok {
		return g
	}
	if s.Parent != nil {
		return s.Parent.lookupGroupSpecifier(name)
	}
	return nil
}

func (s *Scope) lookupVariable(name string) *Variable {
	if v, ok := s.Elements[name]; ok {
		if v2, ok := v.(*Variable); ok {
			return v2
		}
		return nil
	}
	if s.Parent != nil {
		return s.Parent.lookupVariable(name)
	}
	return nil
}

// Returns nil, if the scope does not belong to a component.
func (s *Scope) inComponent() *ComponentType {
	for ; s != nil; s = s.Parent {
		if s.Kind == ComponentScope {
			return s.Component
		}
		if s.Kind == FunctionScope {
			return nil
		}
	}
	return nil
}
