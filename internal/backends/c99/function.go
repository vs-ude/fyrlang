package c99

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"

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
	for _, p := range irf.Func.Type.In.Params {
		f.Parameters = append(f.Parameters, &FunctionParameter{Name: "p_" + p.Name, Type: mapType(mod, p.Type)})
	}
	if len(irf.Func.Type.Out.Params) == 0 {
		f.ReturnType = NewTypeDecl("void")
	} else if len(irf.Func.Type.Out.Params) == 1 {
		f.ReturnType = mapType(mod, irf.Func.Type.Out.Params[0].Type)
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
	case ircode.OpIf:
		arg := generateArgument(mod, cmd.Args[0], b)
		ifclause := &If{Expr: arg}
		b.Nodes = append(b.Nodes, ifclause)
		b2 := &CBlockBuilder{}
		for _, c := range cmd.Block {
			generateStatement(mod, c, b2)
		}
		ifclause.Body = b2.Nodes
		if cmd.Else != nil {
			b3 := &CBlockBuilder{}
			generateStatement(mod, cmd.Else, b3)
			ifclause.ElseClause = &Else{Body: b3.Nodes}
		}
	case ircode.OpLoop:
		f := &For{}
		b.Nodes = append(b.Nodes, f)
		b2 := &CBlockBuilder{}
		for _, c := range cmd.Block {
			generateStatement(mod, c, b2)
		}
		f.Body = b2.Nodes
	case ircode.OpBreak:
		b.Nodes = append(b.Nodes, &Break{})
	case ircode.OpContinue:
		b.Nodes = append(b.Nodes, &Continue{})
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
		if cmd.Dest[0].Kind == ircode.VarParameter {
			return nil
		}
		return &Var{Name: varName(cmd.Dest[0]), Type: mapType(mod, cmd.Dest[0].Type.Type)}
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
	case ircode.OpLess:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "<", Left: arg1, Right: arg2}
	case ircode.OpNot:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n = &Unary{Operator: "!", Expr: arg1}
	case ircode.OpGet:
		arg := generateArgument(mod, cmd.Args[0], b)
		n = generateAccess(mod, arg, cmd, 1, b)
	case ircode.OpArray:
		var args []Node
		/*
			// Something like `%78 = []` but the type is [3]int. In this case the `[]` must be expanded
			if a, ok := types.GetArrayType(cmd.Type.Type); ok && len(cmd.Args) == 0 && a.Size > 0 {
				// For large arrays fill the destination variable with memset (if there is a destination variable).
				if a.Size > 12 && cmd.Dest[0].Var != nil {
					mod.AddInclude("string.h", true)
					if cmd.Dest[0].Var.Name[0] == '%' {
						b.Nodes = append(b.Nodes, &Var{Name: varName(cmd.Dest[0].Var), Type: mapType(mod, cmd.Dest[0].Var.Type.Type)})
					}
					return &Constant{Code: "memset(&" + varName(cmd.Dest[0].Var) + ", 0, " + strconv.FormatUint(a.Size, 10) + ")"}
				}
				for i := uint64(0); i < a.Size; i++ {
					// TODO: Use proper default value here
					args = append(args, &Constant{Code: "0"})
				}
				n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: args}
			} else { */
		for _, arg := range cmd.Args {
			args = append(args, generateArgument(mod, arg, b))
		}
		n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: args}
	case ircode.OpStruct:
		var args []Node
		for _, arg := range cmd.Args {
			args = append(args, generateArgument(mod, arg, b))
		}
		n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: args}
	default:
		return nil
		//	panic("Ooooops")
	}
	if cmd.Dest[0] != nil {
		if cmd.Dest[0].Name[0] == '%' {
			return &Var{Name: varName(cmd.Dest[0]), Type: mapType(mod, cmd.Dest[0].Type.Type), InitExpr: n}
		}
		return &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: n}
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
			// Check boundary in case the index is not a constant
			if cmd.Args[argIndex].Const == nil {
				idxVarName := "idx_" + strconv.Itoa(len(b.Nodes))
				b.Nodes = append(b.Nodes, &Var{Name: idxVarName, Type: NewTypeDecl("int"), InitExpr: idx})
				idx = &Identifier{Name: idxVarName}
				at, ok := types.GetArrayType(a.InputType.Type)
				if !ok {
					panic("Oooops")
				}
				cond := &Binary{Operator: ">=", Left: idx, Right: &Constant{Code: strconv.FormatUint(at.Size, 10)}}
				test := &If{Expr: cond}
				test.Body = append(test.Body, &Constant{Code: "exit(1)"})
				b.Nodes = append(b.Nodes, test)
			}
			argIndex++
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
		return generateConstant(mod, arg.Const, b)
	} else if arg.Cmd != nil {
		return generateCommand(mod, arg.Cmd, b)
	} else if arg.Var != nil {
		return &Constant{Code: varName(arg.Var)}
	}
	panic("Oooops")
}

func generateConstant(mod *Module, c *ircode.Constant, b *CBlockBuilder) Node {
	return &Constant{Code: constToString(mod, c.ExprType, b)}
}

func constToString(mod *Module, et *types.ExprType, b *CBlockBuilder) string {
	if types.IsUnsignedIntegerType(et.Type) {
		return et.IntegerValue.Text(10) + "u"
	}
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
	if types.IsSliceType(et.Type) && et.IntegerValue != nil {
		if et.IntegerValue.Uint64() != 0 {
			panic("Oooops, should only be possible for null pointers")
		}
		return "(" + mapType(mod, et.Type).ToString("") + "){0, 0, 0}"
	}
	if _, ok := types.GetArrayType(et.Type); ok {
		str := "(" + mapType(mod, et.Type).ToString("") + "){"
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(mod, element, b)
		}
		// Empty initializer lists are not allowed in C
		if len(et.ArrayValue) == 0 {
			str += "0"
		}
		return str + "}"
	}
	if types.IsSliceType(et.Type) {
		// TODO: Memory allocation is required
		str := "(" + mapType(mod, et.Type).ToString("") + "){"
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(mod, element, b)
		}
		// Empty initializer lists are not allowed in C
		if len(et.ArrayValue) == 0 {
			str += "0"
		}
		return str + "}"
	}
	if ptr, ok := types.GetPointerType(et.Type); ok {
		if et.IntegerValue != nil {
			if et.IntegerValue.Uint64() == 0 {
				return "((" + mapType(mod, et.Type).ToString("") + ")0)"
			}
			return "((" + mapType(mod, et.Type).ToString("") + ")0x" + et.IntegerValue.Text(16) + ")"
		}
		_, ok := types.GetStructType(ptr.ElementType)
		if !ok {
			panic("Oooops")
		}
		str := "(" + mapType(mod, et.Type).ToString("") + "){"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += "." + name + "=(" + constToString(mod, element, b) + ")"
			i++
		}
		// Empty initializer lists are not allowed in C
		if len(et.StructValue) == 0 {
			str += "0"
		}
		return str + "}"
	}
	if _, ok := types.GetStructType(et.Type); ok {
		str := "(" + mapType(mod, et.Type).ToString("") + "){"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += "." + name + "=(" + constToString(mod, element, b) + ")"
			i++
		}
		return str + "}"
	}
	fmt.Printf("%T\n", et.Type)
	panic("TODO")
}

func varName(v *ircode.Variable) string {
	v = v.Original
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
