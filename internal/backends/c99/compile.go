package c99

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vs-ude/fyrlang/internal/irgen"
)

// CompileSources ...
func CompileSources(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	args := []string{"/usr/bin/gcc", "-c", "module.c"}
	println(pkgPath)
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

// Link ...
func Link(p *irgen.Package) error {
	if p.TypePackage.IsExecutable() {
		return linkExecutable(p)
	}
	return linkArchive(p)
}

func linkArchive(p *irgen.Package) error {
	pkgPath := pkgOutputPath(p)
	objFiles := objectFileNames(p)
	args := []string{"/usr/bin/ar", "rcs"}
	args = append(args, pkgPath+".a")
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
	binPath := binOutputPath(p)
	pkgPath := pkgOutputPath(p)
	objFiles := objectFileNames(p)
	archiveFiles := archiveFileNames(p)
	args := []string{"/usr/bin/gcc", "-o", filepath.Join(binPath, filepath.Base(p.TypePackage.Path))}
	args = append(args, objFiles...)
	args = append(args, archiveFiles...)
	procAttr := &os.ProcAttr{Dir: pkgPath}
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
	return filepath.Join(p.TypePackage.RepoPath, "bin", runtime.GOOS+"_"+runtime.GOARCH)
}

func objectFileNames(p *irgen.Package) []string {
	return []string{"module.o"}
}

func archiveFileNames(p *irgen.Package) []string {
	var files []string
	for _, irImport := range irgen.AllImports(p) {
		files = append(files, pkgOutputPath(irImport)+".a")
	}
	return files
}
