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
		//		if n.OpToken.Kind == lexer.TokenWalrus || n.OpToken.Kind == lexer.TokenAssign {
		//			return checkAssignExpression(n, s, log)
		//		}
		//		panic("TODO")
	case *parser.IncrementExpressionNode:
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
	var values []ircode.Argument
	if list, ok := n.Value.(*parser.ExpressionListNode); ok {
		for _, el := range list.Elements {
			arg := genExpression(el.Expression, s, b, vars)
			values = append(values, arg)
		}
	} else {
		values = []ircode.Argument{genExpression(n.Value, s, b, vars)}
	}
	if len(values) != len(n.Names) {
		panic("TODO")
	} else {
		for i, name := range n.Names {
			e := s.GetVariable(name.NameToken.StringValue)
			if e == nil {
				panic("Oooops")
			}

			v, ok := vars[e]
			if !ok {
				v = b.DefineVariable(e.Name(), e.Type)
				vars[e] = v
			}
			b.SetVariable(v, values[i])
		}
	}
	return ircode.Argument{}
}

func genArrayAccessExpression(n *parser.ArrayAccessExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	ab := genGetAccessChain(n, s, b, vars)
	return ircode.NewVarArg(ab.GetValue())
}

func genGetAccessChain(ast parser.Node, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.AccessChainBuilder {
	switch n := ast.(type) {
	case *parser.MemberAccessExpressionNode:
	case *parser.ArrayAccessExpressionNode:
		ab := genGetAccessChain(n.Expression, s, b, vars)
		return genAccessChainArrayAccessExpression(n, s, ab, b, vars)
	}
	source := genExpression(ast, s, b, vars)
	return b.Get(nil, source)
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

func exprType(n parser.Node) *types.ExprType {
	return n.TypeAnnotation().(*types.ExprType)
}
