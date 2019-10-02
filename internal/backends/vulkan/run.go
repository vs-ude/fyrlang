package vulkan

import (
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// Backend This backend compiles Fyr to SPIR-V using vulkan.
type Backend struct{}

// Run Runs the vulkan backend for the given package.
func (Backend) Run(irPackages []*irgen.Package) (message string, err error) {
	// TODO: implement.
	// This runs the actual workflow of the backend. You are free to structure this however you wish.
	message = "Vulkan backend selected"
	return
}

// NewBackend Constructs the backend.
func NewBackend() Backend {
	// TODO: place required setup of the backend in here.
	// This should contain things like paths to required external utilities.
	// They should be stored in the Backend struct.
	// The backend configuration framework (backends/backend/config) will be used in the future but is not stable yet.
	// The possible configuration options in this function will be translated into the config backend once both are ready.
	return Backend{}
}
