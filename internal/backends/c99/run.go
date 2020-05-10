package c99

import (
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend This backend implements compilation to native binaries via C99 code.
type Backend struct {
}

// Run Runs the c99 backend for the given package.
func (b Backend) Run(irPackages []*irgen.Package) (message string, err error) {
	for _, p := range irPackages {
		err = GenerateSources(p)
		if err != nil {
			message = "Error writing target sources"
			return
		}
		err = CompileSources(p)
		if err != nil {
			message = "Unable to compile the sources"
			return
		}
		err = Link(p)
		if err != nil {
			message = "Error while linking the binary"
			return
		}
	}
	message = "Successfully compiled the package."
	return
}

// NewBackend constructs the backend and its configuration according to the given flags.
func NewBackend() *Backend {
	return &Backend{}
}
