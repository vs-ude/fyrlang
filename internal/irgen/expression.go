package irgen

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func transformExpression(ast parser.Node, s *types.Scope, b *ircode.Builder) {
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			transformExpression(e.Expression, s, b)
		}
	case *parser.BinaryExpressionNode:
	case *parser.UnaryExpressionNode:
	case *parser.IsTypeExpressionNode:
	case *parser.MemberAccessExpressionNode:
	case *parser.MemberCallExpressionNode:
	case *parser.ArrayAccessExpressionNode:
	case *parser.ConstantExpressionNode:
	case *parser.IdentifierExpressionNode:
	case *parser.NewExpressionNode:
	case *parser.ParanthesisExpressionNode:
		transformExpression(n.Expression, s, b)
	case *parser.AssignmentExpressionNode:
		//		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
		//			return checkAssignExpression(n, s, log)
		//		}
		//		panic("TODO")
	case *parser.IncrementExpressionNode:
	case *parser.VarExpressionNode:
	case *parser.ArrayLiteralNode:
	case *parser.StructLiteralNode:
	case *parser.ClosureExpressionNode:
	}
	fmt.Printf("%T\n", ast)
	panic("Should not happen")
}
