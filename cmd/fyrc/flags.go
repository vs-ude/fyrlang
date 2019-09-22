package main

import (
	"flag"
)

var flagNative bool

func init() {
	flag.BoolVar(&flagNative, "native", true, "Compiles the target into a system native binary via C.") // TODO: set to false once we have more backends
	flag.BoolVar(&flagNative, "n", true, "Compiles the target into a system native binary via C.")
}
