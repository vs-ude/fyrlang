package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/parser"
	"github.com/vs-ude/fyrlang/internal/types"
)

func transformFunc(f *types.Func) *ircode.Function {
	println("GROUP CHECK FUNC", f.Name())
	b := ircode.NewBuilder(mangleFunctionName(f), f.Type)
	transformBody(f.Ast.Body, f.InnerScope, b)
	b.Finalize()
	return b.Func
}

func transformBody(ast *parser.BodyNode, s *types.Scope, b *ircode.Builder) {
	for _, ch := range ast.Children {
		transformStatement(ch, s, b)
	}
}

func mangleFunctionName(f *types.Func) string {
	// TODO
	return f.Name()
}
