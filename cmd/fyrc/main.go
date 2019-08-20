package main

import (
	"flag"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/types"
)

func main() {
	flag.Parse()
	log := errlog.NewErrorLog()
	lmap := errlog.NewLocationMap()
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
	}
	if len(log.Errors) != 0 {
		for _, e := range log.Errors {
			println(errlog.ErrorToString(e, lmap))
		}
	} else {
		println("OK")
	}
}
