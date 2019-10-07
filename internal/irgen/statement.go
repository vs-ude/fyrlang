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
		s2 := n.Body.Scope().(*types.Scope)
		genBody(n.Body, s2, b, p, vars)
		if n.Else != nil {
			s3 := n.Else.Scope().(*types.Scope)
			b.Else()
			if block, ok := n.Else.(*parser.BodyNode); ok {
				genBody(block, s3, b, p, vars)
			} else {
				genStatement(n.Else, s3, b, p, vars)
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
		genReturnStatement(n, s, b, p, vars)
		return
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

func genReturnStatement(n *parser.ReturnStatementNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) {
	fs := s.FunctionScope()
	f := fs.Func
	if f.Type.HasNamedReturnVariables() && n.Value == nil {
		// Return the named variables
		var args []ircode.Argument
		for _, p := range f.Type.Out.Params {
			element := fs.GetElement(p.Name)
			if element == nil {
				panic("Oooops")
			}
			e, ok := element.(*types.Variable)
			if !ok {
				panic("Oooops")
			}
			v, ok := vars[e]
			if !ok {
				panic("Ooooops")
			}
			args = append(args, ircode.NewVarArg(v))
		}
		println("RETURN TYPE", f.Type.ReturnType().ToString())
		b.Return(f.Type.ReturnType(), args...)
		return
	}
	if n.Value == nil {
		// Return nothing
		b.Return(f.Type.ReturnType())
		return
	}
	// Multiple return values
	if l, ok := n.Value.(*parser.ExpressionListNode); ok {
		var args []ircode.Argument
		for _, element := range l.Elements {
			args = append(args, genExpression(element.Expression, s, b, p, vars))
		}
		b.Return(f.Type.ReturnType(), args...)
		return
	}
	arg := genExpression(n.Value, s, b, p, vars)
	b.Return(f.Type.ReturnType(), arg)
}
