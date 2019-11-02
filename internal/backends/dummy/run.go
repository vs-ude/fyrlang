package dummy

import (
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend This backend does not do anything.
type Backend struct{}

// Run Runs the dummy backend for the given package.
func (Backend) Run(irPackages []*irgen.Package) (message string, err error) {
	return "Dummy backend used, nothing has been compiled.", nil
}

// PrintCurrentConfig dummy function.
func (Backend) PrintCurrentConfig() {}
