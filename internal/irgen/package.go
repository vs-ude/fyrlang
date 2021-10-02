package irgen

import (
	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Package is the IR-code representation of a Fyr package.
type Package struct {
	// The type-checked package for which this ircode package has been generated.
	TypePackage *types.Package
	Funcs       map[*types.Func]*ircode.Function
	Imports     map[*types.Package]*Package
	// May be null if no main function is defined
	MainFunc *ircode.Function
	// Cached value
	malloc *ircode.Function
	// Cached value
	mallocSlice *ircode.Function
	// Cached value
	free *ircode.Function
	// Cached value
	merge *ircode.Function
	// Cached value
	panicFunc *ircode.Function
	// Cached value
	printlnFunc *ircode.Function
	// Cached value
	groupOf *ircode.Function
	// Cached value
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
	all := []*Package{}
	for _, irImport := range p.Imports {
		if irImport.TypePackage.Path == "runtime" {
			all = append(allImports(irImport, all), irImport)
			break
		}
	}
	return allImports(p, all)
}

func allImports(p *Package, all []*Package) []*Package {
	for _, irImport := range p.Imports {
		found := false
		if irImport.TypePackage.Path == "runtime" {
			continue
		}
		all = append(all, allImports(irImport, all)...)
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
		if !f.IsExtern && !f.NoNameMangling {
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
		// Create the dual-function if required.
		if f.DualFunc != nil {
			name = mangleDualFunctionName(f.DualFunc)
			irfDual := ircode.NewFunction(name, f.DualFunc)
			p.Funcs[f.DualFunc] = irfDual
		}
	}
	// Generate init function and global variables
	globalVars := make(map[*types.Variable]*ircode.Variable)
	var globalVarsList []*ircode.Variable
	init := p.Funcs[p.TypePackage.InitFunc]
	// The init function must be callable by other packages to initialize this package
	init.IsExported = true
	b := ircode.NewBuilder(init)
	for _, v := range p.TypePackage.Variables() {
		irv := b.DefineGlobalVariable(v.Name(), v.Type)
		globalVars[v] = irv
		globalVarsList = append(globalVarsList, irv)
	}
	for _, c := range p.TypePackage.Components() {
		if !c.IsStatic {
			continue
		}
		for _, f := range c.Fields {
			irv := b.DefineGlobalVariable(f.Var.Name(), f.Var.Type)
			globalVars[f.Var] = irv
			globalVarsList = append(globalVarsList, irv)
			genVarExpression(f.Expression, c.ComponentScope, b, p, globalVars)
		}
	}
	// globalGrouping := ssa.GenerateGlobalVarsGrouping(irf, globalVarsList, log)
	for _, vexpr := range p.TypePackage.VarExpressions {
		genVarExpression(vexpr, p.TypePackage.Scope, b, p, globalVars)
	}
	b.Finalize()
	// TODO ssa.TransformToSSA(init, init, nil, nil, globalGrouping, log)
	// Generate IR-code for all functions
	for f, irf := range p.Funcs {
		if f.IsExtern {
			// Do not generate IR code for external functions
			continue
		}
		// Do not process the AST of __init_* functions, because they are compiler generated
		if p.TypePackage.InitFunc == f || (f.Component != nil && f == f.Component.InitFunc) {
			continue
		}
		genFunc(p, f, globalVars /* globalGrouping,*/, log)
		if config.Verbose() {
			println(irf.ToString())
		}
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

// ResolvePackage ...
func (p *Package) ResolvePackage(tp *types.Package) *Package {
	if p.TypePackage == tp {
		return p
	}
	rp, ok := p.Imports[tp]
	if !ok {
		panic("Oooops")
	}
	return rp
}

// ResolveFunc ...
func (p *Package) ResolveFunc(f *types.Func) (*ircode.Function, *Package) {
	tpkg := f.OuterScope.PackageScope().Package
	pkg := p.ResolvePackage(tpkg)
	irf, ok := pkg.Funcs[f]
	if !ok {
		panic("Oooops")
	}
	return irf, pkg
}

// GetMalloc returns the `Malloc` functions as implemented in the Fyr runtime.
// May return 0 when Fyr is compiled without a runtime supporting memory allocation.
// GetMalloc also sets the `p.runtimePackage` field if necessary and looksup other builtin functions.
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
		} else if f.Name() == "MallocSlice" {
			p.mallocSlice = irf
		} else if f.Name() == "Free" {
			p.free = irf
		} else if f.Name() == "Merge" {
			p.merge = irf
		} else if f.Name() == "Println" {
			p.printlnFunc = irf
		} else if f.Name() == "Panic" {
			p.panicFunc = irf
		} else if f.Name() == "GroupOf" {
			p.groupOf = irf
		}
	}
	return p.malloc, p.runtimePackage
}

// GetMallocSlice returns the `MalloSlice` functions as implemented in the Fyr runtime.
// May return 0 when Fyr is compiled without a runtime supporting memory allocation.
func (p *Package) GetMallocSlice() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.mallocSlice, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.mallocSlice, p.runtimePackage
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

// GetPrintln returns the `Println` functions as implemented in the Fyr runtime.
func (p *Package) GetPrintln() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.printlnFunc, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.printlnFunc, p.runtimePackage
}

// GetPanic returns the `Panic` functions as implemented in the Fyr runtime.
func (p *Package) GetPanic() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.panicFunc, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.panicFunc, p.runtimePackage
}

// GetGroupOf returns the `GroupOf` functions as implemented in the Fyr runtime.
func (p *Package) GetGroupOf() (*ircode.Function, *Package) {
	if p.runtimePackage != nil {
		return p.groupOf, p.runtimePackage
	}
	// Cache runtime functions
	p.GetMalloc()
	return p.groupOf, p.runtimePackage
}
