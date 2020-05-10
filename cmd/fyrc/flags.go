package main

import (
	"flag"
	"os"

	"github.com/vs-ude/fyrlang/internal/config"
)

var flagVerbose bool
var flagBuildTargetName string
var flagDebug bool
var flagFlash string

func init() {
	// Common flags
	flag.BoolVar(&flagVerbose, "v", false, "More verbose output while compiling. Mostly helpful for compiler development.")
	flag.BoolVar(&flagDebug, "d", false, "Choose the debug build target. A JSON file of the name <build_target>-debug must be located in the build_targets directory.")
	flag.StringVar(&flagFlash, "flash", "", "Flashes the program on the micro-controller. The value is the serial port to which the programmer is connected.")
	flag.StringVar(&flagBuildTargetName, "b", "", "Name of the build target. A JSON file of the same name must be located in the build_targets directory.")
}

func setupCommonFlags() {
	if flagVerbose {
		config.SetVerbose()
	}
	if flagFlash != "" {
		config.SetFlash(flagFlash)
	}
	err := config.LoadBuildTarget(flagBuildTargetName, flagDebug)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}
