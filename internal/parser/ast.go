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

// GenericParamNode ...
type GenericParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
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

// ParamNode ...
type ParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
	Type       Node
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

// GenericInstanceFuncNode ...
type GenericInstanceFuncNode struct {
	NodeBase
	Expression    Node
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

// GroupTypeNode ...
type GroupTypeNode struct {
	NodeBase
	// Either `-` or `->`
	GroupToken *lexer.Token
	// Is null in the case of `->`
	GroupNameToken *lexer.Token
	Type           Node
}

// Location ...
func (n *GroupTypeNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.GroupToken).Join(tloc(n.GroupNameToken)).Join(nloc(n.Type))
	}
	return n.location
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

// TypeListNode ...
type TypeListNode struct {
	NodeBase
	BacktickToken *lexer.Token
	OpenToken     *lexer.Token
	Types         []*TypeListElementNode
	CloseToken    *lexer.Token
}

// Location ...
func (n *TypeListNode) Location() errlog.LocationRange {
	if n == nil {
		return errlog.LocationRange{}
	}
	if n.location.IsNull() {
		n.location = tloc(n.BacktickToken).Join(tloc(n.CloseToken))
	}
	return n.location
}

// TypeListElementNode ...
type TypeListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	Type       Node
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

// ExpressionListElementNode ...
type ExpressionListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	Expression Node
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

// BinaryExpressionNode ...
type BinaryExpressionNode struct {
	NodeBase
	Left    Node
	OpToken *lexer.Token
	Right   Node
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

// ArrayLiteralNode ...
type ArrayLiteralNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Values     *ExpressionListNode
	CloseToken *lexer.Token
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

// StructLiteralNode ...
type StructLiteralNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Fields     []*StructLiteralFieldNode
	CloseToken *lexer.Token
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

// StructLiteralFieldNode ...
type StructLiteralFieldNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
	ColonToken *lexer.Token
	Value      Node
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

// NewExpressionNode ...
type NewExpressionNode struct {
	NodeBase
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

// VarNameNode ...
type VarNameNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
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
