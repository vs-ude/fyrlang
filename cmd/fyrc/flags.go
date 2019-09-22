package main

import (
	"flag"
)

var flagNative bool
var flagNativeCompilerBinary string
var flagNativeCompilerConfiguration string

func init() {
	flag.BoolVar(&flagNative, "native", true, "Compiles the target into a system native binary via C.") // TODO: set to false once we have more backends
	flag.BoolVar(&flagNative, "n", true, "Compiles the target into a system native binary via C.")
	flag.StringVar(&flagNativeCompilerBinary, "nc", "", "Specifies the `compiler` used by the native backend.")
	flag.StringVar(&flagNativeCompilerConfiguration, "ncc", "", "Specifies the `configuration` for the compiler used by the native backend.")
}
