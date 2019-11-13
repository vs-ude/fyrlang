package main

import (
	"flag"
	"os"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/irgen"
	"github.com/vs-ude/fyrlang/internal/types"
)

// The compiler version info
var (
	version   string = "dev"
	buildDate string = "-"
)

func main() {
	flag.Parse()
	commands()
	exitingFlags()
	setupCommonFlags()
	log := errlog.NewErrorLog()
	lmap := errlog.NewLocationMap()
	packageGenerator := types.NewPackageGenerator(log, lmap)
	usedBackend := setupBackend()

	// Generate packages and check types
	packages := packageGenerator.Run(flag.Args())
	// Bail out if type checks failed
	if len(log.Errors) != 0 {
		printErrors(log, lmap)
		os.Exit(1)
	}

	// Generate IR-code
	irPackages := irgen.Run(packages, log)
	// Bail out if IR-code based code analysis detects errors
	if len(log.Errors) != 0 {
		printErrors(log, lmap)
		os.Exit(1)
	}
	// From here on, no further programming errors can be detected.
	println("OK")

	// Generate executable code
	var message, err = usedBackend.Run(irPackages)
	if err != nil {
		println(message, "Description:", err.Error())
		os.Exit(1)
	} else {
		println(message)
	}
}
