package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genStatement(ast parser.Node, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) {
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		genExpression(n.Expression, s, b, vars)
		return
	case *parser.IfStatementNode:
	case *parser.ForStatementNode:
	case *parser.ReturnStatementNode:
	case *parser.ContinueStatementNode:
		b.Continue(0)
		return
	case *parser.BreakStatementNode:
		b.Break(0)
		return
	case *parser.YieldStatementNode:
	case *parser.LineNode:
		// Do nothing
		return
	}
	panic("Should not happen")
}
