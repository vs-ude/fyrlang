package irgen

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/types"
)

// Run Executes the ir generator for the given packages.
func Run(packages []*types.Package, log *errlog.ErrorLog) (generatedPackages []*Package) {
	for _, p := range packages {
		generatedPackages = append(generatedPackages, GeneratePackage(p, log))
	}
	return
}
