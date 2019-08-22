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
	case *parser.IsTypeExpressionNode:
	case *parser.MemberAccessExpressionNode:
	case *parser.MemberCallExpressionNode:
	case *parser.ArrayAccessExpressionNode:
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

func genBinaryExpression(n *parser.BinaryExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
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

func genConstantExpression(n *parser.ConstantExpressionNode, s *types.Scope, b *ircode.Builder, vars map[*types.Variable]*ircode.Variable) ircode.Argument {
	return ircode.NewConstArg(&ircode.Constant{ExprType: exprType(n)})
}

func exprType(n parser.Node) *types.ExprType {
	return n.TypeAnnotation().(*types.ExprType)
}
