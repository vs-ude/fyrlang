package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genStatement(ast parser.Node, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) {
	b.SetLocation(ast.Location())
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		genExpression(n.Expression, s, b, p, vars)
		return
	case *parser.IfStatementNode:
		if n.Statement != nil {
			genStatement(n.Statement, s, b, p, vars)
		}
		cond := genExpression(n.Expression, s, b, p, vars)
		b.If(cond)
		genBody(n.Body, s, b, p, vars)
		if n.Else != nil {
			b.Else()
			if block, ok := n.Else.(*parser.BodyNode); ok {
				genBody(block, s, b, p, vars)
			} else {
				genStatement(n.Else, s, b, p, vars)
			}
			b.End()
		}
		b.End()
		return
	case *parser.ForStatementNode:
		s2 := n.Scope().(*types.Scope)
		if n.StartStatement != nil {
			genStatement(n.StartStatement, s2, b, p, vars)
		}
		b.Loop()
		if n.Condition != nil {
			cond := genExpression(n.Condition, s2, b, p, vars)
			cond2 := b.BooleanNot(nil, cond)
			b.If(ircode.NewVarArg(cond2))
			b.Break(0)
			b.End()
		}
		genBody(n.Body, s2, b, p, vars)
		if n.IncStatement != nil {
			genStatement(n.IncStatement, s2, b, p, vars)
		}
		b.End()
		return
	case *parser.ReturnStatementNode:
	case *parser.ContinueStatementNode:
		b.Continue(0)
		return
	case *parser.BreakStatementNode:
		b.Break(0)
		return
	case *parser.YieldStatementNode:
		// TODO
	case *parser.LineNode:
		// Do nothing
		return
	}
	panic("Should not happen")
}
