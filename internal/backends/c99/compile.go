package c99

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	c "github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// CompileSources translates IR-code into C-code.
// This applies recursively to all imported packages.
func CompileSources(p *irgen.Package, config Config) error {
	for _, irImport := range irgen.AllImports(p) {
		if err := compileSources(irImport, config); err != nil {
			return err
		}
	}
	if err := compileSources(p, config); err != nil {
		return err
	}
	return nil
}

func compileSources(p *irgen.Package, config Config) error {
	pkgPath := pkgOutputPath(p)
	basename := filepath.Base(p.TypePackage.FullPath())
	args := append(getCompilationArgs(config), []string{"-c", "-I" + pkgPath, basename + ".c"}...)
	println("IN", pkgPath)
	compiler := exec.Command(config.Compiler.Bin, args...)
	compiler.Dir = pkgPath
	compiler.Stdout, compiler.Stderr = getOutput()
	println(compiler.String())
	if err := compiler.Run(); !compiler.ProcessState.Success() {
		if err != nil {
			return err
		}
		return errors.New("compiler error")
	}
	return nil
}

// Link a library or executable.
// All imported libraries are linked recursively
func Link(p *irgen.Package, config Config) error {
	for _, irImport := range irgen.AllImports(p) {
		if err := link(irImport, config); err != nil {
			return err
		}
	}
	return link(p, config)
}

func link(p *irgen.Package, config Config) error {
	if p.TypePackage.IsExecutable() {
		return linkExecutable(p, config)
	}
	return linkArchive(p, config)
}

func linkArchive(p *irgen.Package, config Config) error {
	pkgPath := pkgOutputPath(p)
	objFiles := objectFileNames(p)
	args := strings.Split(config.Archiver.Flags, " ")
	args = append(args, archiveFileName(p))
	args = append(args, objFiles...)
	linker := exec.Command(config.Archiver.Bin, args...)
	linker.Dir = pkgPath
	linker.Stdout, linker.Stderr = getOutput()
	println(linker.String())
	if err := linker.Run(); !linker.ProcessState.Success() {
		if err != nil {
			return err
		}
		return errors.New("archive link error")
	}
	return nil
}

func linkExecutable(p *irgen.Package, config Config) error {
	pkgPath := pkgOutputPath(p)
	binPath := binOutputPath(p)
	if err := os.MkdirAll(binPath, 0700); err != nil {
		return err
	}
	objFiles := objectFileNames(p)
	archiveFiles := importArchiveFileNames(p)
	args := []string{"-o", filepath.Join(binPath, filepath.Base(p.TypePackage.Path))}
	args = append(args, objFiles...)
	args = append(args, archiveFiles...)
	compiler := exec.Command(config.Compiler.Bin, args...)
	compiler.Dir = pkgPath
	compiler.Stdout, compiler.Stderr = getOutput()
	println("IN", pkgPath)
	println(compiler.String())
	if err := compiler.Run(); !compiler.ProcessState.Success() {
		if err != nil {
			return err
		}
		return errors.New("executable link error")
	}
	return nil
}

func binOutputPath(p *irgen.Package) string {
	return filepath.Join(p.TypePackage.FullPath(), "bin", runtime.GOOS+"_"+runtime.GOARCH)
}

func objectFileNames(p *irgen.Package) []string {
	return []string{filepath.Base(p.TypePackage.FullPath()) + ".o"}
}

func importArchiveFileNames(p *irgen.Package) []string {
	var files []string
	for _, irImport := range irgen.AllImports(p) {
		files = append(files, archiveFileName(irImport))
	}
	return files
}

func archiveFileName(p *irgen.Package) string {
	if p.TypePackage.IsInFyrPath() {
		return pkgOutputPath(p) + ".a"
	}
	return filepath.Join(pkgOutputPath(p), filepath.Base(p.TypePackage.FullPath())) + ".a"
}

func getOutput() (*os.File, *os.File) {
	if c.Verbose() {
		return os.Stdout, os.Stderr
	}
	return nil, nil
}

func getCompilationArgs(config Config) []string {
	args := strings.Split(config.Compiler.RequiredFlags, " ")
	if c.Verbose() {
		debugArgs := strings.Split(config.Compiler.DebugFlags, " ")
		args = append(args, debugArgs...)
	} else {
		releaseArgs := strings.Split(config.Compiler.ReleaseFlags, " ")
		args = append(args, releaseArgs...)
	}
	return args
}
