package main

import (
	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
	"github.com/vs-ude/fyrlang/internal/backends/dummy"
	"github.com/vs-ude/fyrlang/internal/config"
)

func setupBackend() backend.Backend {
	if config.BuildTarget().C99 != nil {
		return c99.NewBackend()
	}
	return dummy.Backend{}
}
