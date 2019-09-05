package backend

import (
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend is the interface backends have to follow in order to be usable in the compiler.
type Backend interface {
	Run(irPackages []*irgen.Package) (string, error)
}
