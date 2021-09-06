package types

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func checkStatement(ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	switch n := ast.(type) {
	case *parser.ExpressionStatementNode:
		return checkExpression(n.Expression, s, log)
	case *parser.IfStatementNode:
		if n.Statement != nil {
			if err := checkStatement(n.Statement, s, log); err != nil {
				return err
			}
		}
		if err := checkExpression(n.Expression, s, log); err != nil {
			return err
		}
		if err := expectType(n.Expression, boolType, log); err != nil {
			return err
		}
		s2 := newScope(s, IfScope, n.Body.Location())
		n.Body.SetScope(s2)
		if err := checkBody(n.Body, s2, log); err != nil {
			return err
		}
		if n.Else != nil {
			s3 := newScope(s, IfScope, n.Else.Location())
			n.Else.SetScope(s3)
			if block, ok := n.Else.(*parser.BodyNode); ok {
				if err := checkBody(block, s3, log); err != nil {
					return err
				}
			} else if err := checkStatement(n.Else, s3, log); err != nil {
				return err
			}
		}
		return nil
	case *parser.ForStatementNode:
		s2 := newScope(s, ForScope, n.Location())
		n.SetScope(s2)
		if n.StartStatement != nil {
			if err := checkStatement(n.StartStatement, s2, log); err != nil {
				return err
			}
		}
		if n.Condition != nil {
			if err := checkExpression(n.Condition, s2, log); err != nil {
				return err
			}
			if err := expectType(n.Condition, boolType, log); err != nil {
				return err
			}
		}
		if n.IncStatement != nil {
			if err := checkStatement(n.IncStatement, s2, log); err != nil {
				return err
			}
		}
		if err := checkBody(n.Body, s2, log); err != nil {
			return err
		}
		return nil
	case *parser.ReturnStatementNode:
		return checkReturnStatement(n, s, log)
	case *parser.DeleteStatementNode:
		return checkDeleteStatement(n, s, log)
	case *parser.ContinueStatementNode:
		if s.ForScope() == nil {
			log.AddError(errlog.ErrorContinueOutsideLoop, n.Location())
		}
		// Do nothing
		return nil
	case *parser.BreakStatementNode:
		if s.ForScope() == nil {
			log.AddError(errlog.ErrorBreakOutsideLoopOrSwitch, n.Location())
		}
		// Do nothing
		return nil
	case *parser.YieldStatementNode:
		// Do nothing
		return nil
	case *parser.LineNode:
		// Do nothing
		return nil
	}
	fmt.Printf("%T", ast)
	panic("Should not happen")
}

func checkReturnStatement(n *parser.ReturnStatementNode, s *Scope, log *errlog.ErrorLog) error {
	f := s.FunctionScope().Func
	if f.Type.HasNamedReturnVariables() {
		if n.Value == nil {
			return nil
		}
	}
	// Return no values?
	if n.Value == nil {
		if len(f.Type.Out.Params) == 0 {
			return nil
		}
		return log.AddError(errlog.ErrorParameterCountMismatch, n.Location())
	}
	// Return a list of values, e.g. `return a, b`?
	if l, ok := n.Value.(*parser.ExpressionListNode); ok {
		if len(f.Type.Out.Params) != len(l.Elements) {
			return log.AddError(errlog.ErrorParameterCountMismatch, n.Location())
		}
		for i, p := range f.Type.Out.Params {
			if err := checkExpression(l.Elements[i].Expression, s, log); err != nil {
				return err
			}
			if err := checkExprEqualType(NewExprType(p.Type), exprType(l.Elements[i].Expression), Assignable, l.Elements[i].Expression.Location(), log); err != nil {
				return err
			}
		}
	} else {
		// Return a single value, e.g. `return a`
		if err := checkExpression(n.Value, s, log); err != nil {
			return err
		}
		et := exprType(n.Value)
		if st, ok := GetStructType(et.Type); ok && st.Name() == "" {
			// The return type is a tuple and the single value can be decomposed, because it is a struct
			if len(st.Fields) != len(f.Type.Out.Params) {
				return log.AddError(errlog.ErrorParameterCountMismatch, n.Location())
			}
			for i, field := range st.Fields {
				if err := checkExprEqualType(NewExprType(f.Type.Out.Params[i].Type), et.Field(field), Assignable, n.Value.Location(), log); err != nil {
					return err
				}
			}
		} else {
			if len(f.Type.Out.Params) != 1 {
				return log.AddError(errlog.ErrorParameterCountMismatch, n.Location())
			}
			if err := checkExprEqualType(NewExprType(f.Type.Out.Params[0].Type), et, Assignable, n.Value.Location(), log); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDeleteStatement(n *parser.DeleteStatementNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Value, s, log); err != nil {
		return err
	}
	et := exprType(n.Value)
	pt := et.PointerTarget()
	if pt == nil {
		return log.AddError(errlog.ErrorWrongTypeForDelete, n.Location())
	}
	if !pt.Mutable {
		return log.AddError(errlog.ErrorWrongTypeForDelete, n.Location())
	}
	_, ok := GetStructType(pt.Type)
	if !ok {
		return log.AddError(errlog.ErrorWrongTypeForDelete, n.Location())
	}
	return nil
}
