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
	f := &Function{Name: mangleFunctionName(p, irf.Name), IsExtern: irf.IsExtern, IsExported: irf.IsExported, IsGenericInstance: irf.IsGenericInstance}
	// Do not encode the package name into the function name, if the function has external linkage
	if f.IsExtern {
		f.Name = irf.Name
	}
	b := &CBlockBuilder{}
	irft := irf.Type()
	for _, p := range irft.In {
		f.Parameters = append(f.Parameters, &FunctionParameter{Name: "p_" + p.Name, Type: mapType(mod, p.Type)})
	}
	for _, g := range irft.GroupParameters {
		f.Parameters = append(f.Parameters, &FunctionParameter{Name: "g_" + g.Name, Type: &TypeDecl{Code: "uintptr_t*"}})
	}
	if len(irft.Out) == 0 {
		f.ReturnType = NewTypeDecl("void")
	} else {
		f.ReturnType = mapType(mod, irft.ReturnType())
	}
	// Functions with external linkage have no body
	if !f.IsExtern {
		generateStatement(mod, &irf.Body, b)
		f.Body = b.Nodes
	}
	return f
}

func generatePreBlock(mod *Module, cmd *ircode.Command, b *CBlockBuilder) {
	for _, c := range cmd.PreBlock {
		generateStatement(mod, c, b)
	}
}

func generateStatement(mod *Module, cmd *ircode.Command, b *CBlockBuilder) {
	generatePreBlock(mod, cmd, b)
	switch cmd.Op {
	case ircode.OpBlock:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
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
		b2 := &CBlockBuilder{}
		for i, c := range cmd.Block {
			// The first command in a loop body is OpenScope.
			// This must be executed only when entering the loop.
			// Thus, we generate this code before we generate the for-loop.
			if i == 0 {
				if c.Op != ircode.OpOpenScope {
					panic("Oooops")
				}
				generateStatement(mod, c, b)
				b.Nodes = append(b.Nodes, f)
				continue
			}
			generateStatement(mod, c, b2)
		}
		f.Body = b2.Nodes
	case ircode.OpBreak:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
		b.Nodes = append(b.Nodes, &Break{})
	case ircode.OpContinue:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
		b.Nodes = append(b.Nodes, &Continue{})
	case ircode.OpOpenScope:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
	case ircode.OpCloseScope:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
	case ircode.OpPrintln:
		panic("TODO")
	case ircode.OpFree:
		gv := cmd.Args[0].Var
		free, freePkg := mod.Package.GetFree()
		if free == nil {
			panic("Oooops")
		}
		n := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(freePkg, free.Name)}}
		n.Args = []Node{&Constant{Code: varName(gv)}}
		b.Nodes = append(b.Nodes, n)
	case ircode.OpMerge:
		merge, mergePkg := mod.Package.GetMerge()
		if merge == nil {
			panic("Oooops")
		}
		gvAddr := generateGroupVarPointer(cmd.GroupArgs[0])
		for i := 1; i < len(cmd.GroupArgs); i++ {
			gvAddr2 := generateGroupVarPointer(cmd.GroupArgs[i])
			call := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(mergePkg, merge.Name)}}
			call.Args = []Node{gvAddr, gvAddr2}
			b.Nodes = append(b.Nodes, call)
		}
	case ircode.OpReturn:
		if len(cmd.Args) == 0 {
			b.Nodes = append(b.Nodes, &Return{})
		} else if len(cmd.Args) == 1 {
			arg1 := generateArgument(mod, cmd.Args[0], b)
			b.Nodes = append(b.Nodes, &Return{Expr: arg1})
		} else {
			sl := &CompoundLiteral{Type: mapType(mod, cmd.TypeArgs[0])}
			for _, arg := range cmd.Args {
				sl.Values = append(sl.Values, generateArgument(mod, arg, b))
			}
			b.Nodes = append(b.Nodes, &Return{Expr: sl})
		}
	case ircode.OpSet:
		arg := generateArgument(mod, cmd.Args[0], b)
		left := generateAccess(mod, arg, cmd, 1, b)
		// Unless the operation is `--` or `++`, assign the valie to the left-hand side
		if cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessInc && cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessDec {
			argRight := cmd.Args[len(cmd.Args)-1]
			right := generateArgument(mod, argRight, b)
			t := cmd.AccessChain[len(cmd.AccessChain)-1].OutputType
			if _, ok := types.GetPointerType(t.Type); ok && t.PointerDestGroup != nil && t.PointerDestGroup.Kind == types.GroupIsolate {
				// Set an isolated pointer
				gv := generateGroupVar(cmd.Args[len(cmd.Args)-1].GroupInfo())
				right = &CompoundLiteral{Type: mapExprType(mod, t), Values: []Node{right, gv}}
			} else if _, ok := types.GetSliceType(t.Type); ok && t.PointerDestGroup != nil && t.PointerDestGroup.Kind == types.GroupIsolate {
				// Set an isolated slice?
				gv := generateGroupVar(cmd.Args[len(cmd.Args)-1].GroupInfo())
				right = &CompoundLiteral{Type: mapExprType(mod, t), Values: []Node{right, gv}}
			}
			left = &Binary{Operator: "=", Left: left, Right: right}
		}
		b.Nodes = append(b.Nodes, left)
	case ircode.OpAssert:
		arg := generateArgument(mod, cmd.Args[0], b)
		n := &FunctionCall{FuncExpr: &Constant{Code: "assert"}, Args: []Node{arg}}
		b.Nodes = append(b.Nodes, n)
		mod.AddInclude("assert.h", true)
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
		if cmd.Dest[0].Kind == ircode.VarParameter || cmd.Dest[0].Kind == ircode.VarGroupParameter {
			return nil
		}
		if cmd.Dest[0].Kind == ircode.VarGlobal {
			glob := &GlobalVar{Name: varName(cmd.Dest[0]), Type: mapType(mod, cmd.Dest[0].Type.Type)}
			mod.Elements = append(mod.Elements, glob)
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
	case ircode.OpGreater:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: ">", Left: arg1, Right: arg2}
	case ircode.OpLessOrEqual:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "<=", Left: arg1, Right: arg2}
	case ircode.OpGreaterOrEqual:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: ">=", Left: arg1, Right: arg2}
	case ircode.OpMul:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "*", Left: arg1, Right: arg2}
	case ircode.OpDiv:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "/", Left: arg1, Right: arg2}
	case ircode.OpAdd:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "+", Left: arg1, Right: arg2}
	case ircode.OpSub:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "-", Left: arg1, Right: arg2}
	case ircode.OpRemainder:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "%", Left: arg1, Right: arg2}
	case ircode.OpBinaryXor:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "^", Left: arg1, Right: arg2}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapType(mod, cmd.Args[0].Type().ToType()), Expr: n}
		}
	case ircode.OpBinaryOr:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "|", Left: arg1, Right: arg2}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapType(mod, cmd.Args[0].Type().ToType()), Expr: n}
		}
	case ircode.OpBinaryAnd:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "&", Left: arg1, Right: arg2}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapType(mod, cmd.Args[0].Type().ToType()), Expr: n}
		}
	case ircode.OpShiftLeft:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: "<<", Left: arg1, Right: arg2}
	case ircode.OpShiftRight:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		n = &Binary{Operator: ">>", Left: arg1, Right: arg2}
	case ircode.OpBitClear:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "&", Left: arg1, Right: &Unary{Operator: "~", Expr: arg2}}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapType(mod, cmd.Args[0].Type().ToType()), Expr: n}
		}
	case ircode.OpMinusSign:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n = &Unary{Operator: "-", Expr: arg1}
	case ircode.OpBitwiseComplement:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n = &Unary{Operator: "~", Expr: arg1}
	case ircode.OpNot:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n = &Unary{Operator: "!", Expr: arg1}
	case ircode.OpGet:
		arg := generateArgument(mod, cmd.Args[0], b)
		n = generateAccess(mod, arg, cmd, 1, b)
		t := cmd.AccessChain[len(cmd.AccessChain)-1].OutputType
		if _, ok := types.GetPointerType(t.Type); ok && t.PointerDestGroup != nil && t.PointerDestGroup.Kind == types.GroupIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].GroupInfo)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "ptr"}}
		} else if _, ok := types.GetSliceType(t.Type); ok && t.PointerDestGroup != nil && t.PointerDestGroup.Kind == types.GroupIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].GroupInfo)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "slice"}}
		}
	case ircode.OpArray:
		var args []Node
		for _, arg := range cmd.Args {
			args = append(args, generateArgument(mod, arg, b))
		}
		if sl, ok := types.GetSliceType(cmd.Type.Type); ok {
			gv := generateAddrOfGroupVar(cmd.Dest[0])
			malloc, mallocPkg := mod.Package.GetMalloc()
			if malloc == nil {
				panic("Oooops")
			}
			// Malloc
			callMalloc := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(mallocPkg, malloc.Name)}}
			callMalloc.Args = []Node{&Constant{Code: strconv.Itoa(len(cmd.Args))}, &Sizeof{Type: mapType(mod, sl.ElementType)}, gv}
			decl := mapSlicePointerExprType(mod, cmd.Type)
			callMallocWithCast := &TypeCast{Type: decl, Expr: callMalloc}
			// Assign to a slice pointer
			slice := &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: []Node{callMallocWithCast, &Constant{Code: strconv.Itoa(len(cmd.Args))}, &Constant{Code: strconv.Itoa(len(cmd.Args))}}}
			var n2 Node
			if cmd.Dest[0].Name[0] == '%' {
				n2 = &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: slice}
			} else {
				n2 = &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: slice}
			}
			b.Nodes = append(b.Nodes, n2)
			// Assign the value to the allocated memory
			for i, arg := range args {
				n3 := &Binary{Operator: "=", Left: &Binary{Operator: "[", Left: &Constant{Code: varName(cmd.Dest[0]) + ".ptr"}, Right: &Constant{Code: strconv.Itoa(i)}}, Right: arg}
				b.Nodes = append(b.Nodes, n3)
			}
		} else {
			n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: args}
		}
	case ircode.OpStruct:
		var args []Node
		for _, arg := range cmd.Args {
			args = append(args, generateArgument(mod, arg, b))
		}
		if pt, ok := types.GetPointerType(cmd.Type.Type); ok {
			gv := generateAddrOfGroupVar(cmd.Dest[0])
			malloc, mallocPkg := mod.Package.GetMalloc()
			if malloc == nil {
				panic("Oooops")
			}
			// Malloc
			callMalloc := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(mallocPkg, malloc.Name)}}
			callMalloc.Args = []Node{&Constant{Code: "1"}, &Sizeof{Type: mapType(mod, pt.ElementType)}, gv}
			decl := mapExprType(mod, cmd.Type)
			callMallocWithCast := &TypeCast{Type: decl, Expr: callMalloc}
			var n3 Node
			ptr := &TypeCast{Type: mapType(mod, cmd.Dest[0].Type.Type), Expr: callMallocWithCast}
			if cmd.Dest[0].Name[0] == '%' {
				n3 = &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: ptr}
			} else {
				n3 = &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: ptr}
			}
			b.Nodes = append(b.Nodes, n3)
			// Assign the value to the allocated memory
			value := &CompoundLiteral{Type: mapType(mod, pt.ElementType), Values: args}
			n5 := &Binary{Operator: "=", Left: &Unary{Operator: "*", Expr: &Constant{Code: varName(cmd.Dest[0])}}, Right: value}
			b.Nodes = append(b.Nodes, n5)
		} else {
			n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: args}
		}
	case ircode.OpLen:
		n = generateLen(mod, cmd.Args[0], b)
	case ircode.OpCap:
		n = &Binary{Operator: ".", Left: generateArgument(mod, cmd.Args[0], b), Right: &Identifier{Name: "cap"}}
	case ircode.OpSizeOf:
		n = &Sizeof{Type: mapType(mod, cmd.TypeArgs[0])}
	case ircode.OpAppend:
		// Get the slice
		slice := generateArgument(mod, cmd.Args[0], b)
		// How many values will be added?
		additionalSize := generateArgument(mod, cmd.Args[1], b)
		// Store a pointer to the underlying array in a new temporary variable
		ptr := &Binary{Operator: ".", Left: slice, Right: &Identifier{Name: "ptr"}}
		varName := mod.tmpVarName()
		ptrVar := &Var{Name: varName, Type: mapSlicePointerExprType(mod, cmd.Args[0].Type()), InitExpr: ptr}
		b.Nodes = append(b.Nodes, ptrVar)
		// Iterate over all values
		for _, arg := range cmd.Args[2:] {
			if _, ok := types.GetSliceType(arg.Type().Type); ok && arg.Flags&ircode.ArgumentIsEllipsis == ircode.ArgumentIsEllipsis {
				if arg.Const != nil {
					// Append a constant
					for j := 0; j < len(arg.Const.ExprType.ArrayValue); j++ {
						right := &Constant{Code: constToString(mod, arg.Const.ExprType.ArrayValue[j])}
						left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: varName}}}
						assign := &Binary{Operator: "=", Left: left, Right: right}
						b.Nodes = append(b.Nodes, assign)
					}
				} else {
					// How many values does the slice have?
					sizeVarName := mod.tmpVarName()
					sizeVar := &Var{Name: sizeVarName, Type: mapType(mod, types.PrimitiveTypeInt), InitExpr: generateLen(mod, arg, b)}
					b.Nodes = append(b.Nodes, sizeVar)
					// Iterate over all values in the slice and append the values
					loopVarName := mod.tmpVarName()
					loopVar := &Var{Name: loopVarName, Type: mapType(mod, types.PrimitiveTypeInt), InitExpr: &Constant{Code: "0"}}
					loopCond := &Binary{Operator: "<", Left: &Identifier{Name: loopVarName}, Right: &Identifier{Name: sizeVarName}}
					loopExpr := &Unary{Operator: "++", Expr: &Identifier{Name: loopVarName}}
					loop := &For{InitExpr: loopVar, ConditionExpr: loopCond, LoopExpr: loopExpr}
					// Get the slice that will be appended
					val := generateArgument(mod, arg, b)
					// Get a pointer to the underlying array and append a value
					valPtr := &Binary{Operator: ".", Left: val, Right: &Identifier{Name: "ptr"}}
					right := &Binary{Operator: "[", Left: valPtr, Right: &Identifier{Name: loopVarName}}
					left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: varName}}}
					assign := &Binary{Operator: "=", Left: left, Right: right}
					loop.Body = append(loop.Body, assign)
					b.Nodes = append(b.Nodes, loop)
				}
			} else if at, ok := types.GetArrayType(arg.Type().Type); ok && arg.Flags&ircode.ArgumentIsEllipsis == ircode.ArgumentIsEllipsis {
				if arg.Const != nil {
					for j := 0; j < len(arg.Const.ExprType.ArrayValue); j++ {
						right := &Constant{Code: constToString(mod, arg.Const.ExprType.ArrayValue[j])}
						left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: varName}}}
						assign := &Binary{Operator: "=", Left: left, Right: right}
						b.Nodes = append(b.Nodes, assign)
					}
				} else {
					loopVarName := mod.tmpVarName()
					loopVar := &Var{Name: loopVarName, Type: mapType(mod, types.PrimitiveTypeInt), InitExpr: &Constant{Code: "0"}}
					loopCond := &Binary{Operator: "<", Left: &Identifier{Name: loopVarName}, Right: &Constant{Code: strconv.FormatUint(at.Size, 10)}}
					loopExpr := &Unary{Operator: "++", Expr: &Identifier{Name: loopVarName}}
					loop := &For{InitExpr: loopVar, ConditionExpr: loopCond, LoopExpr: loopExpr}
					val := generateArgument(mod, arg, b)
					valPtr := &Binary{Operator: ".", Left: val, Right: &Identifier{Name: "arr"}}
					right := &Binary{Operator: "[", Left: valPtr, Right: &Identifier{Name: loopVarName}}
					left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: varName}}}
					assign := &Binary{Operator: "=", Left: left, Right: right}
					loop.Body = append(loop.Body, assign)
					b.Nodes = append(b.Nodes, loop)
				}
			} else {
				val := generateArgument(mod, arg, b)
				left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: varName}}}
				assign := &Binary{Operator: "=", Left: left, Right: val}
				b.Nodes = append(b.Nodes, assign)
			}
		}
		ptr = &Binary{Operator: ".", Left: slice, Right: &Identifier{Name: "ptr"}}
		capacity := &Binary{Operator: ".", Left: slice, Right: &Identifier{Name: "cap"}}
		size := &Binary{Operator: ".", Left: slice, Right: &Identifier{Name: "size"}}
		size = &Binary{Operator: "+", Left: size, Right: additionalSize}
		n = &CompoundLiteral{Type: mapType(mod, cmd.Type.Type), Values: []Node{ptr, size, capacity}}
	default:
		fmt.Printf("%v\n", cmd.Op)
		panic("Ooooops")
	}
	if len(cmd.Dest) != 0 && cmd.Dest[0] != nil {
		if cmd.Dest[0].Name[0] == '%' {
			if n == nil {
				return nil
			}
			return &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: n}
		}
		if n == nil {
			return nil
		}
		return &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: n}
	}
	return n
}

func generateLen(mod *Module, arg ircode.Argument, b *CBlockBuilder) Node {
	if arg.Const != nil {
		if _, ok := types.GetSliceType(arg.Const.ExprType.Type); ok {
			return &Constant{Code: strconv.Itoa(len(arg.Const.ExprType.ArrayValue))}
		}
		if arr, ok := types.GetArrayType(arg.Const.ExprType.Type); ok {
			return &Constant{Code: strconv.Itoa(int(arr.Size))}
		}
		if arg.Const.ExprType.Type == types.PrimitiveTypeString {
			return &Constant{Code: strconv.Itoa(len(arg.Const.ExprType.StringValue))}
		}
	}
	if _, ok := types.GetSliceType(arg.Var.Type.Type); ok {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "size"}}
	}
	// TODO: String
	panic("Oooops")
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
		case ircode.AccessCall:
			ft, ok := types.GetFuncType(a.InputType.Type)
			if !ok {
				panic("Ooooops")
			}
			irft := ircode.NewFunctionType(ft)
			var args []Node
			for range irft.In {
				arg := generateArgument(mod, cmd.Args[argIndex], b)
				argIndex++
				args = append(args, arg)
			}
			expr = &FunctionCall{FuncExpr: expr, Args: args}
		case ircode.AccessCast:
			et := a.OutputType
			switch et.TypeConversionValue {
			case types.ConvertStringToByte:
				// Do nothing by intention
			case types.ConvertPointerToPointer:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertSliceToPointer:
				expr = &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "ptr"}}
			case types.ConvertIntegerToPointer:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertPointerToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertPointerToSlice:
				panic("TODO")
			case types.ConvertStringToByteSlice:
				panic("TODO")
			case types.ConvertPointerToString:
				panic("TODO")
			case types.ConvertByteSliceToString:
				panic("TODO")
			case types.ConvertIntegerToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertFloatToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertBoolToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertRuneToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConverIntegerToFloat:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertFloatToFloat:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertIntegerToBool:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertIntegerToRune:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			default:
				fmt.Printf("%v\n", et.TypeConversionValue)
				panic("Ooooops")
			}
		}
	}
	return expr
}

func generateArgument(mod *Module, arg ircode.Argument, b *CBlockBuilder) Node {
	if arg.Const != nil {
		return generateConstant(mod, arg.Const)
	} else if arg.Cmd != nil {
		return generateCommand(mod, arg.Cmd, b)
	} else if arg.Var != nil {
		return &Constant{Code: varName(arg.Var)}
	}
	panic("Oooops")
}

func generateConstant(mod *Module, c *ircode.Constant) Node {
	return &Constant{Code: constToString(mod, c.ExprType)}
}

func constToString(mod *Module, et *types.ExprType) string {
	if types.IsUnsignedIntegerType(et.Type) {
		// TODO: The correctness of this depends on the target platform
		if et.Type == types.PrimitiveTypeUint64 || et.Type == types.PrimitiveTypeUintptr {
			return et.IntegerValue.Text(10) + "ull"
		}
		return et.IntegerValue.Text(10) + "u"
	}
	if types.IsIntegerType(et.Type) {
		// TODO: The correctness of this depends on the target platform
		if et.Type == types.PrimitiveTypeInt64 {
			return et.IntegerValue.Text(10) + "ll"
		}
		return et.IntegerValue.Text(10)
	}
	if types.IsFloatType(et.Type) {
		return et.FloatValue.Text('f', 5)
	}
	if et.Type == types.PrimitiveTypeRune {
		return "0x" + et.IntegerValue.Text(16)
	}
	if et.Type == types.PrimitiveTypeString {
		str := mod.AddString(et.StringValue)
		return str.Identifier + ".data"
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
		if et.PointerDestGroup != nil && et.PointerDestGroup.Kind == types.GroupIsolate {
			return "(" + mapExprType(mod, et).ToString("") + "){{0, 0, 0}, 0}"
		}
		return "(" + mapExprType(mod, et).ToString("") + "){0, 0, 0}"
	}
	if _, ok := types.GetArrayType(et.Type); ok {
		str := "(" + mapExprType(mod, et).ToString("") + "){"
		for i, element := range et.ArrayValue {
			if i > 0 {
				str += ", "
			}
			str += constToString(mod, element)
		}
		// Empty initializer lists are not allowed in C
		if len(et.ArrayValue) == 0 {
			str += "0"
		}
		return str + "}"
	}
	if types.IsSliceType(et.Type) {
		panic("Ooooops")
	}
	if _, ok := types.GetPointerType(et.Type); ok {
		if et.IntegerValue != nil {
			if et.PointerDestGroup != nil && et.PointerDestGroup.Kind == types.GroupIsolate {
				return "(" + mapExprType(mod, et).ToString("") + "){0x" + et.IntegerValue.Text(16) + ", 0}"
			}
			if et.IntegerValue.Uint64() == 0 {
				return "((" + mapExprType(mod, et).ToString("") + ")0)"
			}
			return "((" + mapExprType(mod, et).ToString("") + ")0x" + et.IntegerValue.Text(16) + ")"
		}
		panic("Oooops")
	}
	if _, ok := types.GetStructType(et.Type); ok {
		str := "(" + mapExprType(mod, et).ToString("") + "){"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += "." + name + "=(" + constToString(mod, element) + ")"
			i++
		}
		return str + "}"
	}
	if _, ok := types.GetFuncType(et.Type); ok {
		irpkg, irf := resolveFunc(mod, et.FuncValue)
		if irf.IsExtern {
			return irf.Name
		}
		return mangleFunctionName(irpkg, irf.Name)
	}
	fmt.Printf("%T\n", et.Type)
	panic("TODO")
}

func varName(v *ircode.Variable) string {
	v = v.Original
	switch v.Kind {
	case ircode.VarParameter:
		return "p_" + v.Name
	case ircode.VarGroupParameter:
		return "g_" + v.Name
	case ircode.VarTemporary:
		return "tmp_" + v.Name[1:]
	case ircode.VarDefault:
		return "v_" + v.Name
	case ircode.VarGlobal:
		return "pkg_" + v.Name
	}
	panic("Oooops")
}

func mangleFunctionName(p *irgen.Package, name string) string {
	data := p.TypePackage.FullPath() + "//" + name
	sum := sha256.Sum256([]byte(data))
	sumHex := hex.EncodeToString(sum[:])
	return name + "_" + sumHex
}

func generateGroupVar(group ircode.IGroupVariable) Node {
	gv := group.Variable()
	if gv == nil {
		println("NO VAR FOR " + group.GroupVariableName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Unary{Operator: "*", Expr: &Constant{Code: varName(gv)}}
	}
	return &Constant{Code: varName(gv)}
}

func generateAddrOfGroupVar(v *ircode.Variable) Node {
	if v.GroupInfo == nil {
		panic("Ooooops")
	}
	gv := v.GroupInfo.Variable()
	if gv == nil {
		println("NO VAR FOR " + v.GroupInfo.GroupVariableName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Constant{Code: varName(gv)}
	}
	return &Unary{Operator: "&", Expr: &Constant{Code: varName(gv)}}
}

func generateGroupVarPointer(group ircode.IGroupVariable) Node {
	gv := group.Variable()
	if gv == nil {
		println("NO VAR FOR " + group.GroupVariableName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Constant{Code: varName(gv)}
	}
	return &Unary{Operator: "&", Expr: &Constant{Code: varName(gv)}}
}
