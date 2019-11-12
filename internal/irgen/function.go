package irgen

import (
	"crypto/sha256"
	"encoding/hex"
	"unicode"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/ssa"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genFunc(p *Package, f *types.Func, globalVars map[*types.Variable]*ircode.Variable, log *errlog.ErrorLog) *ircode.Function {
	println("GEN FUNC ", f.Name())
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
		v := f.InnerScope.GetVariable(p.Name)
		b.SetLocation(p.Location)
		irv := b.DefineVariable(p.Name, v.Type)
		irv.Kind = ircode.VarParameter
		vars[v] = irv
	}
	// Generate an IR-code variable for all named return parameters
	for _, p := range ft.Out {
		if p.Name == "" {
			continue
		}
		v := f.InnerScope.GetVariable(p.Name)
		b.SetLocation(p.Location)
		vars[v] = b.DefineVariable(p.Name, v.Type)
	}
	// Generate an IR-code variable for all group parameters
	groupVars := make(map[*types.Group]*ircode.Variable)
	for _, g := range ft.GroupParameters {
		b.SetLocation(g.Location)
		irv := b.DefineVariable(g.Name, &types.ExprType{Type: &types.PointerType{Mode: types.PtrUnsafe, ElementType: types.PrimitiveTypeUintptr}})
		irv.Kind = ircode.VarGroupParameter
		groupVars[g] = irv
	}
	// Generate an IR-code variable for all global variables accessible to the function
	var globalVarsList []*ircode.Variable
	for v, irv := range globalVars {
		vars[v] = irv
		globalVarsList = append(globalVarsList, irv)
	}
	// Generate IR-code for the function implementation
	genBody(f.Ast.Body, f.InnerScope, b, p, vars)
	b.Finalize()
	// Attach group information
	// println(irf.ToString())
	ssa.TransformToSSA(irf, groupVars, globalVarsList, log)
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
	if f.Type.Target != nil {
		str = f.Type.Target.ToString() + "::"
	}
	if !f.IsGenericMemberFunc() {
		if str == "" {
			return f.Name()
		}
		sum := sha256.Sum256([]byte(str))
		sumHex := hex.EncodeToString(sum[:])
		println(f.Name(), str)
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
		println(f.Name(), str)
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
