package c99

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Generates a C-AST function from an IR function
func generateFunction(mod *Module, p *irgen.Package, irf *ircode.Function) *Function {
	f := &Function{Name: mangleFunctionName(p, irf.Name)}
	f.Body = generateCommand(&irf.Body, nil)
	//	if len(irf.Type.Out.Params) == 0 {
	f.ReturnType = NewTypeDecl("void")
	//	}
	f.IsGenericInstance = irf.IsGenericInstance
	f.IsExported = irf.IsExported
	return f
}

func generateCommand(cmd *ircode.Command, result []Node) []Node {
	switch cmd.Op {
	case ircode.OpBlock:
		for _, c := range cmd.Block {
			result = generateCommand(c, result)
		}
		return result
	case ircode.OpDefVariable:
		if cmd.Dest[0].Var.Kind == ircode.VarParameter {
			return result
		}
	}
	return result
	//	panic("Ooooops")
}

func mangleFunctionName(p *irgen.Package, name string) string {
	data := p.TypePackage.FullPath() + "//" + name
	sum := sha256.Sum256([]byte(data))
	sumHex := hex.EncodeToString(sum[:])
	return name + "_" + sumHex
}
