package types

import (
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// PackageGenerator is used to generate packages and log events to the provided location.
type PackageGenerator struct {
	log  *errlog.ErrorLog
	lmap *errlog.LocationMap
}

// NewPackageGenerator creates a new PackageGenerator.
func NewPackageGenerator(log *errlog.ErrorLog, lmap *errlog.LocationMap) *PackageGenerator {
	pGen := &PackageGenerator{}
	pGen.log = log
	pGen.lmap = lmap
	return pGen
}

// Run generates the packages and performs type checking.
func (pGen *PackageGenerator) Run(packageNames []string) (packages []*Package) {
	for i := 0; i < len(packageNames); i++ {
		rootScope := NewRootScope()
		// Generate and type check the runtime package
		pRuntime, err := NewPackage(filepath.Join(config.FyrBase(), "lib/runtime"), rootScope, pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		err = pRuntime.Parse(pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		// Generate and type check the package
		pName := packageNames[i]
		println("Target Package:", pName)
		p, err := NewPackage(pName, rootScope, pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		// Compiling the runtime itself? Then we are already done
		if p.FullPath() == pRuntime.FullPath() {
			packages = append(packages, pRuntime)
			continue
		}
		p.addImport(pRuntime)
		err = p.Parse(pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		packages = append(packages, p)
	}
	return
}
