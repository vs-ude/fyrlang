package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vs-ude/fyrlang/internal/config"
)

const (
	helpCommand    string = "help"
	envCommand     string = "env"
	versionCommand string = "version"
)

var help = `
Usage: fyrc <flags> <command> <path>

Commands:
  help             Prints this help message.
  version	       Prints the compiler version.
  env              Prints environment variables and configuration used by the compiler.

Flags:
`

func commands() {
	for _, arg := range flag.Args() {
		switch arg {
		case helpCommand:
			if flag.NFlag() == 0 {
				printHelp()
				os.Exit(0)
			}
		case envCommand:
			err := config.LoadBuildTarget(flagBuildTargetName, flagDebug)
			if err != nil {
				println(err.Error())
			}
			config.PrintConf()
			os.Exit(0)
		case versionCommand:
			if flag.NFlag() == 0 {
				fmt.Printf("Fyr compiler version %s, built on %s\n", version, buildDate)
				os.Exit(0)
			}
		}
	}

	if len(flag.Args()) == 0 {
		printHelp()
		os.Exit(0)
	}
}

func printHelp() {
	fmt.Print(help)
	flag.PrintDefaults()
}
