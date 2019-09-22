package c99

import (
	"fmt"

	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend This backend implements compilation to native binaries via C99 code.
type Backend struct {
	config Config
}

// Run Runs the c99 backend for the given package.
func (Backend) Run(irPackages []*irgen.Package) (message string, err error) {
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
func NewBackend(compilerPath string, compilerConfigPath string) Backend {
	b := Backend{
		config: Config{},
	}
	if compilerPath != "" && compilerConfigPath != "" {
		backend.LoadConfig(compilerConfigPath, &b.config)
		b.config.Compiler.Bin = compilerPath
		fmt.Println("Warning: Incorrect configuration of the compiler could lead to undefined behavior and issues.")
	} else if compilerPath != "" && compilerConfigPath == "" {
		backend.LoadConfig(getConfigName(compilerPath), &b.config)
	} else if compilerPath == "" && compilerConfigPath != "" {
		backend.LoadConfig(compilerConfigPath, &b.config)
	} else {
		b.config.Default()
	}
	return b
}
