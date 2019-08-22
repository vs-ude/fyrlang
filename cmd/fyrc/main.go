package main

import (
	"flag"
	"os"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/irgen"
	"github.com/vs-ude/fyrlang/internal/types"
)

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

	// Generate code
	for _, p := range packages {
		irgen.GenPackage(p)
	}
}
