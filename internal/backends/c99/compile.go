package c99

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	args := []string{"/usr/bin/gcc", "-c", "-I" + pkgPath, basename + ".c"}
	println("IN", pkgPath)
	procAttr := &os.ProcAttr{Dir: pkgPath}
	println(strings.Join(args, " "))
	proc, err := os.StartProcess("/usr/bin/gcc", args, procAttr)
	if err != nil {
		return err
	}
	state, err := proc.Wait()
	if err != nil {
		return err
	}
	if !state.Success() {
		return errors.New("gcc compile error")
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
	args := []string{"/usr/bin/ar", "rcs"}
	args = append(args, archiveFileName(p))
	args = append(args, objFiles...)
	procAttr := &os.ProcAttr{Dir: pkgPath}
	println(strings.Join(args, " "))
	proc, err := os.StartProcess("/usr/bin/ar", args, procAttr)
	if err != nil {
		return err
	}
	state, err := proc.Wait()
	if err != nil {
		return err
	}
	if !state.Success() {
		if err != nil {
			return err
		}
		return errors.New("ar link error")
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
	args := []string{"/usr/bin/gcc", "-o", filepath.Join(binPath, filepath.Base(p.TypePackage.Path))}
	args = append(args, objFiles...)
	args = append(args, archiveFiles...)
	procAttr := &os.ProcAttr{Dir: pkgPath}
	println("IN", pkgPath)
	println("gcc", strings.Join(args, " "))
	proc, err := os.StartProcess("/usr/bin/gcc", args, procAttr)
	if err != nil {
		return err
	}
	state, err := proc.Wait()
	if err != nil {
		return err
	}
	if !state.Success() {
		if err != nil {
			return err
		}
		return errors.New("gcc link error")
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
