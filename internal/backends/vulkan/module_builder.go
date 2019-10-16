package vulkan

import (
	// "github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/irgen"
	// "github.com/vs-ude/fyrlang/internal/types"

	. "github.com/andreas-jonsson/spirv"
)

type ModuleBuilder struct {
	Package *irgen.Package
	// File header
	Header Header
	// Top
	Capabilities      []Capability
	Extensions        []string
	ExtInstImports    InstructionList
	AddressingModel   AddressingModel
	MemoryModel       MemoryModel
	ExecutionModel    ExecutionModel
	EntryPoint        Id // We only expect one entry point
	ExecutionMode     ExecutionMode
	ExecutionModeArgv []uint32
	// Debug
	Debug InstructionList
	// Head
	Types     InstructionList
	Constants InstructionList
	Globals   InstructionList
	// Body
	Functions []InstructionList
}

// NewModule creates a new, default module.
func NewModule(p *irgen.Package) *ModuleBuilder {
	return &ModuleBuilder{
		Package: p,
		Header: Header{
			Magic:          MagicLE,
			Version:        SpecificationVersion,
			GeneratorMagic: 0,
			Bound:          1,
			Reserved:       0,
		},
	}
}

// Retrieve a new unused ResultId for the module and adjust the bound.
func (m *ModuleBuilder) NewResultId() (id Id) {
	id = m.Header.Bound
	m.Header.Bound += 1
	return
}

func (m *ModuleBuilder) AddCapability(cap Capability) {
	m.Capabilities = append(m.Capabilities, cap)
}

func (m *ModuleBuilder) AddExtension(ext string) {
	m.Extensions = append(m.Extensions, ext)
}

func (m *ModuleBuilder) AddExtInstImport(instr Instruction) {
	m.ExtInstImports = append(m.ExtInstImports, instr)
}

func (m *ModuleBuilder) addType(instr Instruction) {
	m.Types = append(m.Types, instr)
}

// Look through the already registered types and either returns its ResultId or adds a new instruction.
func (m *ModuleBuilder) EnsureType(ref_instr Instruction) (id Id) {
	for _, instr := range m.Types.Filter(ref_instr.Opcode()) {
		if InstructionEquals(instr, ref_instr, false) {
			id, ok := InstructionResultId(instr)
			if ok != true {
				panic("type should have ResultId")
			}
			return id
		}
	}
	id = m.NewResultId()
	SetInstructionResultId(&ref_instr, id)
	m.addType(ref_instr)
	return
}

func (m *ModuleBuilder) addConstant(instr Instruction) {
	m.Constants = append(m.Constants, instr)
}

// Look through the already registered constants and either returns its ResultId or adds a new instruction.
func (m *ModuleBuilder) EnsureConstant(ref_instr Instruction) (id Id) {
	for _, instr := range m.Constants.Filter(ref_instr.Opcode()) {
		if InstructionEquals(instr, ref_instr, false) {
			id, ok := InstructionResultId(instr)
			if ok != true {
				panic("type should have ResultId")
			}
			return id
		}
	}
	id = m.NewResultId()
	SetInstructionResultId(&ref_instr, id)
	m.addConstant(ref_instr)
	return
}

func addInstr(smod *Module, instr Instruction) {
	smod.Code = append(smod.Code, instr)
}

func addInstrs(smod *Module, instrs InstructionList) {
	smod.Code = append(smod.Code, instrs...)
}

func (m *ModuleBuilder) BuildSpirvModule() *Module {
	smod := Module{Header: m.Header}

	for _, cap := range m.Capabilities {
		addInstr(&smod, &OpCapability{cap})
	}
	for _, s := range m.Extensions {
		addInstr(&smod, &OpExtension{String(s)})
	}
	addInstrs(&smod, m.ExtInstImports)
	addInstr(&smod, &OpMemoryModel{m.AddressingModel, m.MemoryModel})
	addInstr(&smod, &OpEntryPoint{m.ExecutionModel, m.EntryPoint, "Main", []Id{}})
	addInstr(&smod, &OpExecutionMode{m.EntryPoint, m.ExecutionMode, m.ExecutionModeArgv})

	addInstrs(&smod, m.Debug)
	addInstrs(&smod, m.Types)
	addInstrs(&smod, m.Constants)
	addInstrs(&smod, m.Globals)

	for _, fun := range m.Functions {
		addInstrs(&smod, fun)
	}

	return &smod
}
