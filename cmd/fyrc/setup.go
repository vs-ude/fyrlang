package main

import (
	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
	"github.com/vs-ude/fyrlang/internal/backends/dummy"
	"github.com/vs-ude/fyrlang/internal/backends/vulkan"
)

func setupBackend() backend.Backend {
	if flagNative {
		return c99.NewBackend(flagNativeCompilerBinary, flagNativeCompilerConfiguration)
	} else if flagVulkan {
		return vulkan.NewBackend()
	}
	return dummy.Backend{}
}
