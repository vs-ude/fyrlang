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
		if n.Statement != nil {
			if err := checkStatement(n.Statement, s, log); err != nil {
				return err
			}
		}
		if err := checkExpression(n.Expression, s, log); err != nil {
			return err
		}
		s2 := newScope(s, IfScope, n.Body.Location())
		if err := checkBody(n.Body, s2, log); err != nil {
			return err
		}
		return nil
	case *parser.ForStatementNode:
		s2 := newScope(s, ForScope, n.Location())
		if n.StartStatement != nil {
			if err := checkStatement(n.StartStatement, s2, log); err != nil {
				return err
			}
		}
		if n.Condition != nil {
			if err := checkExpression(n.Condition, s, log); err != nil {
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
	panic("Should not happen")
}
