package irgen

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/types"
)

var genPackages = make(map[*types.Package]bool)

// GenPackage enerates IR code for the package `p` and recursively for all imported packages.
func GenPackage(p *types.Package) {
	if _, ok := genPackages[p]; ok {
		return
	}
	println("PACKAGE", filepath.Join(p.RepoPath, p.Path))
	genPackages[p] = true
	for _, f := range p.Funcs {
		irf := genFunc(f)
		println(irf.ToString())
	}
	for _, imp := range p.Imports {
		GenPackage(imp)
	}
}
