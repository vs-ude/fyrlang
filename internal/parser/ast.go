package parser

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
)

// Node ...
type Node interface {
	SetTypeAnnotation(t interface{})
	TypeAnnotation() interface{}
}

// NodeBase ...
type NodeBase struct {
	typeAnnotation interface{}
}

// FileNode ...
type FileNode struct {
	NodeBase
	File     int
	Children []Node
}

// LineNode ...
type LineNode struct {
	NodeBase
	Token *lexer.Token
}

// ComponentNode ...
type ComponentNode struct {
	NodeBase
	ComponentToken *lexer.Token
	NameToken      *lexer.Token
	NewlineToken   *lexer.Token
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

// ImportNode ...
type ImportNode struct {
	NodeBase
	NameToken    *lexer.Token
	StringToken  *lexer.Token
	NewlineToken *lexer.Token
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

// GenericParamListNode ...
type GenericParamListNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Params     []*GenericParamNode
	CloseToken *lexer.Token
}

// GenericParamNode ...
type GenericParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
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

// ParamListNode ...
type ParamListNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Params     []*ParamNode
	CloseToken *lexer.Token
}

// ParamNode ...
type ParamNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
	Type       Node
}

// GenericInstanceFuncNode ...
type GenericInstanceFuncNode struct {
	NodeBase
	Expression    Node
	TypeArguments *TypeListNode
}

// BodyNode ...
type BodyNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Children   []Node
	CloseToken *lexer.Token
}

// NamedTypeNode ...
type NamedTypeNode struct {
	NodeBase
	Namespace         *NamedTypeNode
	NamespaceDotToken *lexer.Token
	NameToken         *lexer.Token
}

// PointerTypeNode ...
type PointerTypeNode struct {
	NodeBase
	PointerToken *lexer.Token
	ElementType  Node
}

// MutableTypeNode ...
type MutableTypeNode struct {
	NodeBase
	MutToken *lexer.Token
	Type     Node
}

// SliceTypeNode ...
type SliceTypeNode struct {
	NodeBase
	OpenToken   *lexer.Token
	CloseToken  *lexer.Token
	ElementType Node
}

// ArrayTypeNode ...
type ArrayTypeNode struct {
	NodeBase
	OpenToken   *lexer.Token
	Size        Node
	CloseToken  *lexer.Token
	ElementType Node
}

// ClosureTypeNode ...
type ClosureTypeNode struct {
	NodeBase
	FuncToken    *lexer.Token
	Params       *ParamListNode
	ReturnParams *ParamListNode
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

// StructFieldNode ...
type StructFieldNode struct {
	NodeBase
	NameToken    *lexer.Token
	Type         Node
	NewlineToken *lexer.Token
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

// InterfaceFuncNode ...
type InterfaceFuncNode struct {
	NodeBase
	ComponentMutToken *lexer.Token
	FuncToken         *lexer.Token
	MutToken          *lexer.Token
	PointerToken      *lexer.Token
	NameToken         *lexer.Token
	Params            *ParamListNode
	ReturnParams      *ParamListNode
	NewlineToken      *lexer.Token
}

// InterfaceFieldNode ...
type InterfaceFieldNode struct {
	NodeBase
	Type         Node
	NewlineToken *lexer.Token
}

// GroupTypeNode ...
type GroupTypeNode struct {
	NodeBase
	ColonToken     *lexer.Token
	GroupNameToken *lexer.Token
	Type           Node
}

// GenericInstanceTypeNode ...
type GenericInstanceTypeNode struct {
	NodeBase
	Type          *NamedTypeNode
	TypeArguments *TypeListNode
}

// TypeListNode ...
type TypeListNode struct {
	NodeBase
	BacktickToken *lexer.Token
	OpenToken     *lexer.Token
	Types         []*TypeListElementNode
	CloseToken    *lexer.Token
}

// TypeListElementNode ...
type TypeListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	Type       Node
}

// ExpressionListNode ...
type ExpressionListNode struct {
	NodeBase
	Elements []*ExpressionListElementNode
}

// ExpressionListElementNode ...
type ExpressionListElementNode struct {
	NodeBase
	CommaToken *lexer.Token
	Expression Node
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

// BinaryExpressionNode ...
type BinaryExpressionNode struct {
	NodeBase
	Left    Node
	OpToken *lexer.Token
	Right   Node
}

// UnaryExpressionNode ...
type UnaryExpressionNode struct {
	NodeBase
	OpToken    *lexer.Token
	Expression Node
}

// IsTypeExpressionNode ...
type IsTypeExpressionNode struct {
	NodeBase
	Expression Node
	IsToken    *lexer.Token
	Type       Node
}

// MemberAccessExpressionNode ...
type MemberAccessExpressionNode struct {
	NodeBase
	Expression      Node
	DotToken        *lexer.Token
	IdentifierToken *lexer.Token
}

// MemberCallExpressionNode ...
type MemberCallExpressionNode struct {
	NodeBase
	Expression Node
	OpenToken  *lexer.Token
	Arguments  *ExpressionListNode
	CloseToken *lexer.Token
}

// ArrayAccessExpressionNode ...
type ArrayAccessExpressionNode struct {
	NodeBase
	Expression Node
	OpenToken  *lexer.Token
	Index      Node
	CloseToken *lexer.Token
}

// ConstantExpressionNode ...
type ConstantExpressionNode struct {
	NodeBase
	ValueToken *lexer.Token
}

// IdentifierExpressionNode ...
type IdentifierExpressionNode struct {
	NodeBase
	IdentifierToken *lexer.Token
}

// ArrayLiteralNode ...
type ArrayLiteralNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Values     *ExpressionListNode
	CloseToken *lexer.Token
}

// StructLiteralNode ...
type StructLiteralNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Fields     []*StructLiteralFieldNode
	CloseToken *lexer.Token
}

// StructLiteralFieldNode ...
type StructLiteralFieldNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
	ColonToken *lexer.Token
	Value      Node
}

// NewExpressionNode ...
type NewExpressionNode struct {
	NodeBase
	NewToken *lexer.Token
	Type     Node
	Value    Node
}

// ParanthesisExpressionNode ...
type ParanthesisExpressionNode struct {
	NodeBase
	OpenToken  *lexer.Token
	Expression Node
	CloseToken *lexer.Token
}

// AssignmentExpressionNode ...
type AssignmentExpressionNode struct {
	NodeBase
	Left    Node
	OpToken *lexer.Token
	Right   Node
}

// IncrementExpressionNode ...
type IncrementExpressionNode struct {
	NodeBase
	Expression Node
	Token      *lexer.Token
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

// VarNameNode ...
type VarNameNode struct {
	NodeBase
	CommaToken *lexer.Token
	NameToken  *lexer.Token
}

// ExpressionStatementNode ...
type ExpressionStatementNode struct {
	NodeBase
	Expression   Node
	NewlineToken *lexer.Token
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

// ContinueStatementNode ...
type ContinueStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// BreakStatementNode ...
type BreakStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// YieldStatementNode ...
type YieldStatementNode struct {
	NodeBase
	Token        *lexer.Token
	NewlineToken *lexer.Token
}

// ReturnStatementNode ...
type ReturnStatementNode struct {
	NodeBase
	ReturnToken  *lexer.Token
	Value        Node
	NewlineToken *lexer.Token
}

// SetTypeAnnotation ...
func (n *NodeBase) SetTypeAnnotation(t interface{}) {
	n.typeAnnotation = t
}

// TypeAnnotation ...
func (n *NodeBase) TypeAnnotation() interface{} {
	return n.typeAnnotation
}

// LocationRange ...
func (n *NamedTypeNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.NameToken.Location
}

// LocationRange ...
func (n *PointerTypeNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.PointerToken.Location
}

// LocationRange ...
func (n *SliceTypeNode) LocationRange() errlog.LocationRange {
	// TODO
	return errlog.LocationRange{From: n.OpenToken.Location.From, To: n.CloseToken.Location.To}
}

// LocationRange ...
func (n *ArrayTypeNode) LocationRange() errlog.LocationRange {
	// TODO
	return errlog.LocationRange{From: n.OpenToken.Location.From, To: n.CloseToken.Location.To}
}

// LocationRange ...
func (n *StructFieldNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.NewlineToken.Location
}

// LocationRange ...
func (n *InterfaceFieldNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.NewlineToken.Location
}

// LocationRange ...
func (n *InterfaceFuncNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.NewlineToken.Location
}

// LocationRange ...
func (n *ParamNode) LocationRange() errlog.LocationRange {
	// TODO
	if n.NameToken != nil {
		return n.NameToken.Location
	}
	return errlog.LocationRange{}
}

// LocationRange ...
func (n *ClosureTypeNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.FuncToken.Location
}

// LocationRange ...
func (n *FuncNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.FuncToken.Location
}

// LocationRange ...
func (n *ConstantExpressionNode) LocationRange() errlog.LocationRange {
	// TODO
	return n.ValueToken.Location
}

// NodeLocation ...
func NodeLocation(n Node) errlog.LocationRange {
	return errlog.LocationRange{From: 0, To: 0}
}
