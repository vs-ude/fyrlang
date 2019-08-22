package irgen

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Package ...
type Package struct {
	TypePackage *types.Package
	Funcs       map[*types.Func]*ircode.Function
}

var genPackages = make(map[*types.Package]*Package)

// GeneratePackage ...
func GeneratePackage(p *types.Package) *Package {
	if pkg, ok := genPackages[p]; ok {
		return pkg
	}
	pkg := &Package{TypePackage: p, Funcs: make(map[*types.Func]*ircode.Function)}
	genPackages[p] = pkg
	pkg.generate()
	return pkg
}

// FullPath ...
func (p *Package) FullPath() string {
	return filepath.Join(p.TypePackage.RepoPath, p.TypePackage.Path)
}

// Generates IR code for the package `p` and recursively for all imported packages.
func (p *Package) generate() {
	println("PACKAGE", p.FullPath)
	for _, f := range p.TypePackage.Funcs {
		irf := genFunc(f)
		p.Funcs[f] = irf
		println(irf.ToString())
	}
	for _, imp := range p.TypePackage.Imports {
		GeneratePackage(imp)
	}
}
