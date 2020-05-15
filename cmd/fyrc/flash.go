package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/types"
)

func flash(pkg *types.Package) error {
	cfg := config.BuildTarget().C99.Flash
	if cfg == nil || len(cfg.Commands) == 0 {
		return errors.New("No flash commands have been configured in the build target")
	}
	for _, c := range cfg.Commands {
		err := runCommand(pkg, c)
		if err != nil {
			return err
		}
	}
	return nil
}

func runCommand(pkg *types.Package, cfg *config.BuildTargetC99FlashCommand) error {
	binPath := binOutputPath(pkg)
	args := append([]string{}, cfg.Flags...)
	for i := range args {
		args[i] = strings.Replace(args[i], "{{BIN}}", binPath, -1)
		args[i] = strings.Replace(args[i], "{{NAME}}", pkg.Path, -1)
		args[i] = strings.Replace(args[i], "{{FLASH}}", config.Flash(), -1)
	}
	cmd := exec.Command(cfg.Command, args...)
	cmd.Dir = binPath
	cmd.Stdout, cmd.Stderr = getOutput()
	println(cmd.String())
	if err := cmd.Run(); cmd.ProcessState == nil || !cmd.ProcessState.Success() {
		if err != nil {
			return err
		}
		return errors.New("Unknown flash error")
	}
	return nil
}

func binOutputPath(pkg *types.Package) string {
	return filepath.Join(pkg.FullPath(), "bin", config.EncodedPlatformName())
}

func getOutput() (*os.File, *os.File) {
	if config.Verbose() {
		return os.Stdout, os.Stderr
	}
	return nil, nil
}
