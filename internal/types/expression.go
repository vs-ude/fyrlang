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
		return checkMemberAccessExpression(n, s, log)
	case *parser.MemberCallExpressionNode:
		return checkMemberCallExpression(n, s, log)
	case *parser.ArrayAccessExpressionNode:
		return checkArrayAccessExpression(n, s, log)
	case *parser.CastExpressionNode:
		return checkCastExpression(n, s, log)
	case *parser.ConstantExpressionNode:
		return checkConstExpression(n, s, log)
	case *parser.IdentifierExpressionNode:
		return checkIdentifierExpression(n, s, log)
	case *parser.NewExpressionNode:
		return checkNewExpression(n, s, log)
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
		return checkStructLiteralExpression(n, s, log)
	case *parser.ClosureExpressionNode:
		panic("TODO")
	case *parser.MetaAccessNode:
		return checkMetaAccessExpression(n, s, log)
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
		n.SetTypeAnnotation(&ExprType{Type: nullType, IntegerValue: big.NewInt(0), HasValue: true})
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
	if et.ArrayValue == nil {
		// A slice null pointer
		et.IntegerValue = big.NewInt(0)
	}
	n.SetTypeAnnotation(et)
	return nil
}

func checkStructLiteralExpression(n *parser.StructLiteralNode, s *Scope, log *errlog.ErrorLog) error {
	et := &ExprType{Type: structLiteralType, HasValue: true}
	et.StructValue = make(map[string]*ExprType)
	for _, f := range n.Fields {
		if err := checkExpression(f.Value, s, log); err != nil {
			return err
		}
		if _, ok := et.StructValue[f.NameToken.StringValue]; ok {
			return log.AddError(errlog.ErrorLiteralDuplicateField, f.NameToken.Location, f.NameToken.StringValue)
		}
		et.StructValue[f.NameToken.StringValue] = exprType(f.Value)
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
					fet := &ExprType{Type: st.Fields[i].Type, PointerDestGroupSpecifier: vet.PointerDestGroupSpecifier, PointerDestMutable: vet.PointerDestMutable}
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
					fet := &ExprType{Type: a.ElementType, PointerDestGroupSpecifier: vet.PointerDestGroupSpecifier, PointerDestMutable: vet.PointerDestMutable}
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
					if err = inferType(vet, et, false, n.Location(), log); err != nil {
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
					et.PointerDestGroupSpecifier = vet.PointerDestGroupSpecifier
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
					et.PointerDestGroupSpecifier = vet.PointerDestGroupSpecifier
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
			et.PointerDestGroupSpecifier = etRight.PointerDestGroupSpecifier
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
					et.PointerDestGroupSpecifier = ret.PointerDestGroupSpecifier
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
				et.PointerDestGroupSpecifier = ret.PointerDestGroupSpecifier
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
			et.PointerDestGroupSpecifier = etRight.PointerDestGroupSpecifier
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
					tright.GroupSpecifier = ret.GroupSpecifier
					tright.Mutable = ret.Mutable
					tright.PointerDestGroupSpecifier = ret.PointerDestGroupSpecifier
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
				tright.GroupSpecifier = ret.GroupSpecifier
				tright.Mutable = ret.Mutable
				tright.PointerDestGroupSpecifier = ret.PointerDestGroupSpecifier
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
				if err := inferType(tright, tleft, false, values[i].Location(), log); err != nil {
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
		// Pointer arithmetic?
		if IsUnsafePointerType(tleft.Type) {
			if !IsIntegerType(tright.Type) {
				return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
			}
			et := &ExprType{}
			copyExprType(et, tleft)
			n.SetTypeAnnotation(et)
			return nil
		}
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		// TODO: Check for strings
		if !IsIntegerType(tleft.Type) && !IsFloatType(tleft.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
		}
		et := &ExprType{}
		copyExprType(et, tleft)
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			if IsIntegerType(tleft.Type) {
				et.IntegerValue = big.NewInt(0)
				et.IntegerValue.Add(tleft.IntegerValue, tright.IntegerValue)
			} else {
				et.FloatValue = big.NewFloat(0)
				et.FloatValue.Add(tleft.FloatValue, tright.FloatValue)
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenDivision:
		// Pointer arithmetic for Minus?
		if n.OpToken.Kind == lexer.TokenMinus && IsUnsafePointerType(tleft.Type) {
			if !IsIntegerType(tright.Type) {
				return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
			}
			et := &ExprType{}
			copyExprType(et, tleft)
			n.SetTypeAnnotation(et)
			return nil
		}
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		if !IsIntegerType(tleft.Type) && !IsFloatType(tleft.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
		}
		et := &ExprType{}
		copyExprType(et, tleft)
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			if IsIntegerType(tleft.Type) {
				et.IntegerValue = big.NewInt(0)
				if n.OpToken.Kind == lexer.TokenMinus {
					et.IntegerValue.Sub(tleft.IntegerValue, tright.IntegerValue)
				} else if n.OpToken.Kind == lexer.TokenAsterisk {
					et.IntegerValue.Mul(tleft.IntegerValue, tright.IntegerValue)
				} else if n.OpToken.Kind == lexer.TokenDivision {
					et.IntegerValue.Div(tleft.IntegerValue, tright.IntegerValue)
				} else {
					panic("ooops")
				}
			} else {
				et.FloatValue = big.NewFloat(0)
				if n.OpToken.Kind == lexer.TokenMinus {
					et.FloatValue.Sub(tleft.FloatValue, tright.FloatValue)
				} else if n.OpToken.Kind == lexer.TokenAsterisk {
					et.FloatValue.Mul(tleft.FloatValue, tright.FloatValue)
				} else if n.OpToken.Kind == lexer.TokenDivision {
					et.FloatValue.Quo(tleft.FloatValue, tright.FloatValue)
				} else {
					panic("ooops")
				}
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenBinaryOr, lexer.TokenAmpersand, lexer.TokenCaret, lexer.TokenBitClear:
		if IsUnsafePointerType(tleft.Type) {
			if err := checkExprEqualType(&ExprType{Type: PrimitiveTypeUintptr}, tright, Comparable, n.Location(), log); err != nil {
				return err
			}
		} else {
			if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
				return err
			}
			if !IsIntegerType(tleft.Type) {
				return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
			}
		}
		et := &ExprType{}
		copyExprType(et, tleft)
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			et.IntegerValue = big.NewInt(0)
			if n.OpToken.Kind == lexer.TokenBinaryOr {
				et.IntegerValue.Or(tleft.IntegerValue, tright.IntegerValue)
			} else if n.OpToken.Kind == lexer.TokenAmpersand {
				et.IntegerValue.And(tleft.IntegerValue, tright.IntegerValue)
			} else if n.OpToken.Kind == lexer.TokenCaret {
				et.IntegerValue.Xor(tleft.IntegerValue, tright.IntegerValue)
			} else if n.OpToken.Kind == lexer.TokenPercent {
				et.IntegerValue.Rem(tleft.IntegerValue, tright.IntegerValue)
			} else if n.OpToken.Kind == lexer.TokenBitClear {
				et.IntegerValue.AndNot(tleft.IntegerValue, tright.IntegerValue)
			} else {
				panic("ooops")
			}
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenPercent:
		if err := checkExprEqualType(tleft, tright, Comparable, n.Location(), log); err != nil {
			return err
		}
		if !IsIntegerType(tleft.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Location())
		}
		et := &ExprType{}
		copyExprType(et, tleft)
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			et.IntegerValue = big.NewInt(0)
			et.IntegerValue.Rem(tleft.IntegerValue, tright.IntegerValue)
		}
		n.SetTypeAnnotation(et)
		return nil
	case lexer.TokenShiftLeft, lexer.TokenShiftRight:
		if !IsIntegerType(tleft.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Left.Location())
		}
		if !IsUnsignedIntegerType(tright.Type) {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Right.Location())
		}
		if tright.HasValue && !tright.IntegerValue.IsUint64() {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Right.Location())
		}
		et := &ExprType{}
		copyExprType(et, tleft)
		if tleft.HasValue && tright.HasValue {
			et.HasValue = true
			et.IntegerValue = big.NewInt(0)
			if n.OpToken.Kind == lexer.TokenShiftLeft {
				et.IntegerValue.Lsh(tleft.IntegerValue, uint(tright.IntegerValue.Uint64()))
			} else if n.OpToken.Kind == lexer.TokenShiftRight {
				et.IntegerValue.Rsh(tleft.IntegerValue, uint(tright.IntegerValue.Uint64()))
			} else {
				panic("ooops")
			}
		}
		n.SetTypeAnnotation(et)
		return nil
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
		return nil
	case lexer.TokenAsterisk:
		if et.HasValue && et.IntegerValue != nil && et.IntegerValue.Uint64() == 0 {
			return log.AddError(errlog.ErrorDereferencingNullPointer, n.Location())
		}
		pt, ok := GetPointerType(et.Type)
		if !ok {
			return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Expression.Location())
		}
		n.SetTypeAnnotation(derivePointerExprType(et, pt.ElementType))
		return nil
	case lexer.TokenAmpersand:
		if err := checkIsAddressable(n.Expression, log); err != nil {
			return err
		}
		n.SetTypeAnnotation(deriveAddressOfExprType(et, n.OpToken.Location))
		return nil
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
	case lexer.TokenEllipsis:
		return log.AddError(errlog.ErrorIllegalEllipsis, n.OpToken.Location)
	}
	fmt.Printf("%v\n", n.OpToken.Raw)
	panic("Should not happen")
}

func checkNewExpression(n *parser.NewExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	t, err := parseType(n.Type, s, log)
	if err != nil {
		return err
	}
	et := makeExprType(t)
	n.Type.SetTypeAnnotation(et)
	if _, ok := GetStructType(t); ok {
		pt := &PointerType{TypeBase: TypeBase{location: n.Location()}, Mode: PtrOwner, ElementType: t}
		et2 := makeExprType(pt)
		n.SetTypeAnnotation(et2)
		if _, ok := n.Value.(*parser.StructLiteralNode); ok {
			if err := checkExpression(n.Value, s, log); err != nil {
				return err
			}
			if err := checkExprEqualType(et, exprType(n.Value), Assignable, n.Value.Location(), log); err != nil {
				return err
			}
		} else if n.Value == nil {
			// Do nothing
		} else {
			return log.AddError(errlog.ErrorNewInitializerMismatch, n.Location())
		}
	} else if _, ok := GetSliceType(t); ok {
		n.SetTypeAnnotation(et)
		if pe, ok := n.Value.(*parser.ParanthesisExpressionNode); ok {
			// "new []int(0, 100)" or "new []int(100)"
			if el, ok := pe.Expression.(*parser.ExpressionListNode); ok {
				if len(el.Elements) != 2 {
					return log.AddError(errlog.ErrorNewInitializerMismatch, n.Location())
				}
				if err := checkExpression(el.Elements[0].Expression, s, log); err != nil {
					return err
				}
				pet := exprType(el.Elements[0].Expression)
				if err := checkExprIntType(pet, el.Elements[0].Expression.Location(), log); err != nil {
					return err
				}
				if err := checkExpression(el.Elements[1].Expression, s, log); err != nil {
					return err
				}
				pet = exprType(el.Elements[1].Expression)
				if err := checkExprIntType(pet, el.Elements[1].Expression.Location(), log); err != nil {
					return err
				}
			} else {
				if err := checkExpression(pe.Expression, s, log); err != nil {
					return err
				}
				pet := exprType(pe.Expression)
				if err := checkExprIntType(pet, pe.Expression.Location(), log); err != nil {
					return err
				}
			}
		} else if _, ok := n.Value.(*parser.ArrayLiteralNode); ok {
			// "new []int[1, 2, 3]"
			if err := checkExpression(n.Value, s, log); err != nil {
				return err
			}
			if err := checkExprEqualType(et, exprType(n.Value), Assignable, n.Value.Location(), log); err != nil {
				return err
			}
		} else if n.Value == nil {
			// New default value, i.e. "new []int", which is a null slice
			// Do nothing
		} else {
			return log.AddError(errlog.ErrorNewInitializerMismatch, n.Location())
		}
	} else if _, ok := GetArrayType(t); ok {
		pt := &PointerType{TypeBase: TypeBase{location: n.Location()}, Mode: PtrOwner, ElementType: t}
		et2 := makeExprType(pt)
		n.SetTypeAnnotation(et2)
		if _, ok := n.Value.(*parser.ArrayLiteralNode); ok {
			if err := checkExpression(n.Value, s, log); err != nil {
				return err
			}
			if err := checkExprEqualType(et, exprType(n.Value), Assignable, n.Value.Location(), log); err != nil {
				return err
			}
		} else if n.Value == nil {
			// Do nothing
		} else {
			return log.AddError(errlog.ErrorNewInitializerMismatch, n.Location())
		}
	} else {
		pt := &PointerType{TypeBase: TypeBase{location: n.Location()}, Mode: PtrOwner, ElementType: t}
		et2 := makeExprType(pt)
		n.SetTypeAnnotation(et2)
		if pe, ok := n.Value.(*parser.ParanthesisExpressionNode); ok {
			if err := checkExpression(pe.Expression, s, log); err != nil {
				return err
			}
			if err := checkExprEqualType(et, exprType(pe.Expression), Assignable, pe.Expression.Location(), log); err != nil {
				return err
			}
		} else if n.Value == nil {
			// Do nothing
		} else {
			return log.AddError(errlog.ErrorNewInitializerMismatch, n.Location())
		}
	}
	return nil
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
		n.SetTypeAnnotation(&ExprType{Type: e.Type, HasValue: true, FuncValue: e})
		return nil
	case *GenericFunc:
		return log.AddError(errlog.ErrorGenericMustBeInstantiated, loc)
	case *Namespace:
		n.SetTypeAnnotation(&ExprType{Type: namespaceType, HasValue: true, NamespaceValue: e})
		return nil
	}
	panic("Should not happen")
}

func checkArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Expression)
	if n.ColonToken == nil {
		// Expression of the kind `arr[idx]`
		if err := checkExpression(n.Index, s, log); err != nil {
			return err
		}
		iet := exprType(n.Index)
		if err := checkExprIntType(iet, n.Index.Location(), log); err != nil {
			return err
		}
		if a, ok := GetArrayType(et.Type); ok {
			if iet.HasValue {
				i := iet.IntegerValue.Int64()
				if i < 0 || i >= int64(a.Size) {
					return log.AddError(errlog.ErrorNumberOutOfRange, n.Index.Location(), iet.IntegerValue.Text(10))
				}
			}
			n.SetTypeAnnotation(deriveExprType(et, a.ElementType))
			return nil
		} else if s, ok := GetSliceType(et.Type); ok {
			n.SetTypeAnnotation(derivePointerExprType(et, s.ElementType))
			return nil
		}
	} else {
		// Expression of the kind `arr[left:right]`
		var iet1 *ExprType
		var iet2 *ExprType
		if n.Index != nil {
			if err := checkExpression(n.Index, s, log); err != nil {
				return err
			}
			iet1 = exprType(n.Index)
			if err := checkExprIntType(iet1, n.Index.Location(), log); err != nil {
				return err
			}
		}
		if n.Index2 != nil {
			if err := checkExpression(n.Index2, s, log); err != nil {
				return err
			}
			iet2 = exprType(n.Index2)
			if err := checkExprIntType(iet2, n.Index2.Location(), log); err != nil {
				return err
			}
		}
		if a, ok := GetArrayType(et.Type); ok {
			if iet1 != nil && iet1.HasValue {
				i := iet1.IntegerValue.Int64()
				if i < 0 || i >= int64(a.Size) {
					return log.AddError(errlog.ErrorNumberOutOfRange, n.Index.Location(), iet1.IntegerValue.Text(10))
				}
			}
			if iet2 != nil && iet2.HasValue {
				i := iet2.IntegerValue.Int64()
				if i < 0 || i >= int64(a.Size) {
					return log.AddError(errlog.ErrorNumberOutOfRange, n.Index2.Location(), iet2.IntegerValue.Text(10))
				}
			}
			n.SetTypeAnnotation(deriveSliceOfExprType(et, a.ElementType, n.Location()))
			return nil
		} else if IsSliceType(et.Type) {
			n.SetTypeAnnotation(et)
			return nil
		}
	}
	return log.AddError(errlog.ErrorIncompatibleTypeForOp, n.Expression.Location())
}

func checkMemberAccessExpression(n *parser.MemberAccessExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Expression)
	if et.Type == namespaceType {
		element, err := et.NamespaceValue.Scope.LookupElement(n.IdentifierToken.StringValue, n.Location(), log)
		if err != nil {
			return err
		}
		switch e := element.(type) {
		case *Variable:
			n.SetTypeAnnotation(e.Type)
			return nil
		case *Func:
			n.SetTypeAnnotation(&ExprType{Type: e.Type, HasValue: true, FuncValue: e})
			return nil
		case *GenericFunc:
			return log.AddError(errlog.ErrorGenericMustBeInstantiated, n.Location())
		case *Namespace:
			n.SetTypeAnnotation(&ExprType{Type: namespaceType, HasValue: true, NamespaceValue: e})
			return nil
		}
		panic("Oooops")
	}
	if pt, ok := GetPointerType(et.Type); ok {
		et = derivePointerExprType(et, pt.ElementType)
	}
	found := false
	if gt, ok := GetGenericInstanceType(et.Type); ok {
		if fun := gt.Func(n.IdentifierToken.StringValue); fun != nil {
			et = makeExprType(fun.Type)
			et.HasValue = true
			et.FuncValue = fun
			found = true
		}
	}
	if at, ok := GetAliasType(et.Type); !found && ok {
		if fun := at.Func(n.IdentifierToken.StringValue); fun != nil {
			et = makeExprType(fun.Type)
			et.HasValue = true
			et.FuncValue = fun
			found = true
		}
	}
	if st, ok := GetStructType(et.Type); !found && ok {
		if f := st.Field(n.IdentifierToken.StringValue); f != nil {
			et = deriveExprType(et, f.Type)
			found = true
		} else if fun := st.Func(n.IdentifierToken.StringValue); fun != nil {
			et = makeExprType(fun.Type)
			et.HasValue = true
			et.FuncValue = fun
			found = true
		}
	}
	if !found {
		return log.AddError(errlog.ErrorUnknownField, n.IdentifierToken.Location, n.IdentifierToken.StringValue)
	}
	n.SetTypeAnnotation(et)
	return nil
}

func checkMemberCallExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	// Bultin-functions?
	if ident, ok := n.Expression.(*parser.IdentifierExpressionNode); ok {
		if ident.IdentifierToken.StringValue == "len" {
			return checkLenExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "cap" {
			return checkCapExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "append" {
			return checkAppendExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "groupOf" {
			return checkGroupOfExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "panic" {
			return checkPanicExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "println" {
			return checkPrintlnExpression(n, s, log)
		} else if ident.IdentifierToken.StringValue == "take" {
			return checkTakeExpression(n, s, log)
		}
	}
	// TODO: If the expression is a MemberAccessExpression, see whether the member is a function of this type
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Expression)
	ft, ok := GetFuncType(et.Type)
	if !ok {
		return log.AddError(errlog.ErrorNotAFunction, n.Expression.Location())
	}
	// A member function?
	if ft.Target != nil {
		if !et.HasValue || et.FuncValue == nil {
			panic("Ooooops")
		}
		// The target of the member function is a pointer type?
		if _, ok := GetPointerType(ft.Target); ok {
			thisEt := exprType(n.Expression.(*parser.MemberAccessExpressionNode).Expression)
			targetEt := makeExprType(ft.Target)
			// The function requires a mutable target, but the target is not mutable?
			if !thisEt.PointerDestMutable && targetEt.PointerDestMutable {
				if et.FuncValue.DualFunc != nil {
					// Use the dual func
					et.FuncValue = et.FuncValue.DualFunc
				} else {
					return log.AddError(errlog.ErrorTargetIsNotMutable, n.Location())
				}
			}
		}
	}

	if len(ft.In.Params) == len(n.Arguments.Elements) {
		for i, pe := range n.Arguments.Elements {
			if err := checkExpression(pe.Expression, s, log); err != nil {
				return err
			}
			pet := exprType(pe.Expression)
			if err := checkExprEqualType(makeExprType(ft.In.Params[i].Type), pet, Assignable, pe.Location(), log); err != nil {
				return err
			}
		}
	} else {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	et = makeExprType(ft.ReturnType())
	n.SetTypeAnnotation(et)
	return nil
}

func checkTakeExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	if !et.Mutable {
		return log.AddError(errlog.ErrorTargetIsNotMutable, n.Location())
	}
	if _, ok := GetSliceType(et.Type); ok {
		n.SetTypeAnnotation(et)
	} else if _, ok := GetPointerType(et.Type); ok {
		n.SetTypeAnnotation(et)
	} else if et.Type == PrimitiveTypeString {
		n.SetTypeAnnotation(et)
	} else {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	return nil
}

func checkLenExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	etResult := makeExprType(PrimitiveTypeInt)
	if _, ok := GetSliceType(et.Type); ok {
		// Do nothing by intention
		n.SetTypeAnnotation(etResult)
	} else if arr, ok := GetArrayType(et.Type); ok {
		etResult.HasValue = true
		etResult.IntegerValue = big.NewInt(0)
		etResult.IntegerValue.SetUint64(arr.Size)
		n.SetTypeAnnotation(etResult)
	} else if et.Type == PrimitiveTypeString {
		if et.HasValue {
			etResult.HasValue = true
			etResult.IntegerValue = big.NewInt(int64(len(et.StringValue)))
		}
		n.SetTypeAnnotation(etResult)
	} else {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	return nil
}

func checkCapExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	etResult := makeExprType(PrimitiveTypeInt)
	if _, ok := GetSliceType(et.Type); ok {
		// Do nothing by intention
		n.SetTypeAnnotation(etResult)
	} else {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	return nil
}

func checkAppendExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) <= 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	sl, ok := GetSliceType(et.Type)
	if !ok {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	if !et.PointerDestMutable {
		return log.AddError(errlog.ErrorNotMutable, n.Arguments.Elements[0].Expression.Location())
	}
	targetEt := derivePointerExprType(et, sl.ElementType)
	for _, el := range n.Arguments.Elements[1:] {
		if unary, ok := el.Expression.(*parser.UnaryExpressionNode); ok && unary.OpToken.Kind == lexer.TokenEllipsis {
			if err := checkExpression(unary.Expression, s, log); err != nil {
				return err
			}
			argEt := exprType(unary.Expression)
			if sl, ok := GetSliceType(argEt.Type); ok {
				valueEt := derivePointerExprType(argEt, sl.ElementType)
				if err := checkExprEqualType(targetEt, valueEt, Assignable, unary.Expression.Location(), log); err != nil {
					return err
				}
			} else if ar, ok := GetArrayType(argEt.Type); ok {
				valueEt := deriveExprType(argEt, ar.ElementType)
				if err := checkExprEqualType(targetEt, valueEt, Assignable, unary.Expression.Location(), log); err != nil {
					return err
				}
			} else {
				return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
			}
		} else {
			if err := checkExpression(el.Expression, s, log); err != nil {
				return err
			}
			argEt := exprType(el.Expression)
			if err := checkExprEqualType(targetEt, argEt, Assignable, el.Expression.Location(), log); err != nil {
				return err
			}
		}
	}
	n.SetTypeAnnotation(et)
	return nil
}

func checkGroupOfExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	etResult := makeExprType(PrimitiveTypeUintptr)
	n.SetTypeAnnotation(etResult)
	return nil
}

func checkPanicExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	if err := checkExprStringType(et, n.Location(), log); err != nil {
		return err
	}
	etResult := makeExprType(PrimitiveTypeVoid)
	n.SetTypeAnnotation(etResult)
	if et.Type != PrimitiveTypeString {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	return nil
}

func checkPrintlnExpression(n *parser.MemberCallExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if len(n.Arguments.Elements) != 1 {
		return log.AddError(errlog.ErrorParamterCountMismatch, n.Arguments.Location())
	}
	if err := checkExpression(n.Arguments.Elements[0].Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Arguments.Elements[0].Expression)
	if err := checkExprStringType(et, n.Location(), log); err != nil {
		return err
	}
	etResult := makeExprType(PrimitiveTypeVoid)
	n.SetTypeAnnotation(etResult)
	if et.Type != PrimitiveTypeString {
		return log.AddError(errlog.ErrorIncompatibleTypes, n.Arguments.Elements[0].Expression.Location())
	}
	return nil
}

func checkCastExpression(n *parser.CastExpressionNode, s *Scope, log *errlog.ErrorLog) error {
	if err := checkExpression(n.Expression, s, log); err != nil {
		return err
	}
	et := exprType(n.Expression)
	t, err := parseType(n.Type, s, log)
	if err != nil {
		return err
	}
	etResult := makeExprType(t)
	etResult.TypeConversionValue = ConvertIllegal
	if ptResult, ok := GetPointerType(etResult.Type); ok {
		if ptResult.Mode == PtrUnsafe && ptResult.ElementType == PrimitiveTypeByte && et.Type == PrimitiveTypeString {
			// String -> #byte
			etResult.TypeConversionValue = ConvertStringToPointer
		} else if _, ok := GetPointerType(et.Type); ok && ptResult.Mode == PtrUnsafe {
			// #X -> #Y or *X -> #Y
			etResult.TypeConversionValue = ConvertPointerToPointer
		} else if sl, ok := GetSliceType(et.Type); ok && ptResult.Mode == PtrUnsafe && sl.ElementType == ptResult.ElementType {
			// []X -> #X
			etResult.TypeConversionValue = ConvertSliceToPointer
		} else if IsUnsignedIntegerType(et.Type) {
			// int -> #X
			etResult.TypeConversionValue = ConvertIntegerToPointer
		}
	} else if slResult, ok := GetSliceType(etResult.Type); ok {
		if pt, ok := GetPointerType(et.Type); ok && pt.Mode == PtrUnsafe && slResult.ElementType == pt.ElementType {
			// #X -> []X
			etResult.TypeConversionValue = ConvertPointerToSlice
		} else if slResult.ElementType == PrimitiveTypeByte && !etResult.PointerDestMutable && etResult.PointerDestGroupSpecifier == nil && et.Type == PrimitiveTypeString {
			// String -> []byte
			etResult.TypeConversionValue = ConvertStringToByteSlice
		}
	} else if etResult.Type == PrimitiveTypeString {
		if pt, ok := GetPointerType(et.Type); ok && pt.Mode == PtrUnsafe && pt.ElementType == PrimitiveTypeByte {
			// #byte -> String
			etResult.TypeConversionValue = ConvertPointerToString
		} else if sl, ok := GetSliceType(et.Type); ok && sl.ElementType == PrimitiveTypeByte {
			// []byte -> String
			etResult.TypeConversionValue = ConvertByteSliceToString
		}
	} else if IsIntegerType(etResult.Type) || etResult.Type == PrimitiveTypeByte {
		if IsIntegerType(et.Type) || et.Type == PrimitiveTypeByte {
			// Integer -> Integer
			etResult.TypeConversionValue = ConvertIntegerToInteger
		} else if IsFloatType(et.Type) {
			// Float -> Integer
			etResult.TypeConversionValue = ConvertFloatToInteger
		} else if et.Type == PrimitiveTypeBool {
			// Bool -> Integer
			etResult.TypeConversionValue = ConvertBoolToInteger
		} else if et.Type == PrimitiveTypeRune {
			// Rune -> Integer
			etResult.TypeConversionValue = ConvertRuneToInteger
		} else if etResult.Type == PrimitiveTypeUintptr && IsUnsafePointerType(et.Type) {
			// #X -> Integer
			etResult.TypeConversionValue = ConvertPointerToInteger
		}
	} else if IsFloatType(etResult.Type) {
		if IsIntegerType(et.Type) || et.Type == PrimitiveTypeByte {
			// Integer -> Float
			etResult.TypeConversionValue = ConverIntegerToFloat
		} else if IsFloatType(et.Type) {
			// Float -> Float
			etResult.TypeConversionValue = ConvertFloatToFloat
		}
	} else if etResult.Type == PrimitiveTypeBool {
		if IsIntegerType(et.Type) || et.Type == PrimitiveTypeByte {
			// Integer -> Bool
			etResult.TypeConversionValue = ConvertIntegerToBool
		}
	} else if etResult.Type == PrimitiveTypeRune {
		if IsIntegerType(et.Type) || et.Type == PrimitiveTypeByte {
			// Integer -> Rune
			etResult.TypeConversionValue = ConvertIntegerToRune
		}
	}
	if etResult.TypeConversionValue == ConvertIllegal {
		return log.AddError(errlog.ErrorIllegalCast, n.Location(), et.Type.ToString(), etResult.Type.ToString())
	}
	n.SetTypeAnnotation(etResult)
	return nil
}

func checkMetaAccessExpression(n *parser.MetaAccessNode, s *Scope, log *errlog.ErrorLog) error {
	t, err := parseType(n.Type, s, log)
	if err != nil {
		return err
	}
	n.Type.SetTypeAnnotation(makeExprType(t))
	if n.IdentifierToken.StringValue == "size" {
		n.SetTypeAnnotation(makeExprType(PrimitiveTypeInt))
	} else {
		return log.AddError(errlog.ErrorUnknownMetaProperty, n.IdentifierToken.Location, n.IdentifierToken.StringValue)
	}
	return nil
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
	case *parser.UnaryExpressionNode:
		if n2.OpToken.Kind == lexer.TokenAsterisk {
			return nil
		}
	}
	return log.AddError(errlog.ErrorTemporaryNotAssignable, n.Location())
}

func checkIsAddressable(n parser.Node, log *errlog.ErrorLog) error {
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
	return log.AddError(errlog.ErrorTemporaryNotAddressable, n.Location())
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
