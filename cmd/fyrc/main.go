package main

import (
	"flag"
	"os"

	"github.com/vs-ude/fyrlang/internal/backends/backend"
	"github.com/vs-ude/fyrlang/internal/backends/c99"
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
	var packages []*types.Package
	for i := 0; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		println("Target Package:", arg)
		rootScope := types.NewRootScope()
		p, err := types.NewPackage(arg, rootScope, lmap, log)
		if err != nil {
			continue
		}
		err = p.Parse(lmap, log)
		if err != nil {
			continue
		}
		packages = append(packages, p)
	}
	if len(log.Errors) != 0 {
		for _, e := range log.Errors {
			println(errlog.ErrorToString(e, lmap))
		}
		println("ERROR")
		os.Exit(1)
	} else {
		println("OK")
	}

	// Setup the backend
	var usedBackend backend.Backend
	if flagNative {
		usedBackend = c99.C99Backend{}
	}

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
