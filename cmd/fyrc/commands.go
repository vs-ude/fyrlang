package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vs-ude/fyrlang/internal/config"
)

const (
	helpCommand         string = "help"
	envCommand          string = "env"
	printBackendCommand string = "printBackendConf"
)

var help = `
Usage: fyrc <flags> <command> <path>

Commands:
  help             Prints this help message.
  env              Prints environment variables and configuration used by the compiler.
  printBackendConf Prints the selected configuration of the defined backend type.

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
			if flag.NFlag() == 0 {
				config.PrintConf()
				os.Exit(0)
			}
		case printBackendCommand:
			b := setupBackend()
			b.PrintCurrentConfig()
			os.Exit(0)
		}
	}
}

func printHelp() {
	fmt.Print(help)
	flag.PrintDefaults()
}
