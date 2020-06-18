package irgen

import (
	"crypto/sha256"
	"encoding/hex"
	"unicode"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/ssa"
	"github.com/vs-ude/fyrlang/internal/types"
)

// genFunc populates an ircode.Function from a type-checked function.
// The ircode.Function has been created before.
func genFunc(p *Package, f *types.Func, globalVars map[*types.Variable]*ircode.Variable, globalGrouping *ssa.Grouping, log *errlog.ErrorLog) *ircode.Function {
	if config.Verbose() {
		println("GEN FUNC ", f.Name())
	}
	irf := p.Funcs[f]
	irf.IsGenericInstance = f.IsGenericInstanceMemberFunc() || f.IsGenericInstanceFunc()
	irf.IsExported = isUpperCaseName(f.Name()) || f.IsExported
	irf.IsExtern = f.IsExtern
	b := ircode.NewBuilder(irf)
	b.SetLocation(f.Location)
	vars := make(map[*types.Variable]*ircode.Variable)
	ft := irf.Type()
	// Generate an IR-code variable for all in parameters
	for _, p := range ft.In {
		b.SetLocation(p.Location)
		irv := b.DefineVariable(p.Name, types.NewExprType(p.Type))
		irv.Kind = ircode.VarParameter
		irf.InVars = append(irf.InVars, irv)
		if !p.IsGenerated {
			v := f.InnerScope.GetVariable(p.Name)
			vars[v] = irv
		}
	}
	// Generate an IR-code variable for all named return parameters
	for _, p := range ft.Out {
		if p.Name == "" {
			continue
		}
		v := f.InnerScope.GetVariable(p.Name)
		b.SetLocation(p.Location)
		irv := b.DefineVariable(p.Name, v.Type)
		vars[v] = irv
		irf.OutVars = append(irf.OutVars, irv)
	}
	if f.Type.IsDestructor {
		irv := b.DefineVariable("this", types.NewExprType(f.Type.Target))
		irv.Kind = ircode.VarDefault
		irf.InVars = append(irf.InVars, irv)
		// Cast the parameter __this__ to the required pointer type and store it in the variable `this`.
		ab := b.Get(irv, ircode.NewVarArg(irf.InVars[0]))
		et := types.NewExprType(f.Type.Target)
		et.TypeConversionValue = types.ConvertPointerToPointer
		ab.Cast(et)
		ab.GetValue()
		v := f.InnerScope.GetVariable("this")
		vars[v] = irv
		if pt, ok := types.GetPointerType(f.Type.Target); ok {
			if st, ok := types.GetStructType(pt.ElementType); ok {
				// Generate code to free all fields of the struct
				// TODO: BaseType
				for _, f := range st.Fields {
					if _, ok := types.GetPointerType(f.Type); ok && types.NeedsDestructor(f.Type) {
						ab := b.GetForeignGroup(nil, ircode.NewVarArg(irv))
						ab.PointerStructField(f, types.NewExprType(f.Type))
						gval := ab.GetValue()
						b.Free(ircode.NewVarArg(gval))
					} else if _, ok := types.GetPointerType(f.Type); ok && types.NeedsDestructor(f.Type) {
						// TODO: Slice
					}
				}
			}
		}
		/*
			// Move all generated code from the function body intp the closeScopeCommand.
			closeScopeCommand := irf.Body.Block[len(irf.Body.Block)-1]
			if closeScopeCommand.Op != ircode.OpCloseScope {
				panic("Oooops")
			}
			closeScopeCommand.Block = append(closeScopeCommand.Block, irf.Body.Block[1:len(irf.Body.Block)-1]...)
			irf.Body.Block = []*ircode.Command{irf.Body.Block[0], closeScopeCommand}
		*/
	}
	// Generate an IR-code variable for all groups mentioned in the function's parameters.
	// These parameters hold group pointers which are passed to the function upon invocation.
	parameterGroupVars := make(map[*types.GroupSpecifier]*ircode.Variable)
	for _, g := range ft.GroupSpecifiers {
		b.SetLocation(g.Location)
		irv := b.DefineVariable(g.Name, &types.ExprType{Type: &types.PointerType{Mode: types.PtrUnsafe, ElementType: types.PrimitiveTypeUintptr}})
		irv.Kind = ircode.VarGroupParameter
		parameterGroupVars[g] = irv
	}
	// Make all global variables accessible to the function and compile
	// a list of all global variables.
	var globalVarsList []*ircode.Variable
	for v, irv := range globalVars {
		vars[v] = irv
		globalVarsList = append(globalVarsList, irv)
	}
	// Generate IR-code for the function body.
	if f.Ast != nil {
		// Generated destructors have no AST. Hence the check here.
		genBody(f.Ast.Body, f.InnerScope, b, p, vars)
	}
	b.Finalize()
	// Attach group information and transform to SSA .
	init := p.Funcs[p.TypePackage.InitFunc]
	ssa.TransformToSSA(init, irf, parameterGroupVars, globalVarsList, globalGrouping, log)
	return irf
}

func genBody(ast *parser.BodyNode, s *types.Scope, b *ircode.Builder, p *Package, vars map[*types.Variable]*ircode.Variable) {
	if s2, ok := ast.Scope().(*types.Scope); ok {
		s = s2
	}
	for _, ch := range ast.Children {
		genStatement(ch, s, b, p, vars)
	}
}

// mangleFunctionName returns a name for this function that is unique inside this package.
// For normal functions (no generic, no member functions) mangleFunctionName
// does not change the name at all.
func mangleFunctionName(f *types.Func) string {
	str := ""
	if f.Component != nil {
		str += f.Component.Name() + ":::"
	}
	if f.Type.Target != nil {
		str += f.Type.Target.ToString() + "::"
	}
	if !f.IsGenericMemberFunc() {
		if str == "" {
			return f.Name()
		}
		sum := sha256.Sum256([]byte(str))
		sumHex := hex.EncodeToString(sum[:])
		if config.Verbose() {
			println(f.Name(), str)
		}
		return f.Name() + "_" + sumHex
	}
	for _, t := range f.TypeArguments {
		str += ","
		str += t.ToString()
	}
	sum := sha256.Sum256([]byte(str))
	sumHex := hex.EncodeToString(sum[:])
	return f.Name() + "_" + sumHex
}

func mangleDualFunctionName(f *types.Func) string {
	str := "dual$$"
	if f.Type.Target != nil {
		str += f.Type.Target.ToString() + "::"
	}
	if !f.IsGenericMemberFunc() {
		if str == "dual$$" {
			return f.Name()
		}
		sum := sha256.Sum256([]byte(str))
		sumHex := hex.EncodeToString(sum[:])
		if config.Verbose() {
			println(f.Name(), str)
		}
		return f.Name() + "_" + sumHex
	}
	for _, t := range f.TypeArguments {
		str += ","
		str += t.ToString()
	}
	sum := sha256.Sum256([]byte(str))
	sumHex := hex.EncodeToString(sum[:])
	return f.Name() + "_" + sumHex
}

func isUpperCaseName(name string) bool {
	for _, r := range name {
		if unicode.ToUpper(r) == r {
			return true
		}
		return false
	}
	return false
}
