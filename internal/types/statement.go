package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func checkStatement(ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		return checkExpression(n.Expression, s, log)
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
