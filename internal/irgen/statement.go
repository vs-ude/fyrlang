package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func transformStatement(ast parser.Node, s *types.Scope, b *ircode.Builder) {
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		transformExpression(n.Expression, s, b)
	case *parser.IfStatementNode:
	case *parser.ForStatementNode:
	case *parser.ReturnStatementNode:
	case *parser.ContinueStatementNode:
		// Do nothing
	case *parser.BreakStatementNode:
		// Do nothing
	case *parser.YieldStatementNode:
		// Do nothing
	case *parser.LineNode:
		// Do nothing
	}
	panic("Should not happen")
}
