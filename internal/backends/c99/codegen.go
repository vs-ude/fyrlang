package c99

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

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
		var headerFile string
		if irImport.TypePackage.IsInFyrPath() {
			headerFile = filepath.Join(pkgOutputPath(irImport), basename+".h")
		} else {
			headerFile = filepath.Join(pkgOutputPath(irImport), basename+".h")
		}
		mod.AddInclude(headerFile, false)
	}
	// Declare all named types
	for name, t := range p.TypePackage.Scope.Types {
		declareNamedType(mod, nil, name, t)
	}
	// Define all named types
	for name, t := range p.TypePackage.Scope.Types {
		defineNamedType(mod, nil, name, t)
	}
	// Generate C-code for all functions
	for _, irf := range p.Funcs {
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

func resolveFunc(mod *Module, f *types.Func) *ircode.Function {
	// Determine the package in which the function is defined
	fpkg := f.OuterScope.PackageScope().Package
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
		panic("Oooops")
	}
	return irf
}

func pkgOutputPath(p *irgen.Package) string {
	if p.TypePackage.IsInFyrBase() {
		return filepath.Join(config.CacheDirPath(), "lib")
	} else if p.TypePackage.IsInFyrPath() {
		return filepath.Join(p.TypePackage.RepoPath, "pkg", runtime.GOOS+"_"+runtime.GOARCH, p.TypePackage.Path)
	}
	return filepath.Join(p.TypePackage.FullPath(), "pkg", runtime.GOOS+"_"+runtime.GOARCH)
}
