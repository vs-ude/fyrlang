package parser

/**************************************
 *
 * Abstract Syntax Tree
 *
 * This file implements the AST
 * which is the result of parsing.
 *
 **************************************/

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
)

// Node is implemented by all nodes of the AST.
type Node interface {
	SetTypeAnnotation(t interface{})
	TypeAnnotation() interface{}
	SetScope(s interface{})
	Scope() interface{}
	Location() errlog.LocationRange
	Clone() Node
}

func clone(n Node) Node {
	if n == nil {
		return nil
	}
	return n.Clone()
}

// NodeBase implements basic functionality use by all AST nodes.
type NodeBase struct {
	location       errlog.LocationRange
	typeAnnotation interface{}
	scope          interface{}
}

// FileNode ...
type FileNode struct {
	NodeBase
	File int
	// FuncNode, ExpressionStatementNode (for var and let), LineNode, ComponentNode, ImportNode, TypedefNode
	Children []Node
}

// Location ...
func (n *FileNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = aloc(n.Children)
	}
	return n.location
}

// Clone ...
func (n *FileNode) Clone() Node {
	return n.Clone()
}

func (n *FileNode) clone() *FileNode {
	if n == nil {
		return n
	}
	c := &FileNode{NodeBase: NodeBase{location: n.location}, File: n.File}
	for _, ch := range n.Children {
		c.Children = append(c.Children, ch.Clone())
	}
	return c
}

// LineNode ...
type LineNode struct {
	NodeBase
	Token *lexer.Token
}

// Location ...
func (n *LineNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.Token)
	}
	return n.location
}

// Clone ...
func (n *LineNode) Clone() Node {
	return n.clone()
}

func (n *LineNode) clone() *LineNode {
	if n == nil {
		return n
	}
	c := &LineNode{NodeBase: NodeBase{location: n.location}, Token: n.Token}
	return c
}

// ComponentNode ...
type ComponentNode struct {
	NodeBase
	ComponentToken *lexer.Token
	NameToken      *lexer.Token
	NewlineToken   *lexer.Token
}

// Location ...
func (n *ComponentNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ComponentToken).Join(tloc(n.NameToken))
	}
	return n.location
}

// Clone ...
func (n *ComponentNode) Clone() Node {
	return n.clone()
}

func (n *ComponentNode) clone() *ComponentNode {
	if n == nil {
		return n
	}
	c := &ComponentNode{NodeBase: NodeBase{location: n.location}, ComponentToken: n.ComponentToken, NameToken: n.NameToken, NewlineToken: n.NewlineToken}
	return c
}

// ExternNode ...
type ExternNode struct {
	NodeBase
	ExternToken   *lexer.Token
	StringToken   *lexer.Token
	OpenToken     *lexer.Token
	NewlineToken1 *lexer.Token
	// ExternFuncNode
	Elements      []Node
	CloseToken    *lexer.Token
	NewlineToken2 *lexer.Token
}

// Location ...
func (n *ExternNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ExternToken).Join(tloc(n.OpenToken)).Join(aloc(n.Elements)).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ExternNode) Clone() Node {
	return n.clone()
}

func (n *ExternNode) clone() *ExternNode {
	if n == nil {
		return n
	}
	c := &ExternNode{NodeBase: NodeBase{location: n.location}, ExternToken: n.ExternToken, StringToken: n.StringToken, OpenToken: n.OpenToken,
		NewlineToken1: n.NewlineToken1, CloseToken: n.CloseToken, NewlineToken2: n.NewlineToken2}
	for _, ch := range n.Elements {
		c.Elements = append(c.Elements, ch.Clone())
	}
	return c
}

// ExternFuncNode ...
type ExternFuncNode struct {
	NodeBase
	ExportToken  *lexer.Token
	FuncToken    *lexer.Token
	MutToken     *lexer.Token
	PointerToken *lexer.Token
	NameToken    *lexer.Token
	Params       *ParamListNode
	ReturnParams *ParamListNode
	NewlineToken *lexer.Token
}

// Location ...
func (n *ExternFuncNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ExportToken).Join(tloc(n.NameToken)).Join(nloc(n.Params)).Join(nloc(n.ReturnParams))
	}
	return n.location
}

// Clone ...
func (n *ExternFuncNode) Clone() Node {
	return n.clone()
}

func (n *ExternFuncNode) clone() *ExternFuncNode {
	if n == nil {
		return n
	}
	c := &ExternFuncNode{NodeBase: NodeBase{location: n.location}, ExportToken: n.ExportToken, FuncToken: n.FuncToken, MutToken: n.MutToken,
		PointerToken: n.PointerToken, NameToken: n.NameToken, Params: n.Params.clone(),
		ReturnParams: n.ReturnParams.clone(), NewlineToken: n.NewlineToken}
	return c
}

// ImportBlockNode ...
type ImportBlockNode struct {
	NodeBase
	ImportToken   *lexer.Token
	OpenToken     *lexer.Token
	NewlineToken1 *lexer.Token
	// ImportNode or LineNode
	Imports       []Node
	CloseToken    *lexer.Token
	NewlineToken2 *lexer.Token
}

// Location ...
func (n *ImportBlockNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ImportToken).Join(tloc(n.OpenToken)).Join(aloc(n.Imports)).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ImportBlockNode) Clone() Node {
	return n.clone()
}

func (n *ImportBlockNode) clone() *ImportBlockNode {
	if n == nil {
		return n
	}
	c := &ImportBlockNode{NodeBase: NodeBase{location: n.location}, ImportToken: n.ImportToken, OpenToken: n.OpenToken, NewlineToken1: n.NewlineToken1,
		CloseToken: n.CloseToken, NewlineToken2: n.NewlineToken2}
	for _, ch := range n.Imports {
		c.Imports = append(c.Imports, ch.Clone())
	}
	return c
}

// ImportNode ...
type ImportNode struct {
	NodeBase
	NameToken    *lexer.Token
	StringToken  *lexer.Token
	NewlineToken *lexer.Token
}

// Location ...
func (n *ImportNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.NameToken).Join(tloc(n.StringToken))
	}
	return n.location
}

// Clone ...
func (n *ImportNode) Clone() Node {
	return n.clone()
}

func (n *ImportNode) clone() *ImportNode {
	if n == nil {
		return n
	}
	c := &ImportNode{NodeBase: NodeBase{location: n.location}, NameToken: n.NameToken, StringToken: n.StringToken, NewlineToken: n.NewlineToken}
	return c
}

// TypedefNode ...
type TypedefNode struct {
	NodeBase
	TypeToken     *lexer.Token
	NameToken     *lexer.Token
	Type          Node
	GenericParams *GenericParamListNode
	NewlineToken  *lexer.Token
}

// Location ...
func (n *TypedefNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.TypeToken).Join(tloc(n.NameToken)).Join(nloc(n.Type)).Join(nloc(n.GenericParams))
	}
	return n.location
}

// Clone ...
func (n *TypedefNode) Clone() Node {
	return n.clone()
}

func (n *TypedefNode) clone() *TypedefNode {
	if n == nil {
		return n
	}
	c := &TypedefNode{NodeBase: NodeBase{location: n.location}, TypeToken: n.TypeToken, NameToken: n.NameToken, Type: clone(n.Type),
		GenericParams: n.GenericParams.clone(), NewlineToken: n.NewlineToken}
	return c
}

// GenericParamListNode ...
type GenericParamListNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Params     []*GenericParamNode
	CloseToken *lexer.Token
}

// Location ...
func (n *GenericParamListNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *GenericParamListNode) Clone() Node {
	return n.clone()
}

func (n *GenericParamListNode) clone() *GenericParamListNode {
	if n == nil {
		return n
	}
	c := &GenericParamListNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken}
	for _, ch := range n.Params {
		c.Params = append(c.Params, ch.Clone().(*GenericParamNode))
	}
	return c
}

// GenericParamNode ...
type GenericParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	NameToken    *lexer.Token
}

// Location ...
func (n *GenericParamNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(tloc(n.NameToken))
	}
	return n.location
}

// Clone ...
func (n *GenericParamNode) Clone() Node {
	return n.clone()
}

func (n *GenericParamNode) clone() *GenericParamNode {
	if n == nil {
		return n
	}
	c := &GenericParamNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken, NameToken: n.NameToken}
	return c
}

// FuncNode ...
type FuncNode struct {
	NodeBase
	ComponentMutToken *lexer.Token
	FuncToken         *lexer.Token
	Type              Node
	DotToken          *lexer.Token
	NameToken         *lexer.Token
	GenericParams     *GenericParamListNode
	Params            *ParamListNode
	ReturnParams      *ParamListNode
	Body              *BodyNode
	NewlineToken      *lexer.Token
}

// Location ...
func (n *FuncNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ComponentMutToken).Join(tloc(n.FuncToken)).Join(nloc(n.Body))
	}
	return n.location
}

// Clone ...
func (n *FuncNode) Clone() Node {
	return n.clone()
}

func (n *FuncNode) clone() *FuncNode {
	if n == nil {
		return n
	}
	c := &FuncNode{NodeBase: NodeBase{location: n.location}, ComponentMutToken: n.ComponentMutToken, FuncToken: n.FuncToken,
		Type: clone(n.Type), DotToken: n.DotToken, NameToken: n.NameToken, GenericParams: n.GenericParams.clone(),
		Params: n.Params.clone(), ReturnParams: n.ReturnParams.clone(),
		Body: n.Body.clone(), NewlineToken: n.NewlineToken}
	return c
}

// ParamListNode ...
type ParamListNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Params     []*ParamNode
	CloseToken *lexer.Token
}

// Location ...
func (n *ParamListNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ParamListNode) Clone() Node {
	return n.clone()
}

func (n *ParamListNode) clone() *ParamListNode {
	if n == nil {
		return n
	}
	c := &ParamListNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken}
	for _, ch := range n.Params {
		c.Params = append(c.Params, ch.Clone().(*ParamNode))
	}
	return c
}

// ParamNode ...
type ParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	NameToken    *lexer.Token
	Type         Node
}

// Location ...
func (n *ParamNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(tloc(n.NameToken)).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *ParamNode) Clone() Node {
	return n.clone()
}

func (n *ParamNode) clone() *ParamNode {
	if n == nil {
		return n
	}
	c := &ParamNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken, NameToken: n.NameToken,
		Type: clone(n.Type)}
	return c
}

// GenericInstanceFuncNode ...
type GenericInstanceFuncNode struct {
	NodeBase
	Expression    Node
	BacktickToken *lexer.Token
	TypeArguments *TypeListNode
}

// Location ...
func (n *GenericInstanceFuncNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(nloc(n.TypeArguments))
	}
	return n.location
}

// Clone ...
func (n *GenericInstanceFuncNode) Clone() Node {
	return n.clone()
}

func (n *GenericInstanceFuncNode) clone() *GenericInstanceFuncNode {
	if n == nil {
		return n
	}
	c := &GenericInstanceFuncNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), BacktickToken: n.BacktickToken,
		TypeArguments: n.TypeArguments.clone()}
	return c
}

// BodyNode ...
type BodyNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Children   []Node
	CloseToken *lexer.Token
}

// Location ...
func (n *BodyNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *BodyNode) Clone() Node {
	return n.clone()
}

func (n *BodyNode) clone() *BodyNode {
	if n == nil {
		return n
	}
	c := &BodyNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken}
	for _, ch := range n.Children {
		c.Children = append(c.Children, clone(ch))
	}
	return c
}

// NamedTypeNode ...
type NamedTypeNode struct {
	NodeBase
	Namespace         *NamedTypeNode
	NamespaceDotToken *lexer.Token
	NameToken         *lexer.Token
}

// Location ...
func (n *NamedTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Namespace).Join(tloc(n.NamespaceDotToken)).Join(tloc(n.NameToken))
	}
	return n.location
}

// Clone ...
func (n *NamedTypeNode) Clone() Node {
	return n.clone()
}

func (n *NamedTypeNode) clone() *NamedTypeNode {
	if n == nil {
		return n
	}
	c := &NamedTypeNode{NodeBase: NodeBase{location: n.location}, Namespace: n.Namespace.clone(),
		NamespaceDotToken: n.NamespaceDotToken, NameToken: n.NameToken}
	return c
}

// PointerTypeNode ...
type PointerTypeNode struct {
	NodeBase
	PointerToken *lexer.Token
	ElementType  Node
}

// Location ...
func (n *PointerTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.PointerToken).Join(nloc(n.ElementType))
	}
	return n.location
}

// Clone ...
func (n *PointerTypeNode) Clone() Node {
	return n.clone()
}

func (n *PointerTypeNode) clone() *PointerTypeNode {
	if n == nil {
		return n
	}
	c := &PointerTypeNode{NodeBase: NodeBase{location: n.location}, PointerToken: n.PointerToken, ElementType: clone(n.ElementType)}
	return c
}

// MutableTypeNode ...
type MutableTypeNode struct {
	NodeBase
	// Can be TokenMut or TokenDual
	MutToken *lexer.Token
	Type     Node
}

// Location ...
func (n *MutableTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.MutToken).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *MutableTypeNode) Clone() Node {
	return n.clone()
}

func (n *MutableTypeNode) clone() *MutableTypeNode {
	if n == nil {
		return n
	}
	c := &MutableTypeNode{NodeBase: NodeBase{location: n.location}, MutToken: n.MutToken, Type: clone(n.Type)}
	return c
}

// SliceTypeNode ...
type SliceTypeNode struct {
	NodeBase
	OpenToken   *lexer.Token
	CloseToken  *lexer.Token
	ElementType Node
}

// Location ...
func (n *SliceTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(nloc(n.ElementType))
	}
	return n.location
}

// Clone ...
func (n *SliceTypeNode) Clone() Node {
	return n.clone()
}

func (n *SliceTypeNode) clone() *SliceTypeNode {
	if n == nil {
		return n
	}
	c := &SliceTypeNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken, ElementType: clone(n.ElementType)}
	return c
}

// ArrayTypeNode ...
type ArrayTypeNode struct {
	NodeBase
	OpenToken   *lexer.Token
	Size        Node
	CloseToken  *lexer.Token
	ElementType Node
}

// Location ...
func (n *ArrayTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(nloc(n.ElementType))
	}
	return n.location
}

// Clone ...
func (n *ArrayTypeNode) Clone() Node {
	return n.clone()
}

func (n *ArrayTypeNode) clone() *ArrayTypeNode {
	if n == nil {
		return n
	}
	c := &ArrayTypeNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken, Size: clone(n.Size), ElementType: clone(n.ElementType)}
	return c
}

// ClosureTypeNode ...
type ClosureTypeNode struct {
	NodeBase
	FuncToken    *lexer.Token
	Params       *ParamListNode
	ReturnParams *ParamListNode
}

// Location ...
func (n *ClosureTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.FuncToken).Join(nloc(n.Params)).Join(nloc(n.ReturnParams))
	}
	return n.location
}

// Clone ...
func (n *ClosureTypeNode) Clone() Node {
	return n.clone()
}

func (n *ClosureTypeNode) clone() *ClosureTypeNode {
	if n == nil {
		return n
	}
	c := &ClosureTypeNode{NodeBase: NodeBase{location: n.location}, FuncToken: n.FuncToken, Params: n.Params.clone(), ReturnParams: n.ReturnParams.clone()}
	return c
}

// StructTypeNode ...
type StructTypeNode struct {
	NodeBase
	StructToken  *lexer.Token
	OpenToken    *lexer.Token
	NewlineToken *lexer.Token
	// StructFieldNode or LineNode
	Fields     []Node
	CloseToken *lexer.Token
}

// Location ...
func (n *StructTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.StructToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *StructTypeNode) Clone() Node {
	return n.clone()
}

func (n *StructTypeNode) clone() *StructTypeNode {
	if n == nil {
		return n
	}
	c := &StructTypeNode{NodeBase: NodeBase{location: n.location}, StructToken: n.StructToken, OpenToken: n.OpenToken,
		NewlineToken: n.NewlineToken, CloseToken: n.CloseToken}
	for _, ch := range n.Fields {
		c.Fields = append(c.Fields, clone(ch))
	}
	return c
}

// StructFieldNode ...
type StructFieldNode struct {
	NodeBase
	NameToken    *lexer.Token
	Type         Node
	NewlineToken *lexer.Token
}

// Location ...
func (n *StructFieldNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.NameToken).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *StructFieldNode) Clone() Node {
	return n.clone()
}

func (n *StructFieldNode) clone() *StructFieldNode {
	if n == nil {
		return n
	}
	c := &StructFieldNode{NodeBase: NodeBase{location: n.location}, NameToken: n.NameToken, Type: clone(n.Type), NewlineToken: n.NewlineToken}
	return c
}

// InterfaceTypeNode ...
type InterfaceTypeNode struct {
	NodeBase
	InterfaceToken *lexer.Token
	OpenToken      *lexer.Token
	NewlineToken   *lexer.Token
	// InterfaceFuncNode or LineNode or InterfaceFieldNode
	Fields     []Node
	CloseToken *lexer.Token
}

// Location ...
func (n *InterfaceTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.InterfaceToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *InterfaceTypeNode) Clone() Node {
	return n.clone()
}

func (n *InterfaceTypeNode) clone() *InterfaceTypeNode {
	if n == nil {
		return n
	}
	c := &InterfaceTypeNode{NodeBase: NodeBase{location: n.location}, InterfaceToken: n.InterfaceToken, OpenToken: n.OpenToken,
		NewlineToken: n.NewlineToken, CloseToken: n.CloseToken}
	for _, ch := range n.Fields {
		c.Fields = append(c.Fields, clone(ch))
	}
	return c
}

// InterfaceFuncNode ...
type InterfaceFuncNode struct {
	NodeBase
	// TokenMut or null
	ComponentMutToken *lexer.Token
	FuncToken         *lexer.Token
	// TokenMut, TokenDual or null
	MutToken     *lexer.Token
	PointerToken *lexer.Token
	NameToken    *lexer.Token
	Params       *ParamListNode
	ReturnParams *ParamListNode
	NewlineToken *lexer.Token
}

// Location ...
func (n *InterfaceFuncNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ComponentMutToken).Join(tloc(n.NameToken)).Join(nloc(n.Params)).Join(nloc(n.ReturnParams))
	}
	return n.location
}

// Clone ...
func (n *InterfaceFuncNode) Clone() Node {
	return n.clone()
}

func (n *InterfaceFuncNode) clone() *InterfaceFuncNode {
	if n == nil {
		return n
	}
	c := &InterfaceFuncNode{NodeBase: NodeBase{location: n.location}, ComponentMutToken: n.ComponentMutToken, FuncToken: n.FuncToken,
		MutToken: n.MutToken, PointerToken: n.PointerToken, NameToken: n.NameToken, Params: n.Params.clone(),
		ReturnParams: n.ReturnParams.clone(), NewlineToken: n.NewlineToken}
	return c
}

// InterfaceFieldNode ...
type InterfaceFieldNode struct {
	NodeBase
	Type         Node
	NewlineToken *lexer.Token
}

// Location ...
func (n *InterfaceFieldNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Type)
	}
	return n.location
}

// Clone ...
func (n *InterfaceFieldNode) Clone() Node {
	return n.clone()
}

func (n *InterfaceFieldNode) clone() *InterfaceFieldNode {
	if n == nil {
		return n
	}
	c := &InterfaceFieldNode{NodeBase: NodeBase{location: n.location}, Type: clone(n.Type), NewlineToken: n.NewlineToken}
	return c
}

// UnionTypeNode ...
type UnionTypeNode struct {
	NodeBase
	UnionToken   *lexer.Token
	OpenToken    *lexer.Token
	NewlineToken *lexer.Token
	// StructFieldNode or LineNode
	Fields     []Node
	CloseToken *lexer.Token
}

// Location ...
func (n *UnionTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.UnionToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *UnionTypeNode) Clone() Node {
	return n.clone()
}

func (n *UnionTypeNode) clone() *UnionTypeNode {
	if n == nil {
		return n
	}
	c := &UnionTypeNode{NodeBase: NodeBase{location: n.location}, UnionToken: n.UnionToken, OpenToken: n.OpenToken,
		NewlineToken: n.NewlineToken, CloseToken: n.CloseToken}
	for _, ch := range n.Fields {
		c.Fields = append(c.Fields, clone(ch))
	}
	return c
}

// GroupedTypeNode ...
type GroupedTypeNode struct {
	NodeBase
	// Either `-` or `->`
	GroupToken *lexer.Token
	// Is null in the case of `->`
	GroupNameToken *lexer.Token
	Type           Node
}

// Location ...
func (n *GroupedTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.GroupToken).Join(tloc(n.GroupNameToken)).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *GroupedTypeNode) Clone() Node {
	return n.clone()
}

func (n *GroupedTypeNode) clone() *GroupedTypeNode {
	if n == nil {
		return n
	}
	c := &GroupedTypeNode{NodeBase: NodeBase{location: n.location}, GroupToken: n.GroupToken, GroupNameToken: n.GroupNameToken, Type: clone(n.Type)}
	return c
}

// GenericInstanceTypeNode ...
type GenericInstanceTypeNode struct {
	NodeBase
	Type          *NamedTypeNode
	TypeArguments *TypeListNode
}

// Location ...
func (n *GenericInstanceTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Type).Join(nloc(n.TypeArguments))
	}
	return n.location
}

// Clone ...
func (n *GenericInstanceTypeNode) Clone() Node {
	return n.clone()
}

func (n *GenericInstanceTypeNode) clone() *GenericInstanceTypeNode {
	if n == nil {
		return n
	}
	c := &GenericInstanceTypeNode{NodeBase: NodeBase{location: n.location}, Type: n.Type.clone(), TypeArguments: n.TypeArguments.clone()}
	return c
}

// TypeListNode ...
type TypeListNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Types      []*TypeListElementNode
	CloseToken *lexer.Token
}

// Location ...
func (n *TypeListNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *TypeListNode) Clone() Node {
	return n.clone()
}

func (n *TypeListNode) clone() *TypeListNode {
	if n == nil {
		return n
	}
	c := &TypeListNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, CloseToken: n.CloseToken}
	for _, ch := range n.Types {
		c.Types = append(c.Types, ch.clone())
	}
	return c
}

// TypeListElementNode ...
type TypeListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	Type         Node
}

// Location ...
func (n *TypeListElementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *TypeListElementNode) Clone() Node {
	return n.clone()
}

func (n *TypeListElementNode) clone() *TypeListElementNode {
	if n == nil {
		return n
	}
	c := &TypeListElementNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken, Type: clone(n.Type)}
	return c
}

// ExpressionListNode ...
type ExpressionListNode struct {
	NodeBase
	Elements []*ExpressionListElementNode
}

// Location ...
func (n *ExpressionListNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		for _, e := range n.Elements {
			n.location = n.location.Join(nloc(e))
		}
	}
	return n.location
}

// Clone ...
func (n *ExpressionListNode) Clone() Node {
	return n.clone()
}

func (n *ExpressionListNode) clone() *ExpressionListNode {
	if n == nil {
		return n
	}
	c := &ExpressionListNode{NodeBase: NodeBase{location: n.location}}
	for _, ch := range n.Elements {
		c.Elements = append(c.Elements, ch.clone())
	}
	return c
}

// ExpressionListElementNode ...
type ExpressionListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	Expression   Node
}

// Location ...
func (n *ExpressionListElementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(nloc(n.Expression))
	}
	return n.location
}

// Clone ...
func (n *ExpressionListElementNode) Clone() Node {
	return n.clone()
}

func (n *ExpressionListElementNode) clone() *ExpressionListElementNode {
	if n == nil {
		return n
	}
	c := &ExpressionListElementNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken, Expression: clone(n.Expression)}
	return c
}

// ClosureExpressionNode ...
type ClosureExpressionNode struct {
	NodeBase
	AtToken      *lexer.Token
	OpenToken    *lexer.Token
	Expression   Node
	NewlineToken *lexer.Token
	Children     []Node
	CloseToken   *lexer.Token
}

// Location ...
func (n *ClosureExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.AtToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ClosureExpressionNode) Clone() Node {
	return n.clone()
}

func (n *ClosureExpressionNode) clone() *ClosureExpressionNode {
	if n == nil {
		return n
	}
	c := &ClosureExpressionNode{NodeBase: NodeBase{location: n.location}, AtToken: n.AtToken, OpenToken: n.OpenToken,
		Expression: clone(n.Expression), NewlineToken: n.NewlineToken, CloseToken: n.CloseToken}
	for _, ch := range n.Children {
		c.Children = append(c.Children, clone(ch))
	}
	return c
}

// BinaryExpressionNode ...
type BinaryExpressionNode struct {
	NodeBase
	Left    Node
	OpToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	Right        Node
}

// Location ...
func (n *BinaryExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Left).Join(nloc(n.Right))
	}
	return n.location
}

// Clone ...
func (n *BinaryExpressionNode) Clone() Node {
	return n.clone()
}

func (n *BinaryExpressionNode) clone() *BinaryExpressionNode {
	if n == nil {
		return n
	}
	c := &BinaryExpressionNode{NodeBase: NodeBase{location: n.location}, Left: clone(n.Left), OpToken: n.OpToken, NewlineToken: n.NewlineToken, Right: clone(n.Right)}
	return c
}

// UnaryExpressionNode ...
type UnaryExpressionNode struct {
	NodeBase
	OpToken    *lexer.Token
	Expression Node
}

// Location ...
func (n *UnaryExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpToken).Join(nloc(n.Expression))
	}
	return n.location
}

// Clone ...
func (n *UnaryExpressionNode) Clone() Node {
	return n.clone()
}

func (n *UnaryExpressionNode) clone() *UnaryExpressionNode {
	if n == nil {
		return n
	}
	c := &UnaryExpressionNode{NodeBase: NodeBase{location: n.location}, OpToken: n.OpToken, Expression: clone(n.Expression)}
	return c
}

// IsTypeExpressionNode ...
type IsTypeExpressionNode struct {
	NodeBase
	Expression Node
	IsToken    *lexer.Token
	Type       Node
}

// Location ...
func (n *IsTypeExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(nloc(n.Type))
	}
	return n.location
}

// Clone ...
func (n *IsTypeExpressionNode) Clone() Node {
	return n.clone()
}

func (n *IsTypeExpressionNode) clone() *IsTypeExpressionNode {
	if n == nil {
		return n
	}
	c := &IsTypeExpressionNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), IsToken: n.IsToken, Type: clone(n.Type)}
	return c
}

// CastExpressionNode ...
type CastExpressionNode struct {
	NodeBase
	BacktickToken *lexer.Token
	Type          Node
	OpenToken     *lexer.Token
	Expression    Node
	CloseToken    *lexer.Token
}

// Location ...
func (n *CastExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.BacktickToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *CastExpressionNode) Clone() Node {
	return n.clone()
}

func (n *CastExpressionNode) clone() *CastExpressionNode {
	if n == nil {
		return n
	}
	c := &CastExpressionNode{NodeBase: NodeBase{location: n.location}, BacktickToken: n.BacktickToken, Type: clone(n.Type),
		OpenToken: n.OpenToken, Expression: clone(n.Expression), CloseToken: n.CloseToken}
	return c
}

// MemberAccessExpressionNode ...
type MemberAccessExpressionNode struct {
	NodeBase
	Expression      Node
	DotToken        *lexer.Token
	IdentifierToken *lexer.Token
}

// Location ...
func (n *MemberAccessExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(tloc(n.IdentifierToken))
	}
	return n.location
}

// Clone ...
func (n *MemberAccessExpressionNode) Clone() Node {
	return n.clone()
}

func (n *MemberAccessExpressionNode) clone() *MemberAccessExpressionNode {
	if n == nil {
		return n
	}
	c := &MemberAccessExpressionNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), DotToken: n.DotToken, IdentifierToken: n.IdentifierToken}
	return c
}

// MemberCallExpressionNode ...
type MemberCallExpressionNode struct {
	NodeBase
	Expression Node
	OpenToken  *lexer.Token
	Arguments  *ExpressionListNode
	CloseToken *lexer.Token
}

// Location ...
func (n *MemberCallExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *MemberCallExpressionNode) Clone() Node {
	return n.clone()
}

func (n *MemberCallExpressionNode) clone() *MemberCallExpressionNode {
	if n == nil {
		return n
	}
	c := &MemberCallExpressionNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), OpenToken: n.OpenToken,
		Arguments: n.Arguments.clone(), CloseToken: n.CloseToken}
	return c
}

// ArrayAccessExpressionNode ...
type ArrayAccessExpressionNode struct {
	NodeBase
	Expression Node
	OpenToken  *lexer.Token
	// Can be empy on slices like [:a]
	Index Node
	// Used for slicing, i.e. [a:b]
	ColonToken *lexer.Token
	// Used for slicing, i.e. [a:b]
	Index2     Node
	CloseToken *lexer.Token
}

// Location ...
func (n *ArrayAccessExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ArrayAccessExpressionNode) Clone() Node {
	return n.clone()
}

func (n *ArrayAccessExpressionNode) clone() *ArrayAccessExpressionNode {
	if n == nil {
		return n
	}
	c := &ArrayAccessExpressionNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), OpenToken: n.OpenToken,
		Index: clone(n.Index), ColonToken: n.ColonToken, Index2: clone(n.Index2), CloseToken: n.CloseToken}
	return c
}

// ConstantExpressionNode ...
type ConstantExpressionNode struct {
	NodeBase
	ValueToken *lexer.Token
}

// Location ...
func (n *ConstantExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ValueToken)
	}
	return n.location
}

// Clone ...
func (n *ConstantExpressionNode) Clone() Node {
	return n.clone()
}

func (n *ConstantExpressionNode) clone() *ConstantExpressionNode {
	if n == nil {
		return n
	}
	c := &ConstantExpressionNode{NodeBase: NodeBase{location: n.location}, ValueToken: n.ValueToken}
	return c
}

// IdentifierExpressionNode ...
type IdentifierExpressionNode struct {
	NodeBase
	IdentifierToken *lexer.Token
}

// Location ...
func (n *IdentifierExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.IdentifierToken)
	}
	return n.location
}

// Clone ...
func (n *IdentifierExpressionNode) Clone() Node {
	return n.clone()
}

func (n *IdentifierExpressionNode) clone() *IdentifierExpressionNode {
	if n == nil {
		return n
	}
	c := &IdentifierExpressionNode{NodeBase: NodeBase{location: n.location}, IdentifierToken: n.IdentifierToken}
	return c
}

// ArrayLiteralNode ...
type ArrayLiteralNode struct {
	NodeBase
	OpenToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	Values       *ExpressionListNode
	CloseToken   *lexer.Token
}

// Location ...
func (n *ArrayLiteralNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ArrayLiteralNode) Clone() Node {
	return n.clone()
}

func (n *ArrayLiteralNode) clone() *ArrayLiteralNode {
	if n == nil {
		return n
	}
	c := &ArrayLiteralNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, NewlineToken: n.NewlineToken,
		Values: n.Values.clone(), CloseToken: n.CloseToken}
	return c
}

// StructLiteralNode ...
type StructLiteralNode struct {
	NodeBase
	OpenToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	Fields       []*StructLiteralFieldNode
	CloseToken   *lexer.Token
}

// Location ...
func (n *StructLiteralNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *StructLiteralNode) Clone() Node {
	return n.clone()
}

func (n *StructLiteralNode) clone() *StructLiteralNode {
	if n == nil {
		return n
	}
	c := &StructLiteralNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken,
		NewlineToken: n.NewlineToken, CloseToken: n.CloseToken}
	for _, ch := range n.Fields {
		c.Fields = append(c.Fields, ch.clone())
	}
	return c
}

// StructLiteralFieldNode ...
type StructLiteralFieldNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	NameToken    *lexer.Token
	ColonToken   *lexer.Token
	Value        Node
}

// Location ...
func (n *StructLiteralFieldNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(tloc(n.NameToken)).Join(nloc(n.Value))
	}
	return n.location
}

// Clone ...
func (n *StructLiteralFieldNode) Clone() Node {
	return n.clone()
}

func (n *StructLiteralFieldNode) clone() *StructLiteralFieldNode {
	if n == nil {
		return n
	}
	c := &StructLiteralFieldNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken,
		NameToken: n.NameToken, ColonToken: n.ColonToken, Value: clone(n.Value)}
	return c
}

// NewExpressionNode ...
type NewExpressionNode struct {
	NodeBase
	// Either TokenNew or TokenNewSlice
	NewToken *lexer.Token
	Type     Node
	// ParanthesisExpressionNode, StructLiteralNode, ArrayLiteralNode
	Value Node
}

// Location ...
func (n *NewExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.NewToken).Join(nloc(n.Type)).Join(nloc(n.Value))
	}
	return n.location
}

// Clone ...
func (n *NewExpressionNode) Clone() Node {
	return n.clone()
}

func (n *NewExpressionNode) clone() *NewExpressionNode {
	if n == nil {
		return n
	}
	c := &NewExpressionNode{NodeBase: NodeBase{location: n.location}, NewToken: n.NewToken, Type: clone(n.Type), Value: clone(n.Value)}
	return c
}

// ParanthesisExpressionNode ...
type ParanthesisExpressionNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Expression Node
	CloseToken *lexer.Token
}

// Location ...
func (n *ParanthesisExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.OpenToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// Clone ...
func (n *ParanthesisExpressionNode) Clone() Node {
	return n.clone()
}

func (n *ParanthesisExpressionNode) clone() *ParanthesisExpressionNode {
	if n == nil {
		return n
	}
	c := &ParanthesisExpressionNode{NodeBase: NodeBase{location: n.location}, OpenToken: n.OpenToken, Expression: clone(n.Expression), CloseToken: n.CloseToken}
	return c
}

// AssignmentExpressionNode ...
type AssignmentExpressionNode struct {
	NodeBase
	Left    Node
	OpToken *lexer.Token
	Right   Node
}

// Location ...
func (n *AssignmentExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Left).Join(nloc(n.Right))
	}
	return n.location
}

// Clone ...
func (n *AssignmentExpressionNode) Clone() Node {
	return n.clone()
}

func (n *AssignmentExpressionNode) clone() *AssignmentExpressionNode {
	if n == nil {
		return n
	}
	c := &AssignmentExpressionNode{NodeBase: NodeBase{location: n.location}, Left: clone(n.Left), OpToken: n.OpToken, Right: clone(n.Right)}
	return c
}

// IncrementExpressionNode ...
type IncrementExpressionNode struct {
	NodeBase
	Expression Node
	Token      *lexer.Token
}

// Location ...
func (n *IncrementExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression).Join(tloc(n.Token))
	}
	return n.location
}

// Clone ...
func (n *IncrementExpressionNode) Clone() Node {
	return n.clone()
}

func (n *IncrementExpressionNode) clone() *IncrementExpressionNode {
	if n == nil {
		return n
	}
	c := &IncrementExpressionNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), Token: n.Token}
	return c
}

// VarExpressionNode ...
type VarExpressionNode struct {
	NodeBase
	VarToken    *lexer.Token
	Names       []*VarNameNode
	Type        Node
	AssignToken *lexer.Token
	Value       Node
}

// Location ...
func (n *VarExpressionNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.VarToken)
		for _, e := range n.Names {
			n.location = n.location.Join(nloc(e))
		}
		n.location = n.location.Join(nloc(n.Type)).Join(tloc(n.AssignToken)).Join(nloc(n.Value))
	}
	return n.location
}

// Clone ...
func (n *VarExpressionNode) Clone() Node {
	return n.clone()
}

func (n *VarExpressionNode) clone() *VarExpressionNode {
	if n == nil {
		return n
	}
	c := &VarExpressionNode{NodeBase: NodeBase{location: n.location}, VarToken: n.VarToken, Type: clone(n.Type), AssignToken: n.AssignToken, Value: clone(n.Value)}
	for _, ch := range n.Names {
		c.Names = append(c.Names, ch.clone())
	}
	return c
}

// VarNameNode ...
type VarNameNode struct {
	NodeBase
	CommaToken *lexer.Token
	// Optional
	NewlineToken *lexer.Token
	NameToken    *lexer.Token
}

// Location ...
func (n *VarNameNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.CommaToken).Join(tloc(n.NameToken))
	}
	return n.location
}

// Clone ...
func (n *VarNameNode) Clone() Node {
	return n.clone()
}

func (n *VarNameNode) clone() *VarNameNode {
	if n == nil {
		return n
	}
	c := &VarNameNode{NodeBase: NodeBase{location: n.location}, CommaToken: n.CommaToken, NewlineToken: n.NewlineToken, NameToken: n.NameToken}
	return c
}

// ExpressionStatementNode ...
type ExpressionStatementNode struct {
	NodeBase
	Expression   Node
	NewlineToken *lexer.Token
}

// Location ...
func (n *ExpressionStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = nloc(n.Expression)
	}
	return n.location
}

// Clone ...
func (n *ExpressionStatementNode) Clone() Node {
	return n.clone()
}

func (n *ExpressionStatementNode) clone() *ExpressionStatementNode {
	if n == nil {
		return n
	}
	c := &ExpressionStatementNode{NodeBase: NodeBase{location: n.location}, Expression: clone(n.Expression), NewlineToken: n.NewlineToken}
	return c
}

// IfStatementNode ...
type IfStatementNode struct {
	NodeBase
	IfToken        *lexer.Token
	Statement      Node
	SemicolonToken *lexer.Token
	Expression     Node
	Body           *BodyNode
	ElseToken      *lexer.Token
	Else           Node
	NewlineToken   *lexer.Token
}

// Location ...
func (n *IfStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.IfToken).Join(nloc(n.Body)).Join(nloc(n.Else))
	}
	return n.location
}

// Clone ...
func (n *IfStatementNode) Clone() Node {
	return n.clone()
}

func (n *IfStatementNode) clone() *IfStatementNode {
	if n == nil {
		return n
	}
	c := &IfStatementNode{NodeBase: NodeBase{location: n.location}, IfToken: n.IfToken, Statement: clone(n.Statement), SemicolonToken: n.SemicolonToken,
		Expression: clone(n.Expression), Body: n.Body.clone(), ElseToken: n.ElseToken, Else: clone(n.Else), NewlineToken: n.NewlineToken}
	return c
}

// ForStatementNode ...
type ForStatementNode struct {
	NodeBase
	ForToken        *lexer.Token
	StartStatement  Node
	SemicolonToken1 *lexer.Token
	Condition       Node
	SemicolonToken2 *lexer.Token
	IncStatement    Node
	Body            *BodyNode
	NewlineToken    *lexer.Token
}

// Location ...
func (n *ForStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ForToken).Join(nloc(n.Body))
	}
	return n.location
}

// Clone ...
func (n *ForStatementNode) Clone() Node {
	return n.clone()
}

func (n *ForStatementNode) clone() *ForStatementNode {
	if n == nil {
		return n
	}
	c := &ForStatementNode{NodeBase: NodeBase{location: n.location}, ForToken: n.ForToken, StartStatement: clone(n.StartStatement),
		SemicolonToken1: n.SemicolonToken1, Condition: clone(n.Condition), SemicolonToken2: n.SemicolonToken2, IncStatement: clone(n.IncStatement),
		Body: n.Body.clone(), NewlineToken: n.NewlineToken}
	return c
}

// ContinueStatementNode ...
type ContinueStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// Location ...
func (n *ContinueStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.Token)
	}
	return n.location
}

// Clone ...
func (n *ContinueStatementNode) Clone() Node {
	return n.clone()
}

func (n *ContinueStatementNode) clone() *ContinueStatementNode {
	if n == nil {
		return n
	}
	c := &ContinueStatementNode{NodeBase: NodeBase{location: n.location}, Token: n.Token, NewlineToken: n.NewlineToken}
	return c
}

// BreakStatementNode ...
type BreakStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// Location ...
func (n *BreakStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.Token)
	}
	return n.location
}

// Clone ...
func (n *BreakStatementNode) Clone() Node {
	return n.clone()
}

func (n *BreakStatementNode) clone() *BreakStatementNode {
	if n == nil {
		return n
	}
	c := &BreakStatementNode{NodeBase: NodeBase{location: n.location}, Token: n.Token, NewlineToken: n.NewlineToken}
	return c
}

// YieldStatementNode ...
type YieldStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// Location ...
func (n *YieldStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.Token)
	}
	return n.location
}

// Clone ...
func (n *YieldStatementNode) Clone() Node {
	return n.clone()
}

func (n *YieldStatementNode) clone() *YieldStatementNode {
	if n == nil {
		return n
	}
	c := &YieldStatementNode{NodeBase: NodeBase{location: n.location}, Token: n.Token, NewlineToken: n.NewlineToken}
	return c
}

// ReturnStatementNode ...
type ReturnStatementNode struct {
	NodeBase
	ReturnToken  *lexer.Token
	Value        Node
	NewlineToken *lexer.Token
}

// Location ...
func (n *ReturnStatementNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.ReturnToken).Join(nloc(n.Value))
	}
	return n.location
}

// Clone ...
func (n *ReturnStatementNode) Clone() Node {
	return n.clone()
}

func (n *ReturnStatementNode) clone() *ReturnStatementNode {
	if n == nil {
		return n
	}
	c := &ReturnStatementNode{NodeBase: NodeBase{location: n.location}, ReturnToken: n.ReturnToken, Value: clone(n.Value), NewlineToken: n.NewlineToken}
	return c
}

// MetaAccessNode ...
type MetaAccessNode struct {
	NodeBase
	BacktickToken    *lexer.Token
	Type             Node
	BacktickDotToken *lexer.Token
	IdentifierToken  *lexer.Token
}

// Location ...
func (n *MetaAccessNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.BacktickToken).Join(tloc(n.IdentifierToken))
	}
	return n.location
}

// Clone ...
func (n *MetaAccessNode) Clone() Node {
	return n.clone()
}

func (n *MetaAccessNode) clone() *MetaAccessNode {
	if n == nil {
		return n
	}
	c := &MetaAccessNode{NodeBase: NodeBase{location: n.location}, BacktickToken: n.BacktickToken, Type: clone(n.Type), BacktickDotToken: n.BacktickDotToken,
		IdentifierToken: n.IdentifierToken}
	return c
}

// SetTypeAnnotation ...
func (n *NodeBase) SetTypeAnnotation(t interface{}) {
	n.typeAnnotation = t
}

// TypeAnnotation ...
func (n *NodeBase) TypeAnnotation() interface{} {
	return n.typeAnnotation
}

// SetScope ...
func (n *NodeBase) SetScope(s interface{}) {
	n.scope = s
}

// Scope ...
func (n *NodeBase) Scope() interface{} {
	return n.scope
}

/*******************************************
 *
 * Helper functions
 *
 *******************************************/

func nloc(n Node) errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	return n.Location()
}

func aloc(arr []Node) errlog.LocationRange {
	var loc errlog.LocationRange
	for _, n := range arr {
		loc = loc.Join(nloc(n))
	}
	return loc
}

func tloc(t *lexer.Token) errlog.LocationRange {
	if t == nil {
		return errlog.LocationRange{}
	}
	return t.Location
}
