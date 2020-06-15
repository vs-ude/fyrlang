package irgen

import (
	"fmt"
	"math/big"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/lexer"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

// var builtinFunctionNames = []string{"len", "cap", "append", "panic", "groupOf"}

func genExpression(ast parser.Node, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	b.SetLocation(ast.Location())
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			genExpression(e.Expression, s, b, p, vars)
		}
		return ircode.Argument{}
	case *parser.BinaryExpressionNode:
		return genBinaryExpression(n, s, b, p, vars)
	case *parser.UnaryExpressionNode:
		return genUnaryExpression(n, s, b, p, vars)
	case *parser.IsTypeExpressionNode:
		panic("TODO")
	case *parser.MemberAccessExpressionNode:
		return genMemberAccessExpression(n, s, b, p, vars)
	case *parser.MemberCallExpressionNode:
		return genCallExpression(n, s, b, p, vars)
	case *parser.ArrayAccessExpressionNode:
		return genArrayAccessExpression(n, s, b, p, vars)
	case *parser.CastExpressionNode:
		return genCastExpression(n, s, b, p, vars)
	case *parser.ConstantExpressionNode:
		return genConstantExpression(n, s, b, p, vars)
	case *parser.IdentifierExpressionNode:
		return genIdentifierExpression(n, s, b, p, vars)
	case *parser.NewExpressionNode:
		return genNewExpression(n, s, b, p, vars)
	case *parser.ParanthesisExpressionNode:
		return genExpression(n.Expression, s, b, p, vars)
	case *parser.AssignmentExpressionNode:
		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
			return genAssignmentExpression(n, s, b, p, vars)
		}
		return genAssignmentOpExpression(n, s, b, p, vars)
	case *parser.IncrementExpressionNode:
		return genIncrementExpression(n, s, b, p, vars)
	case *parser.VarExpressionNode:
		return genVarExpression(n, s, b, p, vars)
	case *parser.ArrayLiteralNode:
		return genArrayLiteralExpression(n, s, b, p, vars)
	case *parser.StructLiteralNode:
		return genStructLiteralExpression(n, s, b, p, vars)
	case *parser.ClosureExpressionNode:
		panic("TODO")
	case *parser.MetaAccessNode:
		return genMetaAccessExpression(n, s, b, p, vars)
	}
	fmt.Printf("%T\n", ast)
	// panic("Should not happen")
	// HACK
	return ircode.NewIntArg(0)
}

func genIdentifierExpression(n *parser.IdentifierExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	element := s.GetElement(n.IdentifierToken.StringValue)
	if element == nil {
		panic("Oooops")
	}
	switch e := element.(type) {
	case *types.Variable:
		v, ok := vars[e]
		if !ok {
			v = b.DefineVariable(e.Name(), e.Type)
			vars[e] = v
		}
		return ircode.NewVarArg(v)
	case *types.Func:
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	case *types.Namespace:
		// Generate no code
		return ircode.Argument{}
	}
	panic("Should not happen")
}

func genArrayLiteralExpression(n *parser.ArrayLiteralNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.IsConstant() {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	var values []ircode.Argument
	for _, v := range n.Values.Elements {
		values = append(values, genExpression(v.Expression, s, b, p, vars))
	}
	return ircode.NewVarArg(b.Array(nil, exprType(n), values))
}

func genStructLiteralExpression(n *parser.StructLiteralNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.IsConstant() {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	t := et.Type
	if ptr, ok := types.GetPointerType(t); ok {
		t = ptr.ElementType
	}
	st, ok := types.GetStructType(t)
	if !ok {
		panic("Oooops")
	}
	fields := make(map[string]ircode.Argument)
	for _, f := range n.Fields {
		fields[f.NameToken.StringValue] = genExpression(f.Value, s, b, p, vars)
	}
	var values []ircode.Argument
	if st.BaseType != nil {
		if arg, ok := fields[st.BaseType.Name()]; ok {
			values = append(values, arg)
		} else {
			values = append(values, genDefaultValue(st.BaseType))
		}
	}
	for _, f := range st.Fields {
		if arg, ok := fields[f.Name]; ok {
			values = append(values, arg)
		} else {
			values = append(values, genDefaultValue(f.Type))
		}
	}
	return ircode.NewVarArg(b.Struct(nil, exprType(n), values))
}

func genConstantExpression(n *parser.ConstantExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
}

func genBinaryExpression(n *parser.BinaryExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.IsConstant() {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	left := genExpression(n.Left, s, b, p, vars)
	right := genExpression(n.Right, s, b, p, vars)
	tleft := exprType(n.Left)
	switch n.OpToken.Kind {
	case lexer.TokenLogicalOr:
		return ircode.NewVarArg(b.BooleanOp(ircode.OpLogicalOr, nil, left, right))
	case lexer.TokenLogicalAnd:
		return ircode.NewVarArg(b.BooleanOp(ircode.OpLogicalAnd, nil, left, right))
	case lexer.TokenEqual:
		if tleft.Type == types.PrimitiveTypeBool || types.IsIntegerType(tleft.Type) || types.IsFloatType(tleft.Type) || types.IsPointerType(tleft.Type) || types.IsSliceType(tleft.Type) || types.IsFuncType(tleft.Type) {
			return ircode.NewVarArg(b.Compare(ircode.OpEqual, nil, left, right))
		}
		panic("TODO")
	case lexer.TokenNotEqual:
		return ircode.NewVarArg(b.Compare(ircode.OpNotEqual, nil, left, right))
	case lexer.TokenLessOrEqual:
		return ircode.NewVarArg(b.Compare(ircode.OpLessOrEqual, nil, left, right))
	case lexer.TokenGreaterOrEqual:
		return ircode.NewVarArg(b.Compare(ircode.OpGreaterOrEqual, nil, left, right))
	case lexer.TokenLess:
		return ircode.NewVarArg(b.Compare(ircode.OpLess, nil, left, right))
	case lexer.TokenGreater:
		return ircode.NewVarArg(b.Compare(ircode.OpGreater, nil, left, right))
	case lexer.TokenPlus:
		if types.IsStringType(et.Type) {
			return ircode.NewVarArg(b.StringConcat(nil, et, left, right))
		}
		return ircode.NewVarArg(b.Add(nil, left, right))
	case lexer.TokenMinus:
		return ircode.NewVarArg(b.Sub(nil, left, right))
	case lexer.TokenAsterisk:
		return ircode.NewVarArg(b.Mul(nil, left, right))
	case lexer.TokenDivision:
		return ircode.NewVarArg(b.Div(nil, left, right))
	case lexer.TokenBinaryOr:
		return ircode.NewVarArg(b.BinaryOr(nil, left, right))
	case lexer.TokenAmpersand:
		return ircode.NewVarArg(b.BinaryAnd(nil, left, right))
	case lexer.TokenCaret:
		return ircode.NewVarArg(b.BinaryXor(nil, left, right))
	case lexer.TokenPercent:
		return ircode.NewVarArg(b.Remainder(nil, left, right))
	case lexer.TokenBitClear:
		return ircode.NewVarArg(b.BitClear(nil, left, right))
	case lexer.TokenShiftLeft:
		return ircode.NewVarArg(b.ShiftLeft(nil, left, right))
	case lexer.TokenShiftRight:
		return ircode.NewVarArg(b.ShiftRight(nil, left, right))
	}
	panic("Should not happen")
}

func genUnaryExpression(n *parser.UnaryExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.HasValue {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	switch n.OpToken.Kind {
	case lexer.TokenBang:
		expr := genExpression(n.Expression, s, b, p, vars)
		return ircode.NewVarArg(b.BooleanNot(nil, expr))
	case lexer.TokenCaret:
		expr := genExpression(n.Expression, s, b, p, vars)
		return ircode.NewVarArg(b.BitwiseComplement(nil, expr))
	case lexer.TokenAsterisk:
		ab := genGetAccessChain(n, s, b, p, vars)
		return ircode.NewVarArg(ab.GetValue())
	case lexer.TokenAmpersand:
		ab := genGetAccessChain(n, s, b, p, vars)
		return ircode.NewVarArg(ab.GetValue())
	case lexer.TokenMinus:
		expr := genExpression(n.Expression, s, b, p, vars)
		return ircode.NewVarArg(b.MinusSign(nil, expr))
	}
	panic("Should not happen")
}

func genVarExpression(n *parser.VarExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	if n.Value == nil {
		return ircode.Argument{}
	}
	var valueNodes []parser.Node
	if list, ok := n.Value.(*parser.ExpressionListNode); ok {
		for _, el := range list.Elements {
			valueNodes = append(valueNodes, el.Expression)
		}
	} else {
		valueNodes = []parser.Node{n.Value}
	}
	if len(valueNodes) != len(n.Names) {
		value := genExpression(valueNodes[0], s, b, p, vars)
		et := exprType(n.Value)
		for i, name := range n.Names {
			e := s.GetVariable(name.NameToken.StringValue)
			if e == nil {
				panic("Oooops")
			}
			ab := b.Get(nil, value)
			if types.IsArrayType(et.Type) {
				ab = ab.ArrayIndex(ircode.NewIntArg(i), e.Type)
			} else if types.IsSliceType(et.Type) {
				ab = ab.SliceIndex(ircode.NewIntArg(i), e.Type)
			} else if st, ok := types.GetStructType(et.Type); ok {
				ab = ab.StructField(st.Fields[i], e.Type)
			} else if pt, ok := types.GetPointerType(et.Type); ok {
				st, _ := types.GetStructType(pt.ElementType)
				ab = ab.PointerStructField(st.Fields[i], e.Type)
			}
			singleValue := ab.GetValue()
			v, ok := vars[e]
			if !ok {
				v = b.DefineVariable(e.Name(), e.Type)
				vars[e] = v
			}
			b.SetVariable(v, ircode.NewVarArg(singleValue))
		}
		return ircode.Argument{}
	}
	for i, name := range n.Names {
		value := genExpression(valueNodes[i], s, b, p, vars)
		e := s.GetVariable(name.NameToken.StringValue)
		if e == nil {
			panic("Oooops")
		}
		v, ok := vars[e]
		if !ok {
			v = b.DefineVariable(e.Name(), e.Type)
			vars[e] = v
		}
		b.SetVariable(v, value)
	}
	return ircode.Argument{}
}

func genAssignmentExpression(n *parser.AssignmentExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	var valueNodes []parser.Node
	if list, ok := n.Right.(*parser.ExpressionListNode); ok {
		for _, el := range list.Elements {
			valueNodes = append(valueNodes, el.Expression)
		}
	} else {
		valueNodes = []parser.Node{n.Right}
	}
	var destNodes []parser.Node
	if list, ok := n.Left.(*parser.ExpressionListNode); ok {
		for _, el := range list.Elements {
			destNodes = append(destNodes, el.Expression)
		}
	} else {
		destNodes = []parser.Node{n.Left}
	}
	if n.OpToken.Kind == lexer.TokenWalrus {
		if len(valueNodes) != len(destNodes) {
			value := genExpression(valueNodes[0], s, b, p, vars)
			et := exprType(n.Right)
			for i, destNode := range destNodes {
				ident, ok := destNode.(*parser.IdentifierExpressionNode)
				if !ok {
					panic("Oooops")
				}
				e := s.GetVariable(ident.IdentifierToken.StringValue)
				if e == nil {
					panic("Oooops")
				}
				var singleValue ircode.Argument
				if value.Array != nil {
					singleValue = value.Array[i]
				} else {
					ab := b.Get(nil, value)
					if types.IsArrayType(et.Type) {
						ab = ab.ArrayIndex(ircode.NewIntArg(i), e.Type)
					} else if types.IsSliceType(et.Type) {
						ab = ab.SliceIndex(ircode.NewIntArg(i), e.Type)
					} else if st, ok := types.GetStructType(et.Type); ok {
						ab = ab.StructField(st.Fields[i], e.Type)
					} else if pt, ok := types.GetPointerType(et.Type); ok {
						st, _ := types.GetStructType(pt.ElementType)
						ab = ab.PointerStructField(st.Fields[i], e.Type)
					}
					singleValue = ircode.NewVarArg(ab.GetValue())
				}
				v, ok := vars[e]
				if !ok {
					v = b.DefineVariable(e.Name(), e.Type)
					vars[e] = v
				}
				b.SetVariable(v, singleValue)
			}
			return ircode.Argument{}
		}
		for i, destNode := range destNodes {
			value := genExpression(valueNodes[i], s, b, p, vars)
			ident, ok := destNode.(*parser.IdentifierExpressionNode)
			if !ok {
				panic("Oooops")
			}
			e := s.GetVariable(ident.IdentifierToken.StringValue)
			if e == nil {
				panic("Oooops")
			}
			v, ok := vars[e]
			if !ok {
				v = b.DefineVariable(e.Name(), e.Type)
				vars[e] = v
			}
			b.SetVariable(v, value)
		}
		return ircode.Argument{}
	}
	if len(valueNodes) != len(destNodes) {
		value := genExpression(valueNodes[0], s, b, p, vars)
		et := exprType(n.Right)
		for i, destNode := range destNodes {
			det := exprType(destNode)
			var singleValue ircode.Argument
			if value.Array != nil {
				singleValue = value.Array[i]
			} else {
				ab := b.Get(nil, value)
				if types.IsArrayType(et.Type) {
					ab = ab.ArrayIndex(ircode.NewIntArg(i), det)
				} else if types.IsSliceType(et.Type) {
					ab = ab.SliceIndex(ircode.NewIntArg(i), det)
				} else if st, ok := types.GetStructType(et.Type); ok {
					ab = ab.StructField(st.Fields[i], det)
				} else if pt, ok := types.GetPointerType(et.Type); ok {
					st, _ := types.GetStructType(pt.ElementType)
					ab = ab.PointerStructField(st.Fields[i], det)
				}
				singleValue = ircode.NewVarArg(ab.GetValue())
			}
			if ident, ok := destNode.(*parser.IdentifierExpressionNode); ok {
				e := s.GetVariable(ident.IdentifierToken.StringValue)
				if e == nil {
					panic("Oooops")
				}
				v, ok := vars[e]
				if !ok {
					v = b.DefineVariable(e.Name(), e.Type)
					vars[e] = v
				}
				b.SetVariable(v, singleValue)
			} else {
				ab := genSetAndOpAccessChain(destNode, ircode.OpSet, s, b, p, vars)
				ab.SetValue(singleValue)
			}
		}
		return ircode.Argument{}
	}
	for i, destNode := range destNodes {
		value := genExpression(valueNodes[i], s, b, p, vars)
		// Trivial case
		if ident, ok := destNode.(*parser.IdentifierExpressionNode); ok {
			e := s.GetVariable(ident.IdentifierToken.StringValue)
			if e == nil {
				panic("Oooops")
			}
			v, ok := vars[e]
			if !ok {
				v = b.DefineVariable(e.Name(), e.Type)
				vars[e] = v
			}
			b.SetVariable(v, value)
		} else {
			ab := genSetAndOpAccessChain(destNode, ircode.OpSet, s, b, p, vars)
			ab.SetValue(value)
		}
	}
	return ircode.Argument{}
}

func genAssignmentOpExpression(n *parser.AssignmentExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	value := genExpression(n.Right, s, b, p, vars)
	var op ircode.Operation
	switch n.OpToken.Kind {
	case lexer.TokenAssignPlus:
		op = ircode.OpSetAndAdd
	case lexer.TokenAssignMinus:
		op = ircode.OpSetAndSub
	case lexer.TokenAssignDivision:
		op = ircode.OpSetAndDiv
	case lexer.TokenAssignAsterisk:
		op = ircode.OpSetAndMul
	case lexer.TokenAssignPercent:
		op = ircode.OpSetAndRemainder
	case lexer.TokenAssignBinaryAnd:
		op = ircode.OpSetAndBinaryAnd
	case lexer.TokenAssignBinaryOr:
		op = ircode.OpSetAndBinaryOr
	case lexer.TokenAssignCaret:
		op = ircode.OpSetAndBinaryXor
	case lexer.TokenAssignAndCaret:
		op = ircode.OpSetAndBitClear
	case lexer.TokenAssignShiftLeft:
		op = ircode.OpSetAndShiftLeft
	case lexer.TokenAssignShiftRight:
		op = ircode.OpSetAndShiftRight
	default:
		panic("Oooops")
	}
	ab := genSetAndOpAccessChain(n.Left, op, s, b, p, vars)
	ab.SetValue(value)
	return ircode.Argument{}
}

func genMetaAccessExpression(n *parser.MetaAccessNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	return ircode.NewVarArg(b.SizeOf(nil, exprType(n.Type).ToType()))
}

func genMemberAccessExpression(n *parser.MemberAccessExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n.Expression)
	if et.Type == types.PrimitiveTypeNamespace {
		element := et.NamespaceValue.Scope.GetElement(n.IdentifierToken.StringValue)
		switch element.(type) {
		case *types.Variable:
			panic("TODO")
			/*
				v, ok := vars[e]
				if !ok {
					v = b.DefineVariable(e.Name(), e.Type)
					vars[e] = v
				}
				return ircode.NewVarArg(v)
			*/
		case *types.Func:
			return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
		case *types.Namespace:
			// Generate no code
			return ircode.Argument{}
		}
		panic("Oooops")
	}
	ab := genGetAccessChain(n, s, b, p, vars)
	return ircode.NewVarArg(ab.GetValue())
}

func genArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	ab := genGetAccessChain(n, s, b, p, vars)
	return ircode.NewVarArg(ab.GetValue())
}

func genCallExpression(n *parser.MemberCallExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	// Bultin-functions?
	if ident, ok := n.Expression.(*parser.IdentifierExpressionNode); ok {
		if ident.IdentifierToken.StringValue == "len" {
			return ircode.NewVarArg(b.Len(nil, genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars)))
		} else if ident.IdentifierToken.StringValue == "cap" {
			return ircode.NewVarArg(b.Cap(nil, genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars)))
		} else if ident.IdentifierToken.StringValue == "append" {
			var args []ircode.Argument
			// The slice ...
			args = append(args, genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars))
			// Placeholder for the amount of values to add ...
			args = append(args, ircode.NewIntArg(0))
			count := 0
			for _, el := range n.Arguments.Elements[1:] {
				if unary, ok := el.Expression.(*parser.UnaryExpressionNode); !ok || unary.OpToken.Kind != lexer.TokenEllipsis {
					count++
				}
			}
			countArg := ircode.NewIntArg(count)
			// The values to add
			for _, el := range n.Arguments.Elements[1:] {
				if unary, ok := el.Expression.(*parser.UnaryExpressionNode); ok && unary.OpToken.Kind == lexer.TokenEllipsis {
					arg := genExpression(unary.Expression, s, b, p, vars)
					arg.Flags |= ircode.ArgumentIsEllipsis
					length := ircode.NewVarArg(b.Len(nil, arg))
					countArg = ircode.NewVarArg(b.Add(nil, countArg, length))
					args = append(args, arg)
				} else {
					args = append(args, genExpression(el.Expression, s, b, p, vars))
					count++
				}
			}
			args[1] = countArg
			// How much can the slice grow?
			sliceCap := ircode.NewVarArg(b.Cap(nil, args[0]))
			sliceLen := ircode.NewVarArg(b.Len(nil, args[0]))
			sliceFree := ircode.NewVarArg(b.Sub(nil, sliceCap, sliceLen))
			// Is there sufficient space?
			assertArg := ircode.NewVarArg(b.Compare(ircode.OpGreaterOrEqual, nil, sliceFree, countArg))
			b.Assert(assertArg)
			return ircode.NewVarArg(b.Append(nil, args))
		} else if ident.IdentifierToken.StringValue == "panic" {
			b.Panic(genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars))
			return ircode.Argument{}
		} else if ident.IdentifierToken.StringValue == "groupOf" {
			return ircode.NewVarArg(b.GroupOf(nil, genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars)))
		} else if ident.IdentifierToken.StringValue == "println" {
			b.Println(genExpression(n.Arguments.Elements[0].Expression, s, b, p, vars))
			return ircode.Argument{}
		} else if ident.IdentifierToken.StringValue == "take" {
			ab := genTakeAccessChain(n.Arguments.Elements[0].Expression, s, b, p, vars)
			return ircode.NewVarArg(ab.TakeValue())
		}
	}

	et := exprType(n.Expression)
	ft, ok := types.GetFuncType(et.Type)
	if !ok {
		panic("Oooops")
	}
	// Calling a member function?
	if isMemberFunction(n.Expression) {
		thisExpr := n.Expression.(*parser.MemberAccessExpressionNode).Expression
		thisEt := exprType(thisExpr)
		var thisArg ircode.Argument
		// Generate this
		if _, ok := types.GetPointerType(ft.Target); ok {
			// Generate a `this` pointer
			if _, ok := types.GetPointerType(thisEt.Type); ok {
				thisArg = genExpression(thisExpr, s, b, p, vars)
			} else {
				thisAb := genGetAccessChain(thisExpr, s, b, p, vars)
				thisAb = thisAb.AddressOf(types.NewExprType(ft.Target))
				thisArg = ircode.NewVarArg(thisAb.GetValue())
			}
		} else {
			// Generate a `this` value
			if _, ok := types.GetPointerType(thisEt.Type); ok {
				thisAb := genGetAccessChain(thisExpr, s, b, p, vars)
				thisAb = thisAb.DereferencePointer(types.NewExprType(ft.Target))
				thisArg = ircode.NewVarArg(thisAb.GetValue())
			} else {
				thisArg = genExpression(thisExpr, s, b, p, vars)
			}
		}
		args := []ircode.Argument{ircode.NewConstArg(&ircode.Constant{ExprType: et}), thisArg}
		// TODO: Parameters right-to-left
		for _, el := range n.Arguments.Elements {
			arg := genExpression(el.Expression, s, b, p, vars)
			args = append(args, arg)
		}
		return ircode.NewVarArrayArg(b.Call(nil, args))
	}
	args := []ircode.Argument{{}}
	// TODO: Parameters right-to-left
	for _, el := range n.Arguments.Elements {
		arg := genExpression(el.Expression, s, b, p, vars)
		args = append(args, arg)
	}
	args[0] = genExpression(n.Expression, s, b, p, vars)
	return ircode.NewVarArrayArg(b.Call(nil, args))
}

func genCastExpression(n *parser.CastExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	ab := genGetAccessChain(n, s, b, p, vars)
	return ircode.NewVarArg(ab.GetValue())
}

func genIncrementExpression(n *parser.IncrementExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	genSetAndOpAccessChain(n, ircode.OpSet, s, b, p, vars)
	return ircode.Argument{}
}

func genGetAccessChain(ast parser.Node, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
		et := exprType(n.Expression)
		if et.Type == types.PrimitiveTypeNamespace {
			break
		}
		ab := genGetAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainMemberAccessExpression(n, s, ab, b, p, vars)
	case *parser.ArrayAccessExpressionNode:
		ab := genGetAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, p, vars)
	case *parser.CastExpressionNode:
		ab := genGetAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainCastExpression(n, s, ab, b, p, vars)
	case *parser.UnaryExpressionNode:
		if n.OpToken.Kind == lexer.TokenAsterisk || n.OpToken.Kind == lexer.TokenAmpersand {
			ab := genGetAccessChain(n.Expression, s, b, p, vars)
			return genAccessChainUnaryExpression(n, s, ab, b, p, vars)
		}
	case *parser.IncrementExpressionNode:
		panic("Should not happen")
	}
	source := genExpression(ast, s, b, p, vars)
	return b.Get(nil, source)
}

func genSetAndOpAccessChain(ast parser.Node, op ircode.Operation, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
		et := exprType(n.Expression)
		if et.Type == types.PrimitiveTypeNamespace {
			break
		}
		ab := genSetAndOpAccessChain(n.Expression, op, s, b, p, vars)
		return genAccessChainMemberAccessExpression(n, s, ab, b, p, vars)
	case *parser.ArrayAccessExpressionNode:
		ab := genSetAndOpAccessChain(n.Expression, op, s, b, p, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, p, vars)
	case *parser.CastExpressionNode:
		ab := genSetAndOpAccessChain(n.Expression, op, s, b, p, vars)
		return genAccessChainCastExpression(n, s, ab, b, p, vars)
	case *parser.IncrementExpressionNode:
		ab := genSetAndOpAccessChain(n.Expression, op, s, b, p, vars)
		return genAccessChainIncrementExpression(n, s, ab, b, p, vars)
	case *parser.UnaryExpressionNode:
		if n.OpToken.Kind == lexer.TokenAsterisk || n.OpToken.Kind == lexer.TokenAmpersand {
			ab := genSetAndOpAccessChain(n.Expression, op, s, b, p, vars)
			return genAccessChainUnaryExpression(n, s, ab, b, p, vars)
		}
	}
	dest := genExpression(ast, s, b, p, vars)
	if dest.Var == nil {
		panic("Oooops")
	}
	return b.SetAndOp(dest.Var, op)
}

func genTakeAccessChain(ast parser.Node, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
		et := exprType(n.Expression)
		if et.Type == types.PrimitiveTypeNamespace {
			break
		}
		ab := genTakeAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainMemberAccessExpression(n, s, ab, b, p, vars)
	case *parser.ArrayAccessExpressionNode:
		ab := genTakeAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, p, vars)
	case *parser.CastExpressionNode:
		ab := genTakeAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainCastExpression(n, s, ab, b, p, vars)
	case *parser.IncrementExpressionNode:
		ab := genTakeAccessChain(n.Expression, s, b, p, vars)
		return genAccessChainIncrementExpression(n, s, ab, b, p, vars)
	case *parser.UnaryExpressionNode:
		if n.OpToken.Kind == lexer.TokenAsterisk || n.OpToken.Kind == lexer.TokenAmpersand {
			ab := genTakeAccessChain(n.Expression, s, b, p, vars)
			return genAccessChainUnaryExpression(n, s, ab, b, p, vars)
		}
	}
	source := genExpression(ast, s, b, p, vars)
	if source.Var == nil {
		panic("Oooops")
	}
	return b.Take(nil, source.Var)
}

func genAccessChainArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	if n.ColonToken != nil {
		var index1 ircode.Argument
		var index2 ircode.Argument
		if n.Index != nil {
			index1 = genExpression(n.Index, s, b, p, vars)
		} else {
			index1.Flags = ircode.ArgumentIsMissing
		}
		if n.Index2 != nil {
			index2 = genExpression(n.Index2, s, b, p, vars)
		} else {
			index2.Flags = ircode.ArgumentIsMissing
		}
		return ab.Slice(index1, index2, exprType(n))
	}
	index := genExpression(n.Index, s, b, p, vars)
	if types.IsArrayType(exprType(n.Expression).Type) {
		return ab.ArrayIndex(index, exprType(n))
	}
	return ab.SliceIndex(index, exprType(n))
}

func genAccessChainMemberAccessExpression(n *parser.MemberAccessExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	et := exprType(n.Expression)
	t := et.Type
	isPointer := false
	if pt, ok := types.GetPointerType(t); ok {
		isPointer = true
		t = pt.ElementType
	}
	if st, ok := types.GetStructType(t); ok {
		fields := st.FieldChain(n.IdentifierToken.StringValue)
		if fields == nil {
			panic("Unknown field")
		}
		for _, f := range fields {
			if isPointer {
				et = types.DerivePointerExprType(et, f.Type)
				isPointer = false
				ab = ab.PointerStructField(f, et)
			} else {
				et = types.DeriveExprType(et, f.Type)
				ab = ab.StructField(f, et)
			}
		}
		return ab
	}
	if st, ok := types.GetUnionType(t); ok {
		f := st.Field(n.IdentifierToken.StringValue)
		if f == nil {
			panic("Unknown field")
		}
		if isPointer {
			return ab.PointerStructField(f, exprType(n))
		}
		return ab.StructField(f, exprType(n))
	}
	panic("Not a struct or union")
}

func genAccessChainUnaryExpression(n *parser.UnaryExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	if n.OpToken.Kind == lexer.TokenAsterisk {
		return ab.DereferencePointer(exprType(n))
	} else if n.OpToken.Kind == lexer.TokenAmpersand {
		return ab.AddressOf(exprType(n))
	}
	panic("Ooooops")
}

func genAccessChainCastExpression(n *parser.CastExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	et := exprType(n)
	return ab.Cast(et)
}

func genAccessChainIncrementExpression(n *parser.IncrementExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	if n.Token.Kind == lexer.TokenInc {
		ab.Increment()
	} else {
		ab.Decrement()
	}
	// The access chain is complete at this point. Hence, return an empty access chain to catch compiler implementation errors
	return ircode.AccessChainBuilder{}
}

func genNewExpression(n *parser.NewExpressionNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n.Type)
	t := et.Type
	if n.NewToken.Kind == lexer.TokenNew {
		if sln, ok := n.Value.(*parser.StructLiteralNode); ok {
			st, ok := types.GetStructType(t)
			if !ok {
				panic("Oooops")
			}
			fields := make(map[string]ircode.Argument)
			for _, f := range sln.Fields {
				fields[f.NameToken.StringValue] = genExpression(f.Value, s, b, p, vars)
			}
			var values []ircode.Argument
			if st.BaseType != nil {
				if arg, ok := fields[st.BaseType.Name()]; ok {
					values = append(values, arg)
				} else {
					values = append(values, genDefaultValue(st.BaseType))
				}
			}
			for _, f := range st.Fields {
				if arg, ok := fields[f.Name]; ok {
					values = append(values, arg)
				} else {
					values = append(values, genDefaultValue(f.Type))
				}
			}
			return ircode.NewVarArg(b.Struct(nil, exprType(n), values))
		} else if pen, ok := n.Value.(*parser.ParanthesisExpressionNode); ok {
			ptr := b.Malloc(nil, exprType(n))
			value := genExpression(pen.Expression, s, b, p, vars)
			b.Set(ptr).DereferencePointer(exprType(pen.Expression)).SetValue(value)
			return ircode.NewVarArg(ptr)
		} else if n.Value == nil {
			return ircode.NewVarArg(b.Malloc(nil, exprType(n)))
		}
		panic("Oooops")
	} else if n.NewToken.Kind == lexer.TokenNewSlice {
		if pe, ok := n.Value.(*parser.ParanthesisExpressionNode); ok {
			// "new[] int(0, 100)" or "new []int(100)"
			if el, ok := pe.Expression.(*parser.ExpressionListNode); ok {
				if len(el.Elements) != 2 {
					panic("Oooops")
				}
				size := genExpression(el.Elements[0].Expression, s, b, p, vars)
				additional := genExpression(el.Elements[1].Expression, s, b, p, vars)
				cap := ircode.NewVarArg(b.Add(nil, size, additional))
				return ircode.NewVarArg(b.MallocSlice(nil, exprType(n), size, cap))
			}
			size := genExpression(pe.Expression, s, b, p, vars)
			return ircode.NewVarArg(b.MallocSlice(nil, exprType(n), size, size))
		} else if aln, ok := n.Value.(*parser.ArrayLiteralNode); ok {
			// "new[] int[1, 2, 3]"
			var values []ircode.Argument
			for _, v := range aln.Values.Elements {
				values = append(values, genExpression(v.Expression, s, b, p, vars))
			}
			return ircode.NewVarArg(b.Array(nil, exprType(n), values))
		}
		panic("Oooops")
	}
	panic("Oooops")
}

func genDefaultValue(t types.Type) ircode.Argument {
	et := types.NewExprType(t)
	et.HasValue = true
	if types.IsIntegerType(t) {
		et.IntegerValue = big.NewInt(0)
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if types.IsFloatType(t) {
		et.FloatValue = big.NewFloat(0)
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if t == types.PrimitiveTypeBool {
		et.BoolValue = false
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if types.IsSliceType(t) {
		et.IntegerValue = big.NewInt(0)
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if types.IsArrayType(t) {
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if t == types.PrimitiveTypeString {
		et.StringValue = ""
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if _, ok := types.GetStructType(t); ok {
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	if types.IsPointerType(t) {
		et.IntegerValue = big.NewInt(0)
		return ircode.Argument{Const: &ircode.Constant{ExprType: et}}
	}
	// TODO: interface type
	panic("Oooops")
}

/*
func isBuiltinFunction(n parser.Node) bool {
	if ident, ok := n.(*parser.IdentifierExpressionNode); ok {
		for _, name := range builtinFunctionNames {
			if name == ident.IdentifierToken.StringValue {
				return true
			}
		}
	}
	return false
}
*/

func isMemberFunction(n parser.Node) bool {
	et := exprType(n)
	return et.HasValue && et.FuncValue != nil && et.FuncValue.Type.Target != nil
}

func exprType(n parser.Node) *types.ExprType {
	return n.TypeAnnotation().(*types.ExprType)
}
