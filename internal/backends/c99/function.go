package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/irgen"
	"github.com/vs-ude/fyrlang/internal/types"
)

// CBlockBuilder ...
type CBlockBuilder struct {
	Nodes []Node
}

// Generates a C-AST function from an IR function
func generateFunction(mod *Module, p *irgen.Package, irf *ircode.Function) *Function {
	f := &Function{Name: mangleFunctionName(p, irf.Name)}
	b := &CBlockBuilder{}
	for _, p := range irf.Type.In.Params {
		f.Parameters = append(f.Parameters, &FunctionParameter{Name: "p_" + p.Name, Type: mapType(mod, p.Type)})
	}
	if len(irf.Type.Out.Params) == 0 {
		f.ReturnType = NewTypeDecl("void")
	} else if len(irf.Type.Out.Params) == 1 {
		f.ReturnType = mapType(mod, irf.Type.Out.Params[0].Type)
	} else {
		// TODO
		f.ReturnType = NewTypeDecl("void")
	}
	f.IsGenericInstance = irf.IsGenericInstance
	f.IsExported = irf.IsExported
	generateStatement(mod, &irf.Body, b)
	f.Body = b.Nodes
	return f
}

func generateStatement(mod *Module, cmd *ircode.Command, b *CBlockBuilder) {
	switch cmd.Op {
	case ircode.OpBlock:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
	case ircode.OpSet:
		arg := generateArgument(mod, cmd.Args[0], b)
		left := generateAccess(mod, arg, cmd, 1, b)
		if cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessInc && cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessDec {
			left = &Binary{Operator: "=", Left: left, Right: generateArgument(mod, cmd.Args[len(cmd.Args)-1], b)}
		}
		b.Nodes = append(b.Nodes, left)
	default:
		n := generateCommand(mod, cmd, b)
		if n != nil {
			b.Nodes = append(b.Nodes, n)
		}
	}
}

func generateCommand(mod *Module, cmd *ircode.Command, b *CBlockBuilder) Node {
	var n Node
	switch cmd.Op {
	case ircode.OpBlock:
		panic("Oooops")
	case ircode.OpDefVariable:
		if cmd.Dest[0].Var.Kind == ircode.VarParameter {
			return nil
		}
		return &Var{Name: varName(cmd.Dest[0].Var), Type: mapType(mod, cmd.Dest[0].Var.Type.Type)}
	case ircode.OpSetVariable:
		n = generateArgument(mod, cmd.Args[0], b)
	case ircode.OpLogicalAnd:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "&&", Left: arg1, Right: arg2}
	case ircode.OpLogicalOr:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "||", Left: arg1, Right: arg2}
	case ircode.OpEqual:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "==", Left: arg1, Right: arg2}
	case ircode.OpNotEqual:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "!=", Left: arg1, Right: arg2}
	case ircode.OpGet:
		arg := generateArgument(mod, cmd.Args[0], b)
		n = generateAccess(mod, arg, cmd, 1, b)
	default:
		return nil
		//	panic("Ooooops")
	}
	if cmd.Dest[0].Var != nil {
		if cmd.Dest[0].Var.Name[0] == '%' {
			return &Var{Name: varName(cmd.Dest[0].Var), Type: mapType(mod, cmd.Dest[0].Var.Type.Type), InitExpr: n}
		}
		return &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0].Var)}, Right: n}
	}
	return n
}

func generateAccess(mod *Module, expr Node, cmd *ircode.Command, argIndex int, b *CBlockBuilder) Node {
	for _, a := range cmd.AccessChain {
		switch a.Kind {
		case ircode.AccessAddressOf:
			expr = &Unary{Expr: expr, Operator: "&"}
		case ircode.AccessArrayIndex:
			idx := generateArgument(mod, cmd.Args[argIndex], b)
			argIndex++
			// TODO: Check boundary
			expr = &Binary{Left: &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "arr"}}, Right: idx, Operator: "["}
		case ircode.AccessCast:
			// TODO
		case ircode.AccessDec:
			expr = &Unary{Expr: expr, Operator: "--"}
		case ircode.AccessInc:
			expr = &Unary{Expr: expr, Operator: "++"}
		case ircode.AccessDereferencePointer:
			// TODO: Null check
			expr = &Unary{Expr: expr, Operator: "*"}
		case ircode.AccessPointerToStruct:
			// TODO: Null check
			expr = &Binary{Left: expr, Right: &Identifier{Name: a.Field.Name}, Operator: "->"}
		case ircode.AccessStruct:
			expr = &Binary{Left: expr, Right: &Identifier{Name: a.Field.Name}, Operator: "."}
		case ircode.AccessUnsafeArrayIndex:
			expr = &Binary{Left: expr, Right: generateArgument(mod, cmd.Args[argIndex], b), Operator: "["}
			argIndex++
		case ircode.AccessSlice:
			// TODO:
		case ircode.AccessSliceIndex:
			idx := generateArgument(mod, cmd.Args[argIndex], b)
			argIndex++
			// TODO: Check boundary
			expr = &Binary{Left: &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "ptr"}}, Right: idx, Operator: "["}
		}
	}
	return expr
}

func generateArgument(mod *Module, arg ircode.Argument, b *CBlockBuilder) Node {
	if arg.Const != nil {
		return generateConstant(arg.Const)
	} else if arg.Cmd != nil {
		return generateCommand(mod, arg.Cmd, b)
	} else if arg.Var.Var != nil {
		return &Constant{Code: varName(arg.Var.Var)}
	}
	panic("Oooops")
}

func generateConstant(c *ircode.Constant) Node {
	return &Constant{Code: constToString(c.ExprType)}
}

func constToString(et *types.ExprType) string {
	if types.IsIntegerType(et.Type) {
		return et.IntegerValue.Text(10)
	}
	if types.IsFloatType(et.Type) {
		return et.FloatValue.Text('f', 5)
	}
	if et.Type == types.PrimitiveTypeString {
		return et.StringValue
	}
	if et.Type == types.PrimitiveTypeBool {
		if et.BoolValue {
			return "true"
		}
		return "false"
	}
	if types.IsArrayType(et.Type) || types.IsSliceType(et.Type) {
		str := "{"
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(element)
		}
		return str + "}"
	}
	if types.IsPointerType(et.Type) {
		if et.IntegerValue.Uint64() == 0 {
			return "0"
		}
		return "0x" + et.IntegerValue.Text(16)
	}
	/*
		case *StructType:
			if c.Composite == nil {
				return "{zero}"
			}
			str := "{"
			for i, element := range c.Composite {
				if i > 0 {
					str += ", "
				}
				str += x.Fields[i].Name + ": "
				str += element.ToString()
			}
			return str + "}"
		}
	*/
	fmt.Printf("%T\n", et.Type)
	panic("TODO")
}

func varName(v *ircode.Variable) string {
	switch v.Kind {
	case ircode.VarParameter:
		return "p_" + v.Name
	case ircode.VarTemporary:
		return "tmp_" + v.Name[1:]
	case ircode.VarDefault:
		return "v_" + v.Name
	}
	panic("Oooops")
}

func mangleFunctionName(p *irgen.Package, name string) string {
	data := p.TypePackage.FullPath() + "//" + name
	sum := sha256.Sum256([]byte(data))
	sumHex := hex.EncodeToString(sum[:])
	return name + "_" + sumHex
}
