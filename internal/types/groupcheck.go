package types

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func gcheckFuncBody(f *Func, log *errlog.ErrorLog) error {
	println("GROUP CHECK FUNC", f.Name())
	return gcheckBody(f.ast.Body, f.InnerScope, log)
}

func gcheckBody(ast *parser.BodyNode, s *Scope, log *errlog.ErrorLog) error {
	var err error
	for _, ch := range ast.Children {
		err2 := gcheckStatement(ch, s, log)
		if err2 != nil {
			if err == nil {
				err = err2
			}
		}
	}
	return nil
}

func gcheckStatement(ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		return gcheckExpression(n.Expression, s, log)
	case *parser.IfStatementNode:
	case *parser.ForStatementNode:
	case *parser.ReturnStatementNode:
	case *parser.ContinueStatementNode:
		// Do nothing
		return nil
	case *parser.BreakStatementNode:
		// Do nothing
		return nil
	case *parser.YieldStatementNode:
		// Do nothing
		return nil
	case *parser.LineNode:
		// Do nothing
		return nil
	}
	panic("Should not happen")
}

func gcheckExpression(ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			if err := gcheckExpression(e.Expression, s, log); err != nil {
				return err
			}
		}
		return nil
	case *parser.BinaryExpressionNode:
		//		return gcheckBinaryExpression(n, s, log)
	case *parser.UnaryExpressionNode:
		//		return gcheckUnaryExpression(n, s, log)
	case *parser.IsTypeExpressionNode:
	case *parser.MemberAccessExpressionNode:
	case *parser.MemberCallExpressionNode:
	case *parser.ArrayAccessExpressionNode:
	case *parser.ConstantExpressionNode:
		// Nothing to do here
		return nil
	case *parser.IdentifierExpressionNode:
		//		return gcheckIdentifierExpression(n, s, log)
	case *parser.NewExpressionNode:
	case *parser.ParanthesisExpressionNode:
		err := gcheckExpression(n.Expression, s, log)
		return err
	case *parser.AssignmentExpressionNode:
		//		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
		//			return checkAssignExpression(n, s, log)
		//		}
		//		panic("TODO")
	case *parser.IncrementExpressionNode:
		//		return gcheckIncrementExpression(n, s, log)
	case *parser.VarExpressionNode:
		//		return gcheckVarExpression(n, s, log)
	case *parser.ArrayLiteralNode:
		//		return gcheckArrayLiteralExpression(n, s, log)
	case *parser.StructLiteralNode:
	case *parser.ClosureExpressionNode:
	}
	fmt.Printf("%T\n", ast)
	panic("Should not happen")
}
