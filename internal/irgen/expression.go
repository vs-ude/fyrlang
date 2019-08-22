package irgen

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genExpression(ast parser.Node, s *types.Scope, b *ircode.Builder) {
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			genExpression(e.Expression, s, b)
		}
		return
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
		genExpression(n.Expression, s, b)
		return
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
	// panic("Should not happen")
}
