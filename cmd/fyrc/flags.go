package main

import (
	"flag"
)

var flagNative bool
var flagNativeCompilerBinary string
var flagNativeCompilerConfiguration string
var flagVulkan bool

func init() {
	// Native (c99) flags
	flag.BoolVar(&flagNative, "native", false, "Compiles the target into a system native binary via C.")
	flag.BoolVar(&flagNative, "n", false, "Compiles the target into a system native binary via C.")
	flag.StringVar(&flagNativeCompilerBinary, "nc", "", "Specifies the `compiler` used by the native backend.")
	flag.StringVar(&flagNativeCompilerConfiguration, "ncc", "", "Specifies the `configuration` for the compiler used by the native backend.")

	// Vulkan flags
	flag.BoolVar(&flagVulkan, "vulkan", false, "Compiles the target into SPIR-V code using vulkan.")
}
