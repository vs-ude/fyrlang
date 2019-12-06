package c99

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend This backend implements compilation to native binaries via C99 code.
type Backend struct {
	config Config
}

// Run Runs the c99 backend for the given package.
func (b Backend) Run(irPackages []*irgen.Package) (message string, err error) {
	for _, p := range irPackages {
		err = GenerateSources(p)
		if err != nil {
			message = "Error writing target sources"
			return
		}
		err = CompileSources(p, b.config)
		if err != nil {
			message = "Unable to compile the sources"
			return
		}
		err = Link(p, b.config)
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
		_, p := filepath.Split(compilerConfigPath)
		b.config.configTriplet = strings.TrimSuffix(p, ".json")
		fmt.Println("Warning: Incorrect configuration of the compiler could lead to undefined behavior and issues.")
	} else if compilerPath != "" && compilerConfigPath == "" {
		p := getConfigName(compilerPath)
		backend.LoadConfig(p, &b.config)
		b.config.configTriplet = strings.TrimSuffix(p, ".json")
	} else if compilerPath == "" && compilerConfigPath != "" {
		backend.LoadConfig(compilerConfigPath, &b.config)
		_, p := filepath.Split(compilerConfigPath)
		b.config.configTriplet = strings.TrimSuffix(p, ".json")
	} else {
		b.config.Default()
	}
	b.config.setupMemConfig()
	return b
}

// PrintCurrentConfig prints the configuration of the backend.
func (b Backend) PrintCurrentConfig() {
	backend.PrintConfig(&b.config)
}
