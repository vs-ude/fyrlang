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
	// FunctionScope ...
	FunctionScope
	// GenericTypeScope ...
	GenericTypeScope
)

// Scope ...
type Scope struct {
	Kind   ScopeKind
	Parent *Scope
	// For scopes of kind PackageScope this points to the respective package.
	Package  *Package
	Types    map[string]Type
	Elements map[string]ScopeElement
	Groups   map[string]*Group
	// The group to which all local variables of this scope belong
	Group    *Group
	Location errlog.LocationRange
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
	// May be null
	Target Type
	Type   *FuncType
	Ast    *parser.FuncNode
	// The scope in which this function is declared.
	OuterScope *Scope
	// The scope that contains the function parameters etc.
	InnerScope *Scope
	// May be null
	GenericFunc *GenericFunc
	// May be null
	TypeArguments map[string]Type
	Location      errlog.LocationRange
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
	Component  *ComponentType
	Type       *ExprType
	Assignment *VariableAssignment
}

// VariableAssignment ...
// Groups the assignment of one expression to multiple variables.
type VariableAssignment struct {
	Variables []*Variable
	Values    []parser.Node
}

// Name ...
func (f *Func) Name() string {
	return f.name
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

func newScope(parent *Scope, kind ScopeKind, loc errlog.LocationRange) *Scope {
	s := &Scope{Types: make(map[string]Type), Elements: make(map[string]ScopeElement), Parent: parent, Kind: kind}
	if kind == FunctionScope {
		s.Groups = make(map[string]*Group)
	}
	s.Group = NewScopedGroup(s, s.Location)
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
	s.AddType(float32Type, nil)
	s.AddType(float64Type, nil)
	s.AddType(boolType, nil)
	s.AddType(byteType, nil)
	s.AddType(runeType, nil)
	s.AddType(nullType, nil)
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
	for ; s.Parent != nil; s = s.Parent {
		if s.Kind == PackageScope {
			return s
		}
	}
	panic("No package")
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
	return nil
}

// AddElement ...
func (s *Scope) AddElement(element ScopeElement, loc errlog.LocationRange, log *errlog.ErrorLog) error {
	if s.Kind == FileScope {
		return s.Parent.AddElement(element, loc, log)
	}
	name := element.Name()
	if _, ok := s.Elements[name]; ok {
		return log.AddError(errlog.ErrorDuplicateScopeName, loc, name)
	}
	s.Elements[name] = element
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
func (s *Scope) lookupType(name string) Type {
	if t, ok := s.Types[name]; ok {
		return t
	}
	if s.Parent != nil {
		return s.Parent.lookupType(name)
	}
	panic("Unknown type " + name)
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

// LookupOrCreateGroup ...
func (s *Scope) LookupOrCreateGroup(name string, loc errlog.LocationRange) *Group {
	g := s.lookupGroup(name)
	if g == nil {
		g = NewNamedGroup(name, loc)
		// Register the group in the function scope (if inside a function)
		for s != nil {
			if s.Kind == FunctionScope {
				s.Groups[name] = g
				break
			}
			s = s.Parent
		}
	}
	return g
}

func (s *Scope) lookupGroup(name string) *Group {
	if g, ok := s.Groups[name]; ok {
		return g
	}
	if s.Parent != nil {
		return s.Parent.lookupGroup(name)
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
