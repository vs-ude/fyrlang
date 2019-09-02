package c99

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vs-ude/fyrlang/internal/irgen"
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
	// Generate C-code for all functions
	for _, irf := range p.Funcs {
		f := generateFunction(mod, p, irf)
		mod.Elements = append(mod.Elements, f)
		if irf == p.MainFunc {
			mod.MainFunc = f
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

func pkgOutputPath(p *irgen.Package) string {
	if p.TypePackage.IsInFyrPath() {
		return filepath.Join(p.TypePackage.RepoPath, "pkg", runtime.GOOS+"_"+runtime.GOARCH, p.TypePackage.Path)
	}
	return filepath.Join(p.TypePackage.FullPath(), "pkg", runtime.GOOS+"_"+runtime.GOARCH)
}
