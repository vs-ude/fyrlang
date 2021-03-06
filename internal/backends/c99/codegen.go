package c99

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/irgen"
	"github.com/vs-ude/fyrlang/internal/types"
)

// GenerateSources writes C-header and C-code files.
// This applies recursively to all imported packages.
func GenerateSources(p *irgen.Package) error {
	if err := generateSources(p); err != nil {
		return err
	}
	for _, irImport := range irgen.AllImports(p) {
		if err := generateSources(irImport); err != nil {
			return err
		}
	}
	return nil
}

func generateSources(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	println("MAKE DIR", pkgPath)
	if err := os.MkdirAll(pkgPath, 0700); err != nil {
		return err
	}
	mod := NewModule(p)
	// Add all imports
	for _, irImport := range p.Imports {
		basename := filepath.Base(irImport.TypePackage.FullPath())
		headerFile := filepath.Join(pkgOutputPath(irImport), basename+".h")
		mod.AddInclude(headerFile, false)
	}
	// Declare all named types
	for name, t := range p.TypePackage.Scope.Types {
		declareNamedType(mod, nil, name, t)
	}
	// Declare all named types in components
	for _, c := range p.TypePackage.Scope.ComponentTypes() {
		for name, t := range c.ComponentScope.Types {
			declareNamedType(mod, nil, name, t)
		}
	}
	// Define all named types
	for name, t := range p.TypePackage.Scope.Types {
		defineNamedType(mod, nil, name, t)
	}
	// Define all named types
	for _, c := range p.TypePackage.Scope.ComponentTypes() {
		for name, t := range c.ComponentScope.Types {
			defineNamedType(mod, nil, name, t)
		}
	}

	// Generate the init function first, because it declares the global variables
	initIrf := p.Funcs[p.TypePackage.InitFunc]
	cf := generateFunction(mod, p, initIrf)
	mod.Elements = append(mod.Elements, cf)
	// Generate C-code for all functions
	for _, irf := range p.Funcs {
		if irf == initIrf {
			continue
		}
		cf := generateFunction(mod, p, irf)
		mod.Elements = append(mod.Elements, cf)
		if irf == p.MainFunc {
			mod.MainFunc = cf
		}
	}

	// ...
	basename := filepath.Base(p.TypePackage.FullPath())
	src := mod.Implementation(p.TypePackage.Path, basename)
	header := mod.Header(p.TypePackage.Path, basename)
	if err := ioutil.WriteFile(filepath.Join(pkgPath, basename+".c"), []byte(src), 0600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(pkgPath, basename+".h"), []byte(header), 0600); err != nil {
		return err
	}
	return nil
}

func resolveFunc(mod *Module, f *types.Func) (*irgen.Package, *ircode.Function) {
	// Determine the package in which the function is defined
	fpkg := f.OuterScope.PackageScope().Package
	// Generics funcs belong to the package that instantiated the generic, not the package that defined the generic.
	if f.IsGenericInstanceMemberFunc() || f.IsGenericInstanceFunc() {
		if f.OuterScope.Kind != types.GenericTypeScope {
			panic("Ooooops")
		}
		fpkg = f.OuterScope.Package
	}
	irpkg := mod.Package
	// Is the function defined in another package than that of the Module?
	if irpkg.TypePackage != fpkg {
		var ok bool
		irpkg, ok = mod.Package.Imports[fpkg]
		if !ok {
			panic("Oooops " + fpkg.Path)
		}
	}
	irf, ok := irpkg.Funcs[f]
	if !ok {
		println(f.IsGenericInstanceMemberFunc())
		fmt.Printf("%T\n", f.Type.Target)
		panic("Oooops " + f.Name() + " " + irpkg.TypePackage.FullPath() + " from " + mod.Package.TypePackage.FullPath())
	}
	return irpkg, irf
}

func pkgOutputPath(p *irgen.Package) string {
	// TODO: This will fail on windows, because TypePackage.Path contains slashes
	if p.TypePackage.IsInFyrBase() {
		return filepath.Join(config.CacheDirPath(), "lib", config.EncodedPlatformName(), p.TypePackage.Path)
	} else if p.TypePackage.IsInFyrPath() {
		return filepath.Join(p.TypePackage.RepoPath, "pkg", config.EncodedPlatformName(), p.TypePackage.Path)
	}
	return filepath.Join(p.TypePackage.FullPath(), "pkg", config.EncodedPlatformName())
}
