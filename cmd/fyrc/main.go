package main

import (
	"flag"
	"os"

	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
	"github.com/vs-ude/fyrlang/internal/backends/dummy"
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/irgen"
	"github.com/vs-ude/fyrlang/internal/types"
)

var flagNative bool

func init() {
	flag.BoolVar(&flagNative, "native", true, "Compiles the target into a system native binary via C.") // TODO: set to false once we have more backends
	flag.BoolVar(&flagNative, "n", true, "Compiles the target into a system native binary via C.")
}

func main() {
	flag.Parse()
	log := errlog.NewErrorLog()
	lmap := errlog.NewLocationMap()
	packageGenerator := types.NewPackageGenerator(log, lmap)
	usedBackend := setupBackend()

	// Generate packages and check types
	packages := packageGenerator.Run(flag.Args())

	if len(log.Errors) != 0 {
		printErrors(log, lmap)
		os.Exit(1)
	}
	println("OK")

	// Generate code
	irPackages := irgen.Run(packages)
	var message, err = usedBackend.Run(irPackages)
	if err != nil {
		println(message, "Description:", err.Error())
		os.Exit(1)
	} else {
		println(message)
	}
}

func setupBackend() (backend backend.Backend) {
	if flagNative {
		backend = c99.Backend{}
	} else {
		backend = dummy.Backend{}
	}
	return
}

func printErrors(log *errlog.ErrorLog, lmap *errlog.LocationMap) {
	for _, e := range log.Errors {
		println(errlog.ErrorToString(e, lmap))
	}
	println("ERROR")
}
