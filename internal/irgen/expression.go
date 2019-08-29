package irgen

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/lexer"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genExpression(ast parser.Node, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	switch n := ast.(type) {
	case *parser.ExpressionListNode:
		for _, e := range n.Elements {
			genExpression(e.Expression, s, b, vars)
		}
		return ircode.Argument{}
	case *parser.BinaryExpressionNode:
		return genBinaryExpression(n, s, b, vars)
	case *parser.UnaryExpressionNode:
		return genUnaryExpression(n, s, b, vars)
	case *parser.IsTypeExpressionNode:
	case *parser.MemberAccessExpressionNode:
	case *parser.MemberCallExpressionNode:
	case *parser.ArrayAccessExpressionNode:
		return genArrayAccessExpression(n, s, b, vars)
	case *parser.ConstantExpressionNode:
		return genConstantExpression(n, s, b, vars)
	case *parser.IdentifierExpressionNode:
		return genIdentifierExpression(n, s, b, vars)
	case *parser.NewExpressionNode:
	case *parser.ParanthesisExpressionNode:
		return genExpression(n.Expression, s, b, vars)
	case *parser.AssignmentExpressionNode:
		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
			return genAssignmentExpression(n, s, b, vars)
		}
		panic("TODO")
	case *parser.IncrementExpressionNode:
		return genIncrementExpression(n, s, b, vars)
	case *parser.VarExpressionNode:
		return genVarExpression(n, s, b, vars)
	case *parser.ArrayLiteralNode:
	case *parser.StructLiteralNode:
	case *parser.ClosureExpressionNode:
	}
	fmt.Printf("%T\n", ast)
	// panic("Should not happen")
	// HACK
	return ircode.NewIntArg(0)
}

func genIdentifierExpression(n *parser.IdentifierExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
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
		panic("TODO")
	}
	panic("Should not happen")
}

func genConstantExpression(n *parser.ConstantExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
}

func genBinaryExpression(n *parser.BinaryExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.HasValue {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	left := genExpression(n.Left, s, b, vars)
	right := genExpression(n.Right, s, b, vars)
	tleft := exprType(n.Left)
	switch n.OpToken.Kind {
	case lexer.TokenLogicalOr:
		return ircode.NewVarArg(b.BooleanOp(ircode.OpLogicalOr, nil, left, right))
	case lexer.TokenLogicalAnd:
		return ircode.NewVarArg(b.BooleanOp(ircode.OpLogicalAnd, nil, left, right))
	case lexer.TokenEqual:
		if tleft.Type == types.PrimitiveTypeBool || types.IsIntegerType(tleft.Type) || types.IsFloatType(tleft.Type) || types.IsPointerType(tleft.Type) || types.IsSliceType(tleft.Type) {
			return ircode.NewVarArg(b.Compare(ircode.OpEqual, nil, left, right))
		}
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
		return ircode.NewVarArg(b.Add(nil, left, right))
	case lexer.TokenMinus, lexer.TokenAsterisk, lexer.TokenDivision:
	case lexer.TokenBinaryOr, lexer.TokenAmpersand, lexer.TokenCaret, lexer.TokenPercent, lexer.TokenBitClear:
	case lexer.TokenShiftLeft, lexer.TokenShiftRight:
	}
	panic("Should not happen")
}

func genUnaryExpression(n *parser.UnaryExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	et := exprType(n)
	if et.HasValue {
		return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
	}
	panic("TODO")
}

func genVarExpression(n *parser.VarExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
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
		value := genExpression(valueNodes[0], s, b, vars)
		et := exprType(n.Value)
		for i, name := range n.Names {
			e := s.GetVariable(name.NameToken.StringValue)
			if e == nil {
				panic("Oooops")
			}
			ab := b.Get(nil, value)
			if types.IsArrayType(et.Type) {
				ab = ab.ArrayIndex(ircode.NewIntArg(i), e.Type)
			} else {
				ab = ab.SliceIndex(ircode.NewIntArg(i), e.Type)
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
		value := genExpression(valueNodes[i], s, b, vars)
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

func genAssignmentExpression(n *parser.AssignmentExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
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
			value := genExpression(valueNodes[0], s, b, vars)
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
				ab := b.Get(nil, value)
				if types.IsArrayType(et.Type) {
					ab = ab.ArrayIndex(ircode.NewIntArg(i), e.Type)
				} else {
					ab = ab.SliceIndex(ircode.NewIntArg(i), e.Type)
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
		for i, destNode := range destNodes {
			value := genExpression(valueNodes[i], s, b, vars)
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
		value := genExpression(valueNodes[0], s, b, vars)
		et := exprType(n.Right)
		for i, destNode := range destNodes {
			det := exprType(destNode)
			ab := b.Get(nil, value)
			if types.IsArrayType(et.Type) {
				ab = ab.ArrayIndex(ircode.NewIntArg(i), det)
			} else {
				ab = ab.SliceIndex(ircode.NewIntArg(i), det)
			}
			singleValue := ab.GetValue()
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
				b.SetVariable(v, ircode.NewVarArg(singleValue))
			} else {
				ab := genSetAccessChain(destNode, s, b, vars)
				ab.SetValue(ircode.NewVarArg(singleValue))
			}
		}
		return ircode.Argument{}
	}
	for i, destNode := range destNodes {
		value := genExpression(valueNodes[i], s, b, vars)
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
			ab := genSetAccessChain(destNode, s, b, vars)
			ab.SetValue(value)
		}
	}
	return ircode.Argument{}
}

func genArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	ab := genGetAccessChain(n, s, b, vars)
	return ircode.NewVarArg(ab.GetValue())
}

func genIncrementExpression(n *parser.IncrementExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	genSetAccessChain(n, s, b, vars)
	return ircode.Argument{}
}

func genGetAccessChain(ast parser.Node, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
	case *parser.ArrayAccessExpressionNode:
		ab := genGetAccessChain(n.Expression, s, b, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, vars)
	case *parser.IncrementExpressionNode:
		panic("Should not happen")
	}
	source := genExpression(ast, s, b, vars)
	return b.Get(nil, source)
}

func genSetAccessChain(ast parser.Node, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
	case *parser.ArrayAccessExpressionNode:
		ab := genSetAccessChain(n.Expression, s, b, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, vars)
	case *parser.IncrementExpressionNode:
		ab := genSetAccessChain(n.Expression, s, b, vars)
		return genAccessChainIncrementExpression(n, s, ab, b, vars)
	}
	dest := genExpression(ast, s, b, vars)
	if dest.Var.Var == nil {
		panic("Oooops")
	}
	return b.Set(dest.Var.Var)
}

func genAccessChainArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	if n.ColonToken != nil {
		index1 := genExpression(n.Index, s, b, vars)
		index2 := genExpression(n.Index2, s, b, vars)
		// TODO: Missing indices
		return ab.Slice(index1, index2, exprType(n))
	}
	index := genExpression(n.Index, s, b, vars)
	if types.IsArrayType(exprType(n.Expression).Type) {
		return ab.ArrayIndex(index, exprType(n))
	}
	return ab.SliceIndex(index, exprType(n))
}

func genAccessChainIncrementExpression(n *parser.IncrementExpressionNode, s *types.Scope, ab ircode.AccessChainBuilder, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	if n.Token.Kind == lexer.TokenInc {
		ab.Increment()
	} else {
		ab.Decrement()
	}
	// The access chain is complete at this point. Hence, return an empty access chain to catch compiler implementation errors
	return ircode.AccessChainBuilder{}
}

func exprType(n parser.Node) *types.ExprType {
	return n.TypeAnnotation().(*types.ExprType)
}
