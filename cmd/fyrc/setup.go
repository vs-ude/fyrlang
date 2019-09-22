package main

import (
	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
	"github.com/vs-ude/fyrlang/internal/backends/dummy"
)

func setupBackend() backend.Backend {
	if flagNative {
		return c99.NewBackend(flagNativeCompilerBinary, flagNativeCompilerConfiguration)
	}
	return dummy.Backend{}
}
