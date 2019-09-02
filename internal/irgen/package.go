package irgen

import (
	"github.com/vs-ude/fyrlang/internal/ircode"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Package ...
type Package struct {
	TypePackage *types.Package
	Funcs       map[*types.Func]*ircode.Function
	Imports     map[*types.Package]*Package
	MainFunc    *ircode.Function
}

var genPackages = make(map[*types.Package]*Package)

// GeneratePackage ...
func GeneratePackage(p *types.Package) *Package {
	if pkg, ok := genPackages[p]; ok {
		return pkg
	}
	pkg := &Package{TypePackage: p, Funcs: make(map[*types.Func]*ircode.Function), Imports: make(map[*types.Package]*Package)}
	genPackages[p] = pkg
	pkg.generate()
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
func (p *Package) generate() {
	println("PACKAGE", p.TypePackage.FullPath())
	for _, f := range p.TypePackage.Funcs {
		irf := genFunc(f)
		p.Funcs[f] = irf
		println(irf.ToString())
	}
	f := p.TypePackage.MainFunc()
	if f != nil {
		p.MainFunc = p.Funcs[f]
	}
	for _, imp := range p.TypePackage.Imports {
		impPackage := GeneratePackage(imp)
		p.Imports[imp] = impPackage
	}
}
