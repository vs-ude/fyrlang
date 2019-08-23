package types

import (
	"fmt"
	"math/big"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/lexer"
	"github.com/vs-ude/fyrlang/internal/parser"
)

func checkExpression(ast parser.Node, s *Scope, log *errlog.ErrorLog) error {
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			if err := checkExpression(e.Expression, s, log); err != nil {
				return err
			}
		}
		return nil
	case *parser.BinaryExpressionNode:
		return checkBinaryExpression(n, s, log)
	case *parser.UnaryExpressionNode:
		return checkUnaryExpression(n, s, log)
	case *parser.IsTypeExpressionNode:
	case *parser.MemberAccessExpressionNode:
	case *parser.MemberCallExpressionNode:
	case *parser.ArrayAccessExpressionNode:
	case *parser.ConstantExpressionNode:
		return checkConstExpression(n, s, log)
	case *parser.IdentifierExpressionNode:
		return checkIdentifierExpression(n, s, log)
	case *parser.NewExpressionNode:
	case *parser.ParanthesisExpressionNode:
		err := checkExpression(n.Expression, s, log)
		ast.SetTypeAnnotation(n.Expression.TypeAnnotation())
		return err
	case *parser.AssignmentExpressionNode:
		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
			return checkAssignExpression(n, s, log)
		}
		panic("TODO")
	case *parser.IncrementExpressionNode:
		return checkIncrementExpression(n, s, log)
	case *parser.VarExpressionNode:
		return checkVarExpression(n, s, log)
	case *parser.ArrayLiteralNode:
		return checkArrayLiteralExpression(n, s, log)
	case *parser.StructLiteralNode:
	case *parser.ClosureExpressionNode:
	}
	fmt.Printf("%T\n", ast)
	panic("Should not happen")
}

func checkConstExpression(n *parser.ConstantExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	switch n.ValueToken.Kind {
	case lexer.TokenFalse:
		n.SetTypeAnnotation(&ExprType{Type: boolType, BoolValue: false, HasValue: true})
		return nil
	case lexer.TokenTrue:
		n.SetTypeAnnotation(&ExprType{Type: boolType, BoolValue: true, HasValue: true})
		return nil
	case lexer.TokenNull:
		n.SetTypeAnnotation(&ExprType{Type: nullType, HasValue: true})
		return nil
	case lexer.TokenInteger:
		n.SetTypeAnnotation(&ExprType{Type: integerType, IntegerValue: n.ValueToken.IntegerValue, HasValue: true})
		return nil
	case lexer.TokenHex:
		n.SetTypeAnnotation(&ExprType{Type: integerType, IntegerValue: n.ValueToken.IntegerValue, HasValue: true})
		return nil
	case lexer.TokenOctal:
		n.SetTypeAnnotation(&ExprType{Type: integerType, IntegerValue: n.ValueToken.IntegerValue, HasValue: true})
		return nil
	case lexer.TokenFloat:
		n.SetTypeAnnotation(&ExprType{Type: floatType, FloatValue: n.ValueToken.FloatValue, HasValue: true})
		return nil
	case lexer.TokenString:
		n.SetTypeAnnotation(&ExprType{Type: stringType, StringValue: n.ValueToken.StringValue, HasValue: true})
		return nil
	case lexer.TokenRune:
		n.SetTypeAnnotation(&ExprType{Type: runeType, RuneValue: n.ValueToken.RuneValue, HasValue: true})
		return nil
	}
	panic("Should not happen")
}

func checkArrayLiteralExpression(n *parser.ArrayLiteralNode, s *Scope, log *errlog.ErrorLog) error {
	et := &ExprType{Type: arrayLiteralType, HasValue: true}
	for _, e := range n.Values.Elements {
		if err := checkExpression(e.Expression, s, log); err != nil {
			return err
		}
		et.ArrayValue = append(et.ArrayValue, exprType(e.Expression))
	}
	n.SetTypeAnnotation(et)
	return nil
}

func checkIncrementExpression(n *parser.IncrementExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	if err := checkIsAssignable(n.Expression, log); err != nil {
		return err
	}
	et := exprType(n.Expression)
	if !IsIntegerType(et.Type) && !IsFloatType(et.Type) {
		return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
	}
	return nil
}

func checkVarExpression(n *parser.VarExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	var err error
	var values []parser.Node
	if n.Value != nil {
		if err = checkExpression(n.Value, s, log); err != nil {
			return err
		}
		if list, ok := n.Value.(*parser.ExpressionListNode); ok {
			for _, el := range list.Elements {
				values = append(values, el.Expression)
			}
		} else {
			values = []parser.Node{n.Value}
		}
	}
	if n.Type != nil {
		/*
		 * Assignment with type definition
		 */
		typ, err := parseType(n.Type, s, log)
		if err != nil {
			return err
		}
		et := makeExprType(typ)
		et.Group = s.Group
		if n.VarToken.Kind == lexer.TokenVar {
			et.Mutable = true
		}
		if n.Value != nil && len(n.Names) != len(values) {
			if len(values) != 1 {
				return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
			}
			vet := exprType(n.Value)
			if err = checkInstantiableExprType(vet, s, n.Value.Location(), log); err != nil {
				return err
			}
			if st, ok := vet.Type.(*StructType); ok {
				/*
				 * Right-hand side is a struct
				 */
				// TODO: Only use the accessible fields here, which depends on the package
				if len(n.Names) != len(st.Fields) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				for i, name := range n.Names {
					fet := &ExprType{Type: st.Fields[i].Type, Group: s.Group, PointerDestGroup: vet.PointerDestGroup, PointerDestMutable: vet.PointerDestMutable}
					name.SetTypeAnnotation(et)
					if err = checkExprEqualType(et, fet, Assignable, n.Location(), log); err != nil {
						return err
					}
					if n.VarToken.Kind == lexer.TokenVar {
						et.Mutable = true
					}
					v := &Variable{name: name.NameToken.StringValue, Type: et}
					err = s.AddElement(v, name.Location(), log)
					if err != nil {
						return err
					}
				}
				return nil
			} else if a, ok := vet.Type.(*ArrayType); ok {
				/*
				 * Right-hand side is an array
				 */
				if a.Size >= (1<<32) || len(n.Names) != int(a.Size) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				for _, name := range n.Names {
					fet := &ExprType{Type: a.ElementType, Group: s.Group, PointerDestGroup: vet.PointerDestGroup, PointerDestMutable: vet.PointerDestMutable}
					name.SetTypeAnnotation(et)
					if err = checkExprEqualType(et, fet, Assignable, n.Location(), log); err != nil {
						return err
					}
					if n.VarToken.Kind == lexer.TokenVar {
						et.Mutable = true
					}
					v := &Variable{name: name.NameToken.StringValue, Type: et}
					err = s.AddElement(v, name.Location(), log)
					if err != nil {
						return err
					}
				}
				return nil
			}
			return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
		}
		/*
		 * Single value on right-hand side
		 */
		for i, name := range n.Names {
			name.SetTypeAnnotation(et)
			v := &Variable{name: name.NameToken.StringValue, Type: et}
			err = s.AddElement(v, name.Location(), log)
			if err != nil {
				return err
			}
			if n.Value != nil {
				if len(n.Names) != len(values) {
					if len(values) != 1 {
						return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
					}
					panic("TODO: Destructive assign")
				}
				vet := exprType(values[i])
				if needsTypeInference(vet) {
					if err = inferType(vet, et, n.Location(), log); err != nil {
						return err
					}
				} else {
					if err = checkExprEqualType(et, vet, Assignable, n.Location(), log); err != nil {
						return err
					}
				}
			}
		}
	} else {
		/*
		 * Assignment without type definition
		 */
		if n.Value == nil {
			return log.AddError(errlog.ErrorVarWithoutType, n.Location())
		}
		if len(n.Names) != len(values) {
			if len(values) != 1 {
				return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
			}
			vet := exprType(n.Value)
			if err = checkInstantiableExprType(vet, s, n.Value.Location(), log); err != nil {
				return err
			}
			if st, ok := vet.Type.(*StructType); ok {
				/*
				 * Right-hand side is a struct
				 */
				// TODO: Only use the accessible fields here, which depends on the package
				if len(n.Names) != len(st.Fields) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				for i, name := range n.Names {
					et := makeExprType(st.Fields[i].Type)
					et.Group = s.Group
					et.PointerDestGroup = vet.PointerDestGroup
					et.PointerDestMutable = vet.PointerDestMutable
					if n.VarToken.Kind == lexer.TokenVar {
						et.Mutable = true
					}
					name.SetTypeAnnotation(et)
					v := &Variable{name: name.NameToken.StringValue, Type: et}
					err = s.AddElement(v, name.Location(), log)
					if err != nil {
						return err
					}
				}
				return nil
			} else if a, ok := vet.Type.(*ArrayType); ok {
				/*
				 * Right-hand side is an array
				 */
				if a.Size >= (1<<32) || len(n.Names) != int(a.Size) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				for _, name := range n.Names {
					et := makeExprType(a.ElementType)
					et.Group = s.Group
					et.PointerDestGroup = vet.PointerDestGroup
					et.PointerDestMutable = vet.PointerDestMutable
					if n.VarToken.Kind == lexer.TokenVar {
						et.Mutable = true
					}
					name.SetTypeAnnotation(et)
					v := &Variable{name: name.NameToken.StringValue, Type: et}
					err = s.AddElement(v, name.Location(), log)
					if err != nil {
						return err
					}
				}
				return nil
			}
			return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
		}
		/*
		 * Single value on right-hand side
		 */
		for i, name := range n.Names {
			etRight := exprType(values[i])
			if err = checkInstantiableExprType(etRight, s, values[i].Location(), log); err != nil {
				return err
			}
			et := makeExprType(etRight.Type)
			et.Group = s.Group
			et.PointerDestGroup = etRight.PointerDestGroup
			et.PointerDestMutable = etRight.PointerDestMutable
			if n.VarToken.Kind == lexer.TokenVar {
				et.Mutable = true
			}
			name.SetTypeAnnotation(et)
			v := &Variable{name: name.NameToken.StringValue, Type: et}
			err = s.AddElement(v, name.Location(), log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func checkAssignExpression(n *parser.AssignmentExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	values := []parser.Node{}
	if err := checkExpression(n.Right, s, log); err != nil {
		return err
	}
	if list, ok := n.Right.(*parser.ExpressionListNode); ok {
		for _, el := range list.Elements {
			values = append(values, el.Expression)
		}
	} else {
		values = []parser.Node{n.Right}
	}
	if n.OpToken.Kind == lexer.TokenWalrus {
		var dests []*parser.IdentifierExpressionNode
		if list, ok := n.Left.(*parser.ExpressionListNode); ok {
			for _, e := range list.Elements {
				left, ok := e.Expression.(*parser.IdentifierExpressionNode)
				if !ok {
					log.AddError(errlog.ErrorExpectedVariable, n.Left.Location())
				}
				dests = append(dests, left)
			}
		} else {
			left, ok := n.Left.(*parser.IdentifierExpressionNode)
			if !ok {
				log.AddError(errlog.ErrorExpectedVariable, n.Left.Location())
			}
			dests = []*parser.IdentifierExpressionNode{left}
		}
		if len(values) != len(dests) {
			if len(values) != 1 {
				return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
			}
			ret := exprType(n.Right)
			if err := checkInstantiableExprType(ret, s, n.Right.Location(), log); err != nil {
				return err
			}
			if st, ok := ret.Type.(*StructType); ok {
				/*
				 * Right-hand side is a struct
				 */
				// TODO: Only use the accessible fields here, which depends on the package
				if len(dests) != len(st.Fields) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				newCount := 0
				for i, dest := range dests {
					et := makeExprType(st.Fields[i].Type)
					et.Mutable = true
					et.Group = s.Group
					et.PointerDestGroup = ret.PointerDestGroup
					et.PointerDestMutable = ret.PointerDestMutable
					if v := s.lookupVariable(dest.IdentifierToken.StringValue); v != nil {
						if err := checkExprEqualType(v.Type, et, Assignable, dest.Location(), log); err != nil {
							return err
						}
						dest.SetTypeAnnotation(v.Type)
						if err := checkIsAssignable(dest, log); err != nil {
							return err
						}
						continue
					}
					dest.SetTypeAnnotation(et)
					v := &Variable{name: dest.IdentifierToken.StringValue, Type: et}
					if err := s.AddElement(v, dest.Location(), log); err != nil {
						return err
					}
					newCount++
				}
				if newCount == 0 {
					return log.AddError(errlog.ErrorNoNewVarsInAssignment, n.Location())
				}
				return nil
			} else if a, ok := ret.Type.(*ArrayType); ok {
				/*
				 * Right-hand side is an array
				 */
				if a.Size >= (1<<32) || len(dests) != int(a.Size) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				newCount := 0
				et := makeExprType(a.ElementType)
				et.Mutable = true
				et.Group = s.Group
				et.PointerDestGroup = ret.PointerDestGroup
				et.PointerDestMutable = ret.PointerDestMutable
				for _, dest := range dests {
					if v := s.lookupVariable(dest.IdentifierToken.StringValue); v != nil {
						if err := checkExprEqualType(v.Type, et, Assignable, dest.Location(), log); err != nil {
							return err
						}
						dest.SetTypeAnnotation(v.Type)
						if err := checkIsAssignable(dest, log); err != nil {
							return err
						}
						continue
					}
					dest.SetTypeAnnotation(et)
					v := &Variable{name: dest.IdentifierToken.StringValue, Type: et}
					if err := s.AddElement(v, dest.Location(), log); err != nil {
						return err
					}
					newCount++
				}
				if newCount == 0 {
					return log.AddError(errlog.ErrorNoNewVarsInAssignment, n.Location())
				}
				return nil
			}
			return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
		}
		/*
		 * Single value on right-hand side
		 */
		newCount := 0
		for i, dest := range dests {
			etRight := exprType(values[i])
			if err := checkInstantiableExprType(etRight, s, values[i].Location(), log); err != nil {
				return err
			}
			if v := s.lookupVariable(dest.IdentifierToken.StringValue); v != nil {
				if err := checkExprEqualType(v.Type, etRight, Assignable, dest.Location(), log); err != nil {
					return err
				}
				dest.SetTypeAnnotation(v.Type)
				if err := checkIsAssignable(dest, log); err != nil {
					return err
				}
				continue
			}
			et := makeExprType(etRight.Type)
			et.Mutable = true
			et.Group = s.Group
			et.PointerDestGroup = etRight.PointerDestGroup
			et.PointerDestMutable = etRight.PointerDestMutable
			dest.SetTypeAnnotation(et)
			v := &Variable{name: dest.IdentifierToken.StringValue, Type: et}
			if err := s.AddElement(v, n.Left.Location(), log); err != nil {
				return err
			}
			newCount++
		}
		if newCount == 0 {
			return log.AddError(errlog.ErrorNoNewVarsInAssignment, n.Location())
		}
	} else {
		if err := checkExpression(n.Left, s, log); err != nil {
			return err
		}
		var dests []parser.Node
		if list, ok := n.Left.(*parser.ExpressionListNode); ok {
			for _, e := range list.Elements {
				dests = append(dests, e.Expression)
			}
		} else {
			dests = []parser.Node{n.Left}
		}
		if len(values) != len(dests) {
			if len(values) != 1 {
				return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
			}
			ret := exprType(n.Right)
			if err := checkInstantiableExprType(ret, s, n.Right.Location(), log); err != nil {
				return err
			}
			if st, ok := ret.Type.(*StructType); ok {
				/*
				 * Right-hand side is a struct
				 */
				// TODO: Only use the accessible fields here, which depends on the package
				if len(dests) != len(st.Fields) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				for i, dest := range dests {
					tleft := exprType(dest)
					tright := makeExprType(st.Fields[i].Type)
					tright.Group = ret.Group
					tright.Mutable = ret.Mutable
					tright.PointerDestGroup = ret.PointerDestGroup
					tright.PointerDestMutable = ret.PointerDestMutable
					if err := checkExprEqualType(tleft, tright, Assignable, n.Location(), log); err != nil {
						return err
					}
					if err := checkIsAssignable(dest, log); err != nil {
						return err
					}
				}
				return nil
			} else if a, ok := ret.Type.(*ArrayType); ok {
				/*
				 * Right-hand side is an array
				 */
				if a.Size >= (1<<32) || len(dests) != int(a.Size) {
					return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
				}
				tright := makeExprType(a.ElementType)
				tright.Group = ret.Group
				tright.Mutable = ret.Mutable
				tright.PointerDestGroup = ret.PointerDestGroup
				tright.PointerDestMutable = ret.PointerDestMutable
				for _, dest := range dests {
					tleft := exprType(dest)
					if err := checkExprEqualType(tleft, tright, Assignable, n.Location(), log); err != nil {
						return err
					}
					if err := checkIsAssignable(dest, log); err != nil {
						return err
					}
				}
				return nil
			}
			return log.AddError(errlog.AssignmentValueCountMismatch, n.Location())
		}
		/*
		 * Single value on right-hand side
		 */
		for i, dest := range dests {
			tleft := exprType(dest)
			tright := exprType(values[i])
			if needsTypeInference(tright) {
				if err := inferType(tright, tleft, values[i].Location(), log); err != nil {
					return err
				}
			} else {
				if err := checkExprEqualType(tleft, tright, Assignable, n.Location(), log); err != nil {
					return err
				}
			}
			if err := checkIsAssignable(dest, log); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkBinaryExpression(n *parser.BinaryExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Left, s, log); err != nil {
		return err
	}
	if err := checkExpression(n.Right, s, log); err != nil {
		return err
	}
	tleft := exprType(n.Left)
	tright := exprType(n.Right)
	switch n.OpToken.Kind {
	case lexer.TokenLogicalOr, lexer.TokenLogicalAnd:
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		if err := expectType(n.Left, boolType, log); err != nil {
			return err
		}
		et := &ExprType{Type: boolType}
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			if n.OpToken.Kind == lexer.TokenLogicalAnd {
				et.BoolValue = tleft.BoolValue && tright.BoolValue
			} else {
				et.BoolValue = tleft.BoolValue || tright.BoolValue
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenEqual, lexer.TokenNotEqual:
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		et := &ExprType{Type: boolType}
		if tleft.HasValue && tright.HasValue {
			if IsIntegerType(tleft.Type) {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenEqual {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) == 0)
				} else {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) != 0)
				}
			} else if IsFloatType(tleft.Type) {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenEqual {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) == 0)
				} else {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) != 0)
				}
			} else if tleft.Type == boolType {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenEqual {
					et.BoolValue = (tleft.BoolValue == tright.BoolValue)
				} else {
					et.BoolValue = (tleft.BoolValue != tright.BoolValue)
				}
			} else if tleft.Type == stringType {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenEqual {
					et.BoolValue = (tleft.StringValue == tright.StringValue)
				} else {
					et.BoolValue = (tleft.StringValue != tright.StringValue)
				}
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenLessOrEqual, lexer.TokenGreaterOrEqual, lexer.TokenLess, lexer.TokenGreater:
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		et := &ExprType{Type: boolType}
		if tleft.HasValue && tright.HasValue {
			if IsIntegerType(tleft.Type) {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenLessOrEqual {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) <= 0)
				} else if n.OpToken.Kind == lexer.TokenGreaterOrEqual {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) >= 0)
				} else if n.OpToken.Kind == lexer.TokenGreater {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) > 0)
				} else {
					et.BoolValue = (tleft.IntegerValue.Cmp(tright.IntegerValue) < 0)
				}
			} else if IsFloatType(tleft.Type) {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenLessOrEqual {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) <= 0)
				} else if n.OpToken.Kind == lexer.TokenGreaterOrEqual {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) >= 0)
				} else if n.OpToken.Kind == lexer.TokenGreater {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) > 0)
				} else {
					et.BoolValue = (tleft.FloatValue.Cmp(tright.FloatValue) < 0)
				}
			} else if tleft.Type == stringType {
				et.HasValue = true
				if n.OpToken.Kind == lexer.TokenLessOrEqual {
					et.BoolValue = (tleft.StringValue <= tright.StringValue)
				} else if n.OpToken.Kind == lexer.TokenGreaterOrEqual {
					et.BoolValue = (tleft.StringValue >= tright.StringValue)
				} else if n.OpToken.Kind == lexer.TokenGreater {
					et.BoolValue = (tleft.StringValue > tright.StringValue)
				} else {
					et.BoolValue = (tleft.StringValue < tright.StringValue)
				}
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenPlus:
	case lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenDivision:
	case lexer.TokenBinaryOr, lexer.TokenAmpersand, lexer.TokenCaret, lexer.TokenPercent, lexer.TokenBitClear:
	case lexer.TokenShiftLeft, lexer.TokenShiftRight:
	}
	panic("Should not happen")
}

func checkUnaryExpression(n *parser.UnaryExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Expression)

	switch n.OpToken.Kind {
	case lexer.TokenBang:
		if err := expectType(n.Expression, boolType, log); err != nil {
			return err
		}
		if et.HasValue {
			n.SetTypeAnnotation(&ExprType{Type: boolType, BoolValue: !et.BoolValue, HasValue: true})
		} else {
			n.SetTypeAnnotation(&ExprType{Type: boolType})
		}
		return nil
	case lexer.TokenCaret:
		if !IsIntegerType(et.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Expression.Location())
		}
		if et.HasValue {
			i := big.NewInt(0)
			i.Not(et.IntegerValue)
			n.SetTypeAnnotation(&ExprType{Type: et.Type, IntegerValue: i, HasValue: true})
		} else {
			n.SetTypeAnnotation(&ExprType{Type: et.Type})
		}
	case lexer.TokenAsterisk:
		panic("TODO")
	case lexer.TokenAmpersand:
		panic("TODO")
	case lexer.TokenMinus:
		if IsSignedIntegerType(et.Type) {
			if et.HasValue {
				i := big.NewInt(0)
				i.Neg(et.IntegerValue)
				n.SetTypeAnnotation(&ExprType{Type: et.Type, IntegerValue: i, HasValue: true})
			} else {
				n.SetTypeAnnotation(&ExprType{Type: et.Type})
			}
			return nil
		} else if IsFloatType(et.Type) {
			if et.HasValue {
				f := big.NewFloat(0)
				f.Neg(et.FloatValue)
				n.SetTypeAnnotation(&ExprType{Type: et.Type, FloatValue: f, HasValue: true})
			} else {
				n.SetTypeAnnotation(&ExprType{Type: et.Type})
			}
			return nil
		}
		return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Expression.Location())
	}
	panic("Should not happen")
}

func checkIdentifierExpression(n *parser.IdentifierExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	loc := n.Location()
	element, err := s.LookupElement(n.IdentifierToken.StringValue, loc, log)
	if err != nil {
		return err
	}
	switch e := element.(type) {
	case *Variable:
		n.SetTypeAnnotation(e.Type)
		return nil
	case *Func:
		n.SetTypeAnnotation(&ExprType{Type: e.Type})
		return nil
	case *GenericFunc:
		return log.AddError(errlog.ErrorGenericMustBeInstantiated, loc)
	case *Namespace:
		return log.AddError(errlog.ErrorNoValueType, loc, n.IdentifierToken.StringValue)
	}
	panic("Should not happen")
}

func checkIsAssignable(n parser.Node, log *errlog.ErrorLog) error {
	et := exprType(n)
	if !et.Mutable {
		return log.AddError(errlog.ErrorNotMutable, n.Location())
	}
	// Ensure that it is not a temporary value
	switch n2 := n.(type) {
	case *parser.IdentifierExpressionNode:
		return nil
	case *parser.ArrayAccessExpressionNode:
		if isSliceExpr(n2.Expression) {
			return nil
		}
		return checkIsAssignable(n2.Expression, log)
	case *parser.MemberAccessExpressionNode:
		if isPointerExpr(n2.Expression) {
			return nil
		}
		return checkIsAssignable(n2.Expression, log)
	}
	return log.AddError(errlog.ErrorTemporaryNotAssignable, n.Location())
}

func isSliceExpr(n parser.Node) bool {
	et := exprType(n)
	return IsSliceType(et.Type)
}

func isPointerExpr(n parser.Node) bool {
	et := exprType(n)
	return IsPointerType(et.Type)
}

func expectType(n parser.Node, mustType Type, log *errlog.ErrorLog) error {
	if isEqualType(n.TypeAnnotation().(*ExprType).Type, mustType, Strict) {
		return nil
	}
	return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
}

func expectTypeMulti(n parser.Node, log *errlog.ErrorLog, mustTypes ...Type) error {
	isType := n.TypeAnnotation().(*ExprType).Type
	for _, mustType := range mustTypes {
		if isEqualType(isType, mustType, Strict) {
			return nil
		}
	}
	return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
}
