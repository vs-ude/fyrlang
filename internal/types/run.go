package types

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

// PackageGenerator Is used to generate packages and log events to the provided location.
type PackageGenerator struct {
	log  *errlog.ErrorLog
	lmap *errlog.LocationMap
}

// NewPackageGenerator Creates a new PackageGenerator.
func NewPackageGenerator(log *errlog.ErrorLog, lmap *errlog.LocationMap) *PackageGenerator {
	pGen := PackageGenerator{}
	pGen.log = log
	pGen.lmap = lmap
	return &pGen
}

// Run Generates the packages and performs type checking.
func (pGen *PackageGenerator) Run(packageNames []string) (packages []*Package) {
	for i := 0; i < len(packageNames); i++ {
		pName := packageNames[i]
		println("Target Package:", pName)
		rootScope := NewRootScope()
		p, err := NewPackage(pName, rootScope, pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		err = p.Parse(pGen.lmap, pGen.log)
		if err != nil {
			continue
		}
		packages = append(packages, p)
	}
	return
}
