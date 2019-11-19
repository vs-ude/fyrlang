package vulkan

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	// "github.com/vs-ude/fyrlang/internal/irgen"
	// "github.com/vs-ude/fyrlang/internal/types"

	. "github.com/vs-ude/spirv"
)

type Function struct {
	name      string
	ResultId  Id
	signature InstructionList
	variables InstructionList
	blocks    []InstructionList
}

func (mod *ModuleBuilder) buildFunction(irf *ircode.Function) *Function {
	// func buildFunction(mod *ModuleBuilder, irf *ircode.Function) *Function {
	f := Function{name: irf.Name, ResultId: mod.NewResultId()}
	// TODO implement
	return &f
}

func (f *Function) Assemble() {
	// TODO implement
}
