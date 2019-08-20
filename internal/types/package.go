package types

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// Package ...
type Package struct {
	Scope    *Scope
	Path     string
	RepoPath string
}

func newPackage(repoPath string, path string, rootScope *Scope, loc errlog.LocationRange) *Package {
	return &Package{RepoPath: repoPath, Path: path, Scope: newScope(rootScope, PackageScope, loc)}
}

// NewPackage ...
func NewPackage(repoPath string, rootScope *Scope, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	repoPath = filepath.Join(wd, repoPath)
	f := errlog.NewSourceFile(repoPath)
	fileNumber := lmap.AddFile(f)
	stat, err := os.Stat(repoPath)
	if err != nil {
		return nil, log.AddError(errlog.ErrorPackageNotFound, errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0), repoPath)
	}
	if !stat.IsDir() {
		return nil, log.AddError(errlog.ErrorPackageNotFound, errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0), repoPath)
	}
	ploc := errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0)
	return newPackage(repoPath, ".", rootScope, ploc), nil
}

// Parse ...
func (pkg *Package) Parse(lmap *errlog.LocationMap, log *errlog.ErrorLog) error {
	dir := filepath.Join(pkg.RepoPath, pkg.Path)
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	stats, err := file.Readdir(0)
	if err != nil {
		return err
	}
	var parseError error
	for _, stat := range stats {
		if stat.IsDir() {
			continue
		}
		if !strings.HasSuffix(stat.Name(), ".fyr") {
			continue
		}
		filePath := filepath.Join(dir, stat.Name())
		println("File:", filePath)
		src, err := ioutil.ReadFile(filePath)
		if err != nil {
			if parseError == nil {
				parseError = err
			}
			continue
		}
		filePtr := errlog.NewSourceFile(filePath)
		fileNumber := lmap.AddFile(filePtr)
		p := parser.NewParser(log)
		n, err := p.Parse(fileNumber, string(src), log)
		if err != nil {
			if parseError == nil {
				parseError = err
			}
			continue
		}
		err = ParseFile(pkg, n, lmap, log)
		if err != nil {
			if parseError == nil {
				parseError = err
			}
			continue
		}
	}
	return nil
}

// LookupPackage ...
func LookupPackage(path string, from *Package, loc errlog.LocationRange, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
	rootScope := from.Scope.Root()
	if filepath.IsAbs(path) || path == "" || path == "." || path == ".." {
		return nil, log.AddError(errlog.ErrorMalformedPackagePath, loc, path)
	}
	path2 := filepath.Clean(path)
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		path2 = filepath.Join(from.Path, path2)
		return lookupPackage(from.RepoPath, path2, rootScope, loc, lmap, log)
	}
	base := os.Getenv("FYRBASE")
	if base != "" {
		base = filepath.Clean(base)
		if p, err := lookupPackage(filepath.Join(base, "src"), path2, rootScope, loc, lmap, log); err == nil {
			return p, nil
		}
	}
	repo := os.Getenv("FYRPATH")
	if repo != "" {
		repos := strings.Split(repo, string(filepath.ListSeparator))
		for _, repoPath := range repos {
			repoPath = filepath.Clean(repoPath)
			if p, err := lookupPackage(repoPath, path2, rootScope, loc, lmap, log); err == nil {
				return p, nil
			}
		}
	}
	return nil, log.AddError(errlog.ErrorPackageNotFound, loc, path)
}

func lookupPackage(repoPath string, path string, rootScope *Scope, loc errlog.LocationRange, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
	dir := filepath.Join(repoPath, path)
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, os.ErrNotExist
	}
	f := errlog.NewSourceFile(repoPath)
	fileNumber := lmap.AddFile(f)
	ploc := errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0)
	return newPackage(repoPath, path, rootScope, ploc), nil
}
