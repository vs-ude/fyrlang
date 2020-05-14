package c99

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/irgen"
)

// CompileSources translates IR-code into C-code.
// This applies recursively to all imported packages.
func CompileSources(p *irgen.Package) error {
	for _, irImport := range irgen.AllImports(p) {
		if err := compileSources(irImport); err != nil {
			return err
		}
	}
	if err := compileSources(p); err != nil {
		return err
	}
	return nil
}

func compileSources(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	basename := filepath.Base(p.TypePackage.FullPath())
	args := append([]string{}, config.BuildTarget().C99.Compiler.Flags...)
	args = append(args, []string{"-c", "-I" + pkgPath, basename + ".c"}...)
	println("IN", pkgPath)
	compiler := exec.Command(config.BuildTarget().C99.Compiler.Command, args...)
	compiler.Dir = pkgPath
	compiler.Stdout, compiler.Stderr = getOutput()
	println(compiler.String())
	if err := compiler.Run(); compiler.ProcessState == nil || !compiler.ProcessState.Success() {
		if err != nil {
			return err
		}
		return errors.New("compiler error")
	}
	return nil
}

// Link a library or executable.
// All imported libraries are linked recursively
func Link(p *irgen.Package) error {
	for _, irImport := range irgen.AllImports(p) {
		if err := link(irImport); err != nil {
			return err
		}
	}
	if !p.TypePackage.IsExecutable() && config.Flash() != "" {
		return errors.New("Cannot flash a library")
	}
	return link(p)
}

func link(p *irgen.Package) error {
	if p.TypePackage.IsExecutable() {
		return linkExecutable(p)
	}
	return linkArchive(p)
}

func linkArchive(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	objFiles := objectFileNames(p)
	args := append([]string{}, config.BuildTarget().C99.Archiver.Flags...)
	args = append(args, archiveFileName(p))
	args = append(args, objFiles...)
	linker := exec.Command(config.BuildTarget().C99.Archiver.Command, args...)
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

func linkExecutable(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	binPath := binOutputPath(p)
	if err := os.MkdirAll(binPath, 0700); err != nil {
		return err
	}
	objFiles := objectFileNames(p)
	archiveFiles := importArchiveFileNames(p)
	args := append([]string{}, config.BuildTarget().C99.Linker.Flags...)
	args = append(args, []string{"-o", filepath.Join(binPath, filepath.Base(p.TypePackage.Path))}...)
	args = append(args, objFiles...)
	args = append(args, archiveFiles...)
	compiler := exec.Command(config.BuildTarget().C99.Linker.Command, args...)
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
	return filepath.Join(p.TypePackage.FullPath(), "bin", config.EncodedPlatformName())
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
	if config.Verbose() {
		return os.Stdout, os.Stderr
	}
	return nil, nil
}
