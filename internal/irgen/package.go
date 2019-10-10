package irgen

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/ssa"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Package is the IR-code representation of a Fyr package.
type Package struct {
	TypePackage *types.Package
	Funcs       map[*types.Func]*ircode.Function
	Imports     map[*types.Package]*Package
	// May be null if no main function is defined
	MainFunc *ircode.Function
	// Cached value
	malloc         *ircode.Function
	free           *ircode.Function
	merge          *ircode.Function
	runtimePackage *Package
}

// Global variable that avoids compiling a package twice
var genPackages = make(map[*types.Package]*Package)

// GeneratePackage generates IR-code for a package, including all packages imported from there.
func GeneratePackage(p *types.Package, log *errlog.ErrorLog) *Package {
	if pkg, ok := genPackages[p]; ok {
		return pkg
	}
	pkg := &Package{TypePackage: p, Funcs: make(map[*types.Func]*ircode.Function), Imports: make(map[*types.Package]*Package)}
	genPackages[p] = pkg
	pkg.generate(log)
	return pkg
}

// AllImports returns a list of all directly or indirectly imported packages.
// The list contains no duplicates.
func AllImports(p *Package) []*Package {
	return allImports(p, nil)
}

func allImports(p *Package, all []*Package) []*Package {
	for _, irImport := range p.Imports {
		found := false
		for _, done := range all {
			if irImport == done {
				found = true
				break
			}
		}
		if found {
			continue
		}
		all = append(all, irImport)
	}
	return all
}

// Generates IR code for the package `p` and recursively for all imported packages.
func (p *Package) generate(log *errlog.ErrorLog) {
	println("PACKAGE", p.TypePackage.FullPath())
	// Generate IR-code for all imported packages
	for _, imp := range p.TypePackage.Imports {
		impPackage := GeneratePackage(imp, log)
		p.Imports[imp] = impPackage
	}
	// Map types.Func to ircode.Function
	for _, f := range p.TypePackage.Funcs {
		var name string
		if !f.IsExtern {
			// Name mangling is required due to generics
			name = mangleFunctionName(f)
		} else {
			// Do not mangle the name. The name of the function is the symbol name seen by the linker
			name = f.Name()
		}
		irf := ircode.NewFunction(name, f)
		if f.IsExtern {
			irf.IsExtern = true
			irf.IsExported = f.IsExported
		}
		p.Funcs[f] = irf
	}
	// Generate init function and global variables
	globalVars := make(map[*types.Variable]*ircode.Variable)
	irf := p.Funcs[p.TypePackage.InitFunc]
	irf.IsExported = true
	b := ircode.NewBuilder(irf)
	for _, v := range p.TypePackage.Variables() {
		irv := b.DefineGlobalVariable(v.Name(), v.Type)
		globalVars[v] = irv
	}
	for _, vexpr := range p.TypePackage.VarExpressions {
		genVarExpression(vexpr, p.TypePackage.Scope, b, p, globalVars)
	}
	b.Finalize()
	ssa.TransformToSSA(irf, nil, log)
	// Generate IR-code for all functions
	for _, f := range p.TypePackage.Funcs {
		if f.IsExtern {
			// Do not generate IR code for external functions
			continue
		}
		if p.TypePackage.InitFunc == f {
			continue
		}
		genFunc(p, f, globalVars, log)
		println(p.Funcs[f].ToString())
	}
	// Lookup the main function (if any)
	f := p.TypePackage.MainFunc()
	if f != nil {
		p.MainFunc = p.Funcs[f]
	}
}

// RuntimePackage ...
func (p *Package) RuntimePackage() *Package {
	tp := p.TypePackage.RuntimePackage()
	// Perhaps the build is without any runtime?
	if tp == nil {
		return nil
	}
	if p.TypePackage == tp {
		return p
	}
	rp, ok := p.Imports[tp]
	if !ok {
		panic("Oooops")
	}
	return rp
}

// GetMalloc returns the `Malloc` functions as implemented in the Fyr runtime.
// May return 0 when Fyr is compiled without a runtime supporting memory allocation.
func (p *Package) GetMalloc() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.malloc, p.runtimePackage
	}
	p.runtimePackage = p.RuntimePackage()
	if p.runtimePackage == nil {
		return nil, p.runtimePackage
	}
	for f, irf := range p.runtimePackage.Funcs {
		if f.Name() == "Malloc" {
			p.malloc = irf
		} else if f.Name() == "Free" {
			p.free = irf
		} else if f.Name() == "Merge" {
			p.merge = irf
		}
	}
	return p.malloc, p.runtimePackage
}

// GetFree returns the `Free` functions as implemented in the Fyr runtime.
// May return 0 when Fyr is compiled without a runtime supporting memory allocation.
func (p *Package) GetFree() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.free, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.free, p.runtimePackage
}

// GetMerge returns the `Merge` functions as implemented in the Fyr runtime.
// May return 0 when Fyr is compiled without a runtime supporting memory allocation.
func (p *Package) GetMerge() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.merge, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.merge, p.runtimePackage
}
