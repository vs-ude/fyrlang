package main

import (
	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
	"github.com/vs-ude/fyrlang/internal/backends/dummy"
)

func setupBackend() (backend backend.Backend) {
	if flagNative {
		backend = c99.Backend{}
	} else {
		backend = dummy.Backend{}
	}
	return
}
