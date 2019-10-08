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
	// TODO: If it is a member function, add the `this` parameter to the function signature
	b := ircode.NewBuilder(irf)
	b.SetLocation(f.Location)
	vars := make(map[*types.Variable]*ircode.Variable)
	for _, p := range f.Type.In.Params {
		v := f.InnerScope.GetVariable(p.Name)
		b.SetLocation(p.Location)
		irv := b.DefineVariable(p.Name, v.Type)
		irv.Kind = ircode.VarParameter
		vars[v] = irv
	}
	for _, p := range f.Type.Out.Params {
		if p.Name == "" {
			continue
		}
		v := f.InnerScope.GetVariable(p.Name)
		b.SetLocation(p.Location)
		vars[v] = b.DefineVariable(p.Name, v.Type)
	}
	var globalVarsList []*ircode.Variable
	for v, irv := range globalVars {
		vars[v] = irv
		globalVarsList = append(globalVarsList, irv)
	}
	genBody(f.Ast.Body, f.InnerScope, b, p, vars)
	b.Finalize()
	ssa.TransformToSSA(b.Func, globalVarsList, log)
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
	if f.Target != nil {
		str = f.Target.ToString() + "::"
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

func isUpperCaseName(name string) bool {
	for _, r := range name {
		if unicode.ToUpper(r) == r {
			return true
		}
		return false
	}
	return false
}
