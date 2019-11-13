package main

import (
	"flag"
	"fmt"
	"os"
)

var flagHelp bool
var flagVersion bool
var flagVerbose bool
var flagNative bool
var flagNativeCompilerBinary string
var flagNativeCompilerConfiguration string
var flagVulkan bool

func init() {
	// Common flags
	flag.BoolVar(&flagHelp, "h", false, "Print this help message.")
	flag.BoolVar(&flagVersion, "V", false, "Print the compiler version.")
	flag.BoolVar(&flagVerbose, "v", false, "More verbose output while compiling. Mostly helpful for compiler development.")

	// Native (c99) flags
	flag.BoolVar(&flagNative, "native", false, "Compiles the target into a system native binary via C.")
	flag.BoolVar(&flagNative, "n", false, "Compiles the target into a system native binary via C.")
	flag.StringVar(&flagNativeCompilerBinary, "nc", "", "Specifies the `compiler` used by the native backend.")
	flag.StringVar(&flagNativeCompilerConfiguration, "ncc", "", "Specifies the `configuration` for the compiler used by the native backend.")

	// Vulkan flags
	flag.BoolVar(&flagVulkan, "vulkan", false, "Compiles the target into SPIR-V code using vulkan.")
}

func exitingFlags() {
	if flagVersion {
		fmt.Printf("Fyr compiler version %s, built on %s\n", version, buildDate)
		os.Exit(0)
	}
	if flagHelp {
		printHelp()
		os.Exit(0)
	}
}
