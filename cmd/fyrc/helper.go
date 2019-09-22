package main

import (
	"github.com/vs-ude/fyrlang/internal/errlog"
)

func printErrors(log *errlog.ErrorLog, lmap *errlog.LocationMap) {
	for _, e := range log.Errors {
		println(errlog.ErrorToString(e, lmap))
	}
	println("ERROR")
}
