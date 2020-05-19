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
	for _, g := range irft.GroupSpecifiers {
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

func generateIterBlock(mod *Module, cmd *ircode.Command, b *CBlockBuilder) {
	for _, c := range cmd.IterBlock {
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
		mod.startLoop()
		f := &For{}
		b2 := &CBlockBuilder{}
		for _, c := range cmd.Block[:len(cmd.Block)-1] {
			generateStatement(mod, c, b2)
		}
		l := &Label{Name: mod.loopLabel()}
		b2.Nodes = append(b2.Nodes, l)
		generateIterBlock(mod, cmd, b2)
		generateStatement(mod, cmd.Block[len(cmd.Block)-1], b2)
		f.Body = b2.Nodes
		b.Nodes = append(b.Nodes, f)
		mod.endLoop()
	case ircode.OpBreak:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
		b.Nodes = append(b.Nodes, &Break{})
	case ircode.OpContinue:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
		b.Nodes = append(b.Nodes, &Goto{Name: mod.loopLabel()})
		// b.Nodes = append(b.Nodes, &Continue{})
	case ircode.OpOpenScope:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
	case ircode.OpCloseScope:
		for _, c := range cmd.Block {
			generateStatement(mod, c, b)
		}
	case ircode.OpPrintln:
		printlnFunc, runtimePkg := mod.Package.GetPrintln()
		if printlnFunc == nil {
			panic("Oooops")
		}
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(runtimePkg, printlnFunc.Name)}}
		n.Args = []Node{arg1, generateGroupVarPointer(cmd.GroupArgs[0])}
		b.Nodes = append(b.Nodes, n)
	case ircode.OpPanic:
		panicFunc, runtimePkg := mod.Package.GetPanic()
		if panicFunc == nil {
			panic("Oooops")
		}
		arg1 := generateArgument(mod, cmd.Args[0], b)
		n := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(runtimePkg, panicFunc.Name)}}
		n.Args = []Node{arg1, generateGroupVarPointer(cmd.GroupArgs[0])}
		b.Nodes = append(b.Nodes, n)
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
		// Unless the operation is `--` or `++`, assign the value to the left-hand side
		if cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessInc && cmd.AccessChain[len(cmd.AccessChain)-1].Kind != ircode.AccessDec {
			argRight := cmd.Args[len(cmd.Args)-1]
			right := generateArgument(mod, argRight, b)
			t := cmd.AccessChain[len(cmd.AccessChain)-1].OutputType
			if _, ok := types.GetPointerType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				// Set an isolated pointer
				gv := generateGroupVar(cmd.Args[len(cmd.Args)-1].Grouping())
				right = &CompoundLiteral{Type: mapExprType(mod, t), Values: []Node{right, gv}}
			} else if _, ok := types.GetSliceType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
				// Set an isolated slice?
				gv := generateGroupVar(cmd.Args[len(cmd.Args)-1].Grouping())
				right = &CompoundLiteral{Type: mapExprType(mod, t), Values: []Node{right, gv}}
			}
			left = &Binary{Operator: "=", Left: left, Right: right}
		}
		b.Nodes = append(b.Nodes, left)
	case ircode.OpSetAndAdd:
		var op string
		if cmd.Op == ircode.OpSetAndAdd {
			op = "+="
		}
		arg := generateArgument(mod, cmd.Args[0], b)
		var left Node
		if len(cmd.AccessChain) == 0 {
			left = arg
		} else {
			left = generateAccess(mod, arg, cmd, 1, b)
		}
		argRight := cmd.Args[len(cmd.Args)-1]
		right := generateArgument(mod, argRight, b)
		statement := &Binary{Operator: op, Left: left, Right: right}
		b.Nodes = append(b.Nodes, statement)
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
			glob := &GlobalVar{Name: varName(cmd.Dest[0]), Type: mapExprType(mod, cmd.Dest[0].Type)}
			mod.Elements = append(mod.Elements, glob)
			return nil
		}
		return &Var{Name: varName(cmd.Dest[0]), Type: mapExprType(mod, cmd.Dest[0].Type)}
	case ircode.OpSetVariable:
		n = generateArgument(mod, cmd.Args[0], b)
	case ircode.OpSetGroupVariable:
		n = generateGroupVarPointer(cmd.GroupArgs[0])
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
			n = &TypeCast{Type: mapExprType(mod, cmd.Args[0].Type()), Expr: n}
		}
	case ircode.OpBinaryOr:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "|", Left: arg1, Right: arg2}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapExprType(mod, cmd.Args[0].Type()), Expr: n}
		}
	case ircode.OpBinaryAnd:
		arg1 := generateArgument(mod, cmd.Args[0], b)
		arg2 := generateArgument(mod, cmd.Args[1], b)
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			arg1 = &TypeCast{Type: &TypeDecl{Code: "uintptr_t"}, Expr: arg1}
		}
		n = &Binary{Operator: "&", Left: arg1, Right: arg2}
		if types.IsUnsafePointerType(cmd.Args[0].Type().Type) {
			n = &TypeCast{Type: mapExprType(mod, cmd.Args[0].Type()), Expr: n}
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
			n = &TypeCast{Type: mapExprType(mod, cmd.Args[0].Type()), Expr: n}
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
		if _, ok := types.GetPointerType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].Grouping)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "ptr"}}
		} else if _, ok := types.GetSliceType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].Grouping)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "slice"}}
		}
	case ircode.OpTake:
		arg := generateArgument(mod, cmd.Args[0], b)
		n = generateAccess(mod, arg, cmd, 1, b)
		t := cmd.AccessChain[len(cmd.AccessChain)-1].OutputType
		if _, ok := types.GetPointerType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].Grouping)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "ptr"}}
		} else if _, ok := types.GetSliceType(t.Type); ok && t.PointerDestGroupSpecifier != nil && t.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapExprType(mod, t), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			gv := generateGroupVar(cmd.Dest[0].Grouping)
			n = &Binary{Operator: "=", Left: gv, Right: &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "group"}}}
			b.Nodes = append(b.Nodes, n)
			n = &Binary{Operator: ".", Left: &Constant{Code: tmpVar.Name}, Right: &Constant{Code: "slice"}}
		}
		if cmd.Dest[0] != nil {
			if cmd.Dest[0].Name[0] == '%' {
				b.Nodes = append(b.Nodes, &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: n})
			} else {
				b.Nodes = append(b.Nodes, &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: n})
			}
		}
		n = nil
		left := generateAccess(mod, arg, cmd, 1, b)
		right := generateDefaultValue(mod, t)
		assign := &Binary{Operator: "=", Left: left, Right: right}
		b.Nodes = append(b.Nodes, assign)
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
			slice := &CompoundLiteral{Type: mapExprType(mod, cmd.Type), Values: []Node{callMallocWithCast, &Constant{Code: strconv.Itoa(len(cmd.Args))}, &Constant{Code: strconv.Itoa(len(cmd.Args))}}}
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
			n = &CompoundLiteral{Type: mapExprType(mod, cmd.Type), Values: args}
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
			ptr := &TypeCast{Type: mapExprType(mod, cmd.Dest[0].Type), Expr: callMallocWithCast}
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
			n = &CompoundLiteral{Type: mapExprType(mod, cmd.Type), Values: args}
		}
	case ircode.OpMalloc:
		pt, ok := types.GetPointerType(cmd.Type.Type)
		if !ok {
			panic("Oooops")
		}
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
		ptr := &TypeCast{Type: mapExprType(mod, cmd.Dest[0].Type), Expr: callMallocWithCast}
		if cmd.Dest[0].Name[0] == '%' {
			n3 = &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: ptr}
		} else {
			n3 = &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: ptr}
		}
		b.Nodes = append(b.Nodes, n3)
	case ircode.OpMallocSlice:
		sl, ok := types.GetSliceType(cmd.Type.Type)
		if !ok {
			panic("Oooops")
		}
		gv := generateAddrOfGroupVar(cmd.Dest[0])
		malloc, mallocPkg := mod.Package.GetMalloc()
		if malloc == nil {
			panic("Oooops")
		}
		// Malloc
		callMalloc := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(mallocPkg, malloc.Name)}}
		size := generateArgument(mod, cmd.Args[0], b)
		cap := generateArgument(mod, cmd.Args[1], b)
		capVarName := "cap_" + strconv.Itoa(len(b.Nodes))
		b.Nodes = append(b.Nodes, &Var{Name: capVarName, Type: NewTypeDecl("int"), InitExpr: cap})
		callMalloc.Args = []Node{&Identifier{Name: capVarName}, &Sizeof{Type: mapType(mod, sl.ElementType)}, gv}
		decl := mapSlicePointerExprType(mod, cmd.Type)
		callMallocWithCast := &TypeCast{Type: decl, Expr: callMalloc}
		// Assign to a slice pointer
		slice := &CompoundLiteral{Type: mapExprType(mod, cmd.Type), Values: []Node{callMallocWithCast, size, &Identifier{Name: capVarName}}}
		var n2 Node
		if cmd.Dest[0].Name[0] == '%' {
			n2 = &Var{Name: varName(cmd.Dest[0]), Type: mapVarExprType(mod, cmd.Dest[0].Type), InitExpr: slice}
		} else {
			n2 = &Binary{Operator: "=", Left: &Constant{Code: varName(cmd.Dest[0])}, Right: slice}
		}
		b.Nodes = append(b.Nodes, n2)
	case ircode.OpStringConcat:
		len1 := generateLen(mod, cmd.Args[0], b)
		len1VarName := "len1_" + strconv.Itoa(len(b.Nodes))
		b.Nodes = append(b.Nodes, &Var{Name: len1VarName, Type: NewTypeDecl("int"), InitExpr: len1})
		len2 := generateLen(mod, cmd.Args[1], b)
		len2VarName := "len2_" + strconv.Itoa(len(b.Nodes))
		b.Nodes = append(b.Nodes, &Var{Name: len2VarName, Type: NewTypeDecl("int"), InitExpr: len2})
		newlen := &Binary{Operator: "+", Left: &Identifier{Name: len1VarName}, Right: &Identifier{Name: len2VarName}}
		gv := generateAddrOfGroupVar(cmd.Dest[0])
		malloc, mallocPkg := mod.Package.GetMalloc()
		if malloc == nil {
			panic("Oooops")
		}
		// Malloc
		callMalloc := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(mallocPkg, malloc.Name)}}
		callMalloc.Args = []Node{newlen, &Constant{Code: "1"}, gv}
		callMallocWithCast := &TypeCast{Type: &TypeDecl{Code: "uint8_t*"}, Expr: callMalloc}
		dataVarName := "data_" + strconv.Itoa(len(b.Nodes))
		b.Nodes = append(b.Nodes, &Var{Name: dataVarName, Type: NewTypeDecl("uint8_t*"), InitExpr: callMallocWithCast})
		decl := defineString(mod)
		n = &CompoundLiteral{Type: decl, Values: []Node{newlen, &Identifier{Name: dataVarName}}}
		cpy := &FunctionCall{FuncExpr: &Constant{Code: "memcpy"}, Args: []Node{&Identifier{Name: dataVarName}, generateData(mod, cmd.Args[0], b), &Identifier{Name: len1VarName}}}
		b.Nodes = append(b.Nodes, cpy)
		cpy = &FunctionCall{FuncExpr: &Constant{Code: "memcpy"}, Args: []Node{&Binary{Operator: "+", Left: &Identifier{Name: dataVarName}, Right: &Identifier{Name: len1VarName}}, generateData(mod, cmd.Args[1], b), &Identifier{Name: len2VarName}}}
		b.Nodes = append(b.Nodes, cpy)
		mod.AddInclude("string.h", true)
	case ircode.OpLen:
		n = generateLen(mod, cmd.Args[0], b)
	case ircode.OpCap:
		n = generateCap(mod, cmd.Args[0], b)
	case ircode.OpGroupOf:
		groupOf, runtimePkg := mod.Package.GetGroupOf()
		if groupOf == nil {
			panic("Oooops")
		}
		grp := &TypeCast{Type: NewTypeDecl("uintptr_t"), Expr: generateGroupVar(cmd.GroupArgs[0])}
		call := &FunctionCall{FuncExpr: &Constant{Code: mangleFunctionName(runtimePkg, groupOf.Name)}}
		call.Args = []Node{grp}
		n = call
	case ircode.OpSizeOf:
		n = &Sizeof{Type: mapType(mod, cmd.TypeArgs[0])}
	case ircode.OpAppend:
		// How many values will be added?
		additionalSize := generateArgument(mod, cmd.Args[1], b)
		// Store a pointer to the underlying array in a new temporary variable
		ptr := &Binary{Operator: "+", Left: generateData(mod, cmd.Args[0], b), Right: generateLen(mod, cmd.Args[0], b)}
		ptrVarName := mod.tmpVarName()
		ptrVar := &Var{Name: ptrVarName, Type: mapSlicePointerExprType(mod, cmd.Args[0].Type()), InitExpr: ptr}
		b.Nodes = append(b.Nodes, ptrVar)
		// Iterate over all arguments
		for _, arg := range cmd.Args[2:] {
			// Argument is of the form `...slice` or `...str`?
			if _, ok := types.GetSliceType(arg.Type().Type); (ok || types.IsStringType(arg.Type().Type)) && arg.Flags&ircode.ArgumentIsEllipsis == ircode.ArgumentIsEllipsis {
				if arg.Const != nil {
					if types.IsStringType(arg.Type().Type) {
						// Append a string constant
						appendSize := &Constant{Code: strconv.Itoa(len(arg.Const.ExprType.StringValue))}
						cpy := &FunctionCall{FuncExpr: &Constant{Code: "memcpy"}, Args: []Node{ptr, &Constant{Code: strconv.QuoteToASCII(arg.Const.ExprType.StringValue)}, appendSize}}
						b.Nodes = append(b.Nodes, cpy)
						// Forward `ptr` by the number of bytes appended
						inc := &Binary{Operator: "+=", Left: &Identifier{Name: ptrVarName}, Right: appendSize}
						b.Nodes = append(b.Nodes, inc)
						mod.AddInclude("string.h", true)
					} else {
						// Append a slice constant
						for j := 0; j < len(arg.Const.ExprType.ArrayValue); j++ {
							right := &Constant{Code: constToString(mod, arg.Const.ExprType.ArrayValue[j])}
							left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: ptrVarName}}}
							assign := &Binary{Operator: "=", Left: left, Right: right}
							b.Nodes = append(b.Nodes, assign)
						}
					}
				} else {
					// How many values does the source slice have?
					sizeVarName := mod.tmpVarName()
					sizeVar := &Var{Name: sizeVarName, Type: mapType(mod, types.PrimitiveTypeInt), InitExpr: generateLen(mod, arg, b)}
					b.Nodes = append(b.Nodes, sizeVar)
					// Iterate over all values in the slice and append the values
					loopVarName := mod.tmpVarName()
					loopVar := &Var{Name: loopVarName, Type: mapType(mod, types.PrimitiveTypeInt), InitExpr: &Constant{Code: "0"}}
					loopCond := &Binary{Operator: "<", Left: &Identifier{Name: loopVarName}, Right: &Identifier{Name: sizeVarName}}
					loopExpr := &Unary{Operator: "++", Expr: &Identifier{Name: loopVarName}}
					loop := &For{InitExpr: loopVar, ConditionExpr: loopCond, LoopExpr: loopExpr}
					// Get a pointer to the underlying array and append a value
					valPtr := generateData(mod, arg, b)
					right := &Binary{Operator: "[", Left: valPtr, Right: &Identifier{Name: loopVarName}}
					left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: ptrVarName}}}
					assign := &Binary{Operator: "=", Left: left, Right: right}
					loop.Body = append(loop.Body, assign)
					b.Nodes = append(b.Nodes, loop)
				}
			} else if at, ok := types.GetArrayType(arg.Type().Type); ok && arg.Flags&ircode.ArgumentIsEllipsis == ircode.ArgumentIsEllipsis {
				// Argument is of form `...array`.
				if arg.Const != nil {
					for j := 0; j < len(arg.Const.ExprType.ArrayValue); j++ {
						right := &Constant{Code: constToString(mod, arg.Const.ExprType.ArrayValue[j])}
						left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: ptrVarName}}}
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
					left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: ptrVarName}}}
					assign := &Binary{Operator: "=", Left: left, Right: right}
					loop.Body = append(loop.Body, assign)
					b.Nodes = append(b.Nodes, loop)
				}
			} else {
				// Argument is a single value
				val := generateArgument(mod, arg, b)
				left := &Unary{Operator: "*", Expr: &Unary{Operator: "++", Expr: &Identifier{Name: ptrVarName}}}
				assign := &Binary{Operator: "=", Left: left, Right: val}
				b.Nodes = append(b.Nodes, assign)
			}
		}
		slicePtr := generateData(mod, cmd.Args[0], b)
		capacity := generateCap(mod, cmd.Args[0], b)
		size := generateLen(mod, cmd.Args[0], b)
		size = &Binary{Operator: "+", Left: size, Right: additionalSize}
		n = &CompoundLiteral{Type: mapExprType(mod, cmd.Type), Values: []Node{slicePtr, size, capacity}}
	case ircode.OpCall:
		ft, ok := types.GetFuncType(cmd.Args[0].Type().Type)
		if !ok {
			panic("Ooooops")
		}
		fexpr := generateArgument(mod, cmd.Args[0], b)
		irft := ircode.NewFunctionType(ft)
		var args []Node
		argIndex := 1
		for range irft.In {
			arg := generateArgument(mod, cmd.Args[argIndex], b)
			argIndex++
			args = append(args, arg)
		}
		for i := range irft.GroupSpecifiers {
			gv := cmd.GroupArgs[i]
			arg := generateGroupVarPointer(gv)
			args = append(args, arg)
		}
		n = &FunctionCall{FuncExpr: fexpr, Args: args}
		if len(cmd.Dest) > 1 {
			rt := irft.ReturnType()
			rts, ok := types.GetStructType(rt)
			if !ok {
				panic("Oooops")
			}
			tmpVar := &Var{Name: mod.tmpVarName(), Type: mapType(mod, rt), InitExpr: n}
			b.Nodes = append(b.Nodes, tmpVar)
			for i, d := range cmd.Dest {
				val := &Binary{Operator: ".", Left: &Identifier{Name: tmpVar.Name}, Right: &Identifier{Name: rts.Fields[i].Name}}
				if d.Name[0] == '%' {
					n = &Var{Name: varName(d), Type: mapVarExprType(mod, d.Type), InitExpr: val}
					b.Nodes = append(b.Nodes, n)
				} else {
					n = &Binary{Operator: "=", Left: &Constant{Code: varName(d)}, Right: val}
					b.Nodes = append(b.Nodes, n)
				}
			}
			return nil
		}
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
		panic("Oooops")
	}
	if _, ok := types.GetSliceType(arg.Var.Type.Type); ok {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "size"}}
	}
	if arg.Var.Type.Type == types.PrimitiveTypeString {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "size"}}
	}
	panic("Oooops")
}

func generateLenFromNode(mod *Module, n Node, t types.Type, b *CBlockBuilder) Node {
	if _, ok := types.GetSliceType(t); ok {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "size"}}
	}
	if t == types.PrimitiveTypeString {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "size"}}
	}
	panic("Oooops")
}

func generateCap(mod *Module, arg ircode.Argument, b *CBlockBuilder) Node {
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
		panic("Oooops")
	}
	if _, ok := types.GetSliceType(arg.Var.Type.Type); ok {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "cap"}}
	}
	if arg.Var.Type.Type == types.PrimitiveTypeString {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "size"}}
	}
	panic("Oooops")
}

func generateCapFromNode(mod *Module, n Node, t types.Type, b *CBlockBuilder) Node {
	if _, ok := types.GetSliceType(t); ok {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "cap"}}
	}
	if t == types.PrimitiveTypeString {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "size"}}
	}
	panic("Oooops")
}

func generateData(mod *Module, arg ircode.Argument, b *CBlockBuilder) Node {
	if arg.Const != nil {
		if _, ok := types.GetSliceType(arg.Const.ExprType.Type); ok {
			if arg.Const.ExprType.IntegerValue.Uint64() != 0 {
				panic("Oooops, should only be possible for null pointers")
			}
			return &Constant{Code: "0"}
		}
		if arg.Const.ExprType.Type == types.PrimitiveTypeString {
			return &Constant{Code: strconv.QuoteToASCII(arg.Const.ExprType.StringValue)}
		}
		panic("Oooops")
	}
	if _, ok := types.GetSliceType(arg.Var.Type.Type); ok {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "ptr"}}
	}
	if arg.Var.Type.Type == types.PrimitiveTypeString {
		left := generateArgument(mod, arg, b)
		return &Binary{Operator: ".", Left: left, Right: &Identifier{Name: "data"}}
	}
	panic("Oooops")
}

func generateDataFromNode(mod *Module, n Node, t types.Type, b *CBlockBuilder) Node {
	if _, ok := types.GetSliceType(t); ok {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "ptr"}}
	}
	if t == types.PrimitiveTypeString {
		return &Binary{Operator: ".", Left: n, Right: &Identifier{Name: "data"}}
	}
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
				mod.AddInclude("stdlib.h", true)
				test.Body = append(test.Body, &Identifier{Name: "exit(1)"})
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
			var exprVarName = mod.tmpVarName()
			exprVar := &Var{Name: exprVarName, Type: mapExprType(mod, a.InputType), InitExpr: expr}
			b.Nodes = append(b.Nodes, exprVar)
			var idx1 Node
			hasIdx1 := false
			if cmd.Args[argIndex].Flags == ircode.ArgumentIsMissing {
				idx1 = &Constant{Code: "0"}
			} else {
				idx1 = generateArgument(mod, cmd.Args[argIndex], b)
				capacity := generateCapFromNode(mod, &Identifier{Name: exprVarName}, a.OutputType.Type, b)
				cmp := &Binary{Operator: ">=", Left: &TypeCast{Type: &TypeDecl{Code: "unsigned int"}, Expr: idx1}, Right: capacity}
				rangeCheck := &If{Expr: cmp}
				b.Nodes = append(b.Nodes, rangeCheck)
				mod.AddInclude("stdlib.h", true)
				rangeCheck.Body = append(rangeCheck.Body, &Identifier{Name: "exit(1)"})
				hasIdx1 = true
			}
			argIndex++
			var idx2 Node
			if cmd.Args[argIndex].Flags == ircode.ArgumentIsMissing {
				idx2 = generateLenFromNode(mod, &Identifier{Name: exprVarName}, a.OutputType.Type, b)
			} else {
				idx2 = generateArgument(mod, cmd.Args[argIndex], b)
				capacity := generateCapFromNode(mod, &Identifier{Name: exprVarName}, a.OutputType.Type, b)
				cmp1 := &Binary{Operator: ">", Left: &TypeCast{Type: &TypeDecl{Code: "unsigned int"}, Expr: idx2}, Right: capacity}
				var cmp Node
				if hasIdx1 {
					cmp2 := &Binary{Operator: ">", Left: idx1, Right: idx2}
					cmp = &Binary{Operator: "||", Left: cmp1, Right: cmp2}
				} else {
					cmp = cmp1
				}
				rangeCheck := &If{Expr: cmp}
				b.Nodes = append(b.Nodes, rangeCheck)
				mod.AddInclude("stdlib.h", true)
				rangeCheck.Body = append(rangeCheck.Body, &Identifier{Name: "exit(1)"})
			}
			argIndex++
			var newSize Node
			var newCapacity Node
			newPtr := generateDataFromNode(mod, &Identifier{Name: exprVarName}, a.InputType.Type, b)
			if hasIdx1 {
				// `[index1:index2]` or `[index1:]`
				newSize = &Binary{Operator: "-", Left: idx2, Right: idx1}
				newCapacity = &Binary{Operator: "-", Left: generateCapFromNode(mod, &Identifier{Name: exprVarName}, a.InputType.Type, b), Right: idx1}
				newPtr = &Binary{Operator: "+", Left: newPtr, Right: idx1}
			} else {
				// `[:index2]` or `[:]`
				newSize = idx2
				newCapacity = generateCapFromNode(mod, expr, a.OutputType.Type, b)
			}
			expr = &CompoundLiteral{Type: mapExprType(mod, a.OutputType), Values: []Node{newPtr, newSize, newCapacity}}
		case ircode.AccessSliceIndex:
			idx := generateArgument(mod, cmd.Args[argIndex], b)
			argIndex++
			// TODO: Check boundary
			expr = &Binary{Left: &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "ptr"}}, Right: idx, Operator: "["}
		case ircode.AccessCast:
			et := a.OutputType
			switch et.TypeConversionValue {
			case types.ConvertStringToPointer:
				expr = &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "data"}}
			case types.ConvertPointerToPointer:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertSliceToPointer:
				expr = &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "ptr"}}
			case types.ConvertIntegerToPointer:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertPointerToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertPointerToSlice:
				st, ok := types.GetSliceType(a.OutputType.Type)
				if !ok {
					panic("Ooooops")
				}
				t := defineSliceType(mod, st, a.OutputType.PointerDestGroupSpecifier, a.OutputType.PointerDestMutable, a.OutputType.PointerDestVolatile)
				expr = &CompoundLiteral{Type: t, Values: []Node{expr, &Constant{Code: "INT_MAX"}, &Constant{Code: "INT_MAX"}}}
				mod.AddInclude("limits.h", true)
			case types.ConvertStringToByteSlice:
				st, ok := types.GetSliceType(a.OutputType.Type)
				if !ok {
					panic("Ooooops")
				}
				t := defineSliceType(mod, st, a.OutputType.PointerDestGroupSpecifier, a.OutputType.PointerDestMutable, a.OutputType.PointerDestVolatile)
				data := &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "data"}}
				size := &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "size"}}
				sl := &CompoundLiteral{Type: t, Values: []Node{data, size, size}}
				expr = &TypeCast{Expr: sl, Type: t}
			case types.ConvertPointerToString:
				exprVarName := mod.tmpVarName()
				exprVar := &Var{Name: exprVarName, Type: &TypeDecl{Code: "uint8_t*"}, InitExpr: expr}
				b.Nodes = append(b.Nodes, exprVar)
				constChar := &TypeCast{Type: &TypeDecl{Code: "const char*"}, Expr: &Identifier{Name: exprVarName}}
				strlen := &FunctionCall{FuncExpr: &Constant{Code: "strlen"}, Args: []Node{constChar}}
				t := defineString(mod)
				expr = &CompoundLiteral{Type: t, Values: []Node{strlen, &Identifier{Name: exprVarName}}}
				mod.AddInclude("string.h", true)
			case types.ConvertByteSliceToString:
				t := defineString(mod)
				data := &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "ptr"}}
				size := &Binary{Operator: ".", Left: expr, Right: &Identifier{Name: "size"}}
				expr = &CompoundLiteral{Type: t, Values: []Node{size, data}}
			case types.ConvertIntegerToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertFloatToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertBoolToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertRuneToInteger:
				expr = &TypeCast{Expr: expr, Type: mapType(mod, a.OutputType.Type)}
			case types.ConvertIntegerToFloat:
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
		// str := mod.AddString(et.StringValue)
		t := defineString(mod)
		str := strconv.QuoteToASCII(et.StringValue)
		return "(" + t.Code + "){" + strconv.Itoa(len(et.StringValue)) + ", (uint8_t*)" + str + "}"
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
		if et.PointerDestGroupSpecifier != nil && et.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
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
			if et.PointerDestGroupSpecifier != nil && et.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
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
	if _, ok := types.GetUnionType(et.Type); ok {
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

func generateDefaultValue(mod *Module, et *types.ExprType) Node {
	return &Constant{Code: defaultValueToString(mod, et)}
}

func defaultValueToString(mod *Module, et *types.ExprType) string {
	if types.IsUnsignedIntegerType(et.Type) {
		// TODO: The correctness of this depends on the target platform
		if et.Type == types.PrimitiveTypeUint64 || et.Type == types.PrimitiveTypeUintptr {
			return "0ull"
		}
		return "0u"
	}
	if types.IsIntegerType(et.Type) {
		// TODO: The correctness of this depends on the target platform
		if et.Type == types.PrimitiveTypeInt64 {
			return "0ll"
		}
		return "0"
	}
	if types.IsFloatType(et.Type) {
		return "0"
	}
	if et.Type == types.PrimitiveTypeRune {
		return "0"
	}
	if et.Type == types.PrimitiveTypeString {
		// str := mod.AddString(et.StringValue)
		t := defineString(mod)
		return "(" + t.Code + "){0, (uint8_t*)0}"
	}
	if et.Type == types.PrimitiveTypeBool {
		return "false"
	}
	if types.IsSliceType(et.Type) {
		if et.PointerDestGroupSpecifier != nil && et.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
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
			str += defaultValueToString(mod, element)
		}
		// Empty initializer lists are not allowed in C
		if len(et.ArrayValue) == 0 {
			str += "0"
		}
		return str + "}"
	}
	if _, ok := types.GetPointerType(et.Type); ok {
		if et.PointerDestGroupSpecifier != nil && et.PointerDestGroupSpecifier.Kind == types.GroupSpecifierIsolate {
			return "(" + mapExprType(mod, et).ToString("") + "){0, 0}"
		}
		return "((" + mapExprType(mod, et).ToString("") + ")0)"
	}
	if _, ok := types.GetStructType(et.Type); ok {
		str := "(" + mapExprType(mod, et).ToString("") + "){"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += "." + name + "=(" + defaultValueToString(mod, element) + ")"
			i++
		}
		return str + "}"
	}
	if _, ok := types.GetUnionType(et.Type); ok {
		str := "(" + mapExprType(mod, et).ToString("") + "){"
		i := 0
		for name, element := range et.StructValue {
			if i > 0 {
				str += ", "
			}
			str += "." + name + "=(" + defaultValueToString(mod, element) + ")"
			i++
		}
		return str + "}"
	}
	if _, ok := types.GetFuncType(et.Type); ok {
		return "0"
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

func generateGroupVar(group ircode.IGrouping) Node {
	if group.IsConstant() {
		return &Constant{Code: "0"}
	}
	gv := group.GroupVariable()
	if gv == nil {
		println("NO VAR FOR " + group.GroupingName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Unary{Operator: "*", Expr: &Constant{Code: varName(gv)}}
	}
	return &Constant{Code: varName(gv)}
}

func generateAddrOfGroupVar(v *ircode.Variable) Node {
	if v.Grouping == nil {
		panic("Ooooops")
	}
	if v.Grouping.IsConstant() {
		return &Constant{Code: "&g_zero"}
	}
	gv := v.Grouping.GroupVariable()
	if gv == nil {
		println("NO VAR FOR " + v.Grouping.GroupingName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Constant{Code: varName(gv)}
	}
	return &Unary{Operator: "&", Expr: &Constant{Code: varName(gv)}}
}

func generateGroupVarPointer(group ircode.IGrouping) Node {
	if group.IsConstant() {
		return &Constant{Code: "&g_zero"}
	}
	gv := group.GroupVariable()
	if gv == nil {
		println("NO VAR FOR " + group.GroupingName())
		panic("Oooops")
	}
	if _, ok := types.GetPointerType(gv.Type.Type); ok {
		return &Constant{Code: varName(gv)}
	}
	return &Unary{Operator: "&", Expr: &Constant{Code: varName(gv)}}
}
