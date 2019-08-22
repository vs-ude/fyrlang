package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func genFunc(f *types.Func) *ircode.Function {
	println("GEN FUNC ", f.Name())
	// TODO: If it is a member function, add the `this` parameter to the function signature
	b := ircode.NewBuilder(mangleFunctionName(f), f.Type)
	genBody(f.Ast.Body, f.InnerScope, b)
	b.Finalize()
	return b.Func
}

func genBody(ast *parser.BodyNode, s *types.Scope, b *ircode.Builder) {
	vars := make(map[*types.Variable]*ircode.Variable)
	for _, ch := range ast.Children {
		genStatement(ch, s, b, vars)
	}
}

func mangleFunctionName(f *types.Func) string {
	// TODO
	return f.Name()
}
