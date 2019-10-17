package vulkan

import (
	"os"
	"path/filepath"
	// "github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/irgen"
	// "github.com/vs-ude/fyrlang/internal/types"

	"github.com/andreas-jonsson/spirv"
)

func CreateShader(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	println("MAKE DIR", pkgPath)
	if err := os.MkdirAll(pkgPath, 0700); err != nil {
		return err
	}

	mod := NewModule(p)

	// Define top
	mod.AddCapability(spirv.CapabilityShader)
	mod.AddCapability(spirv.CapabilityVariablePointers)
	mod.AddExtension("SPV_KHR_variable_pointers")
	mod.AddressingModel = spirv.AddressingModelLogical
	mod.MemoryModel = spirv.MemoryModelGLSL450
	mod.ExecutionModel = spirv.ExecutionModelGLCompute
	mod.ExecutionMode = spirv.ExecutionModeLocalSize
	mod.ExecutionModeArgv = []uint32{1, 1, 1} // TODO read this from somewhere
	// TODO process imports (ignore them all for now)
	// for _, irImport := range p.Imports {}
	// TODO map imported functions to OpExtInst, when reasonable, and add OpExtInstImport instructions

	// We could add debug instructions here, like OpSource, but skip those for now

	// TODO through mod.EnsureType we declare types as we need them

	// Define globals in head
	initIrf := p.Funcs[p.TypePackage.InitFunc]
	// TODO parse constant assignments and defer calculations to main func, if possible

	// Build functions for body and add types and constants to head
	for _, irf := range p.Funcs {
		if irf == initIrf {
			continue
		}
		fun := mod.buildFunction(irf)
		if irf == p.MainFunc {
			mod.EntryPoint = fun.ResultId
		}
	}

	// Finalize the module
	smod := mod.BuildSpirvModule()
	if err := smod.Verify(); err != nil {
		return err
	}

	// Save it
	basename := filepath.Base(p.TypePackage.FullPath())
	targetPath := filepath.Join(pkgPath, basename+".spv")
	// TODO use 0666? 0600 would be a result of the umask, if desired
	// => same as os.Create
	fd, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer fd.Close()
	err = smod.Save(fd)
	if err != nil {
		return err
	}
	return nil
}

func pkgOutputPath(p *irgen.Package) string {
	if p.TypePackage.IsInFyrPath() {
		return filepath.Join(p.TypePackage.RepoPath, "pkg", p.TypePackage.Path)
	}
	return filepath.Join(p.TypePackage.FullPath(), "pkg")
}
