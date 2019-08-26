package c99

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vs-ude/fyrlang/internal/irgen"
)

// GenerateSources ...
func GenerateSources(p *irgen.Package) error {
	path := outputPath(p)
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	mod := &Module{}
	// ...
	src := mod.Implementation(p.TypePackage.Path, "module")
	header := mod.Header(p.TypePackage.Path, "module")
	if err := ioutil.WriteFile(filepath.Join(path, "module.c"), []byte(src), 0700); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(path, "module.h"), []byte(header), 0700); err != nil {
		return err
	}
	return nil
}

func outputPath(p *irgen.Package) string {
	// TODO: Architecture
	return filepath.Join(p.TypePackage.RepoPath, "pkg", runtime.GOOS+"_"+runtime.GOARCH, p.TypePackage.Path)
}
