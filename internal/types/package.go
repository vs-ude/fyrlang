package types

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/vs-ude/fyrlang/internal/config"
	"github.com/vs-ude/fyrlang/internal/errlog"
	"github.com/vs-ude/fyrlang/internal/parser"
)

// Package ...
type Package struct {
	// The package scope holds types, functions and global variables.
	Scope    *Scope
	Path     string
	RepoPath string
	Imports  []*Package
	// List of all functions defined in this package.
	// This list is used by the code generation to generate code for all functions.
	// This includes non-exported functions, member functions and instantiated template functions.
	// Instantiated template functions are listed even for those template types defined in an imported package.
	Funcs    []*Func
	InitFunc *Func
	// All expressions which initially assign to global variables.
	VarExpressions []*parser.VarExpressionNode
	// This variable is used to detect circular dependencies
	parsed bool
	// 1 means yes, -1 means no, 0 means the value needs to be computed
	inFyrPath int
	// 1 means that the package is in the Fyr base repository as specified by the FYRBASE environment variable.
	// -1 means no, 0 means the value needs to be computed
	inFyrBase            int
	genericTypeInstances map[string]*GenericInstanceType
	genericFuncInstances map[string]*Func
}

// List of packages that are either parsed or imported.
// This is used to avoid loading a package twice.
var packages = make(map[string]*Package)

func newPackage(repoPath string, path string, rootScope *Scope, loc errlog.LocationRange) *Package {
	s := newScope(rootScope, PackageScope, loc)
	p := &Package{RepoPath: repoPath, Path: path, Scope: s}
	p.genericTypeInstances = make(map[string]*GenericInstanceType)
	p.genericFuncInstances = make(map[string]*Func)
	s.Package = p
	// Create init function
	ft := &FuncType{TypeBase: TypeBase{name: "__init__", pkg: p}, Out: &ParameterList{}, In: &ParameterList{}}
	p.InitFunc = &Func{name: "__init__", Type: ft, OuterScope: s, Location: loc}
	p.InitFunc.InnerScope = newScope(s, FunctionScope, loc)
	p.InitFunc.InnerScope.Func = p.InitFunc
	p.Funcs = append(p.Funcs, p.InitFunc)
	// Remember this package in a global variable such that it is not created twice
	dir := filepath.Join(p.RepoPath, p.Path)
	packages[dir] = p
	return p
}

// NewPackage ...
func NewPackage(srcPath string, rootScope *Scope, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
	fullSrcPath := srcPath
	if !filepath.IsAbs(srcPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		fullSrcPath = filepath.Join(wd, srcPath)
	}
	repoPath := filepath.Dir(fullSrcPath)
	path := filepath.Base(fullSrcPath)
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
	return newPackage(repoPath, path, rootScope, ploc), nil
}

// Adds `p` to the list of imported packages.
// The function avoids duplicate entries in `Imports`.
func (pkg *Package) addImport(p *Package) {
	for _, imp := range p.Imports {
		if imp == p {
			return
		}
	}
	pkg.Imports = append(pkg.Imports, p)
}

// RuntimePackage ...
func (pkg *Package) RuntimePackage() *Package {
	// The package itself is the runtime?
	if pkg.FullPath() == filepath.Join(config.FyrBase(), "lib/runtime") {
		return pkg
	}
	// Search in the imports for the runtime.
	for _, imp := range pkg.Imports {
		if imp.FullPath() == filepath.Join(config.FyrBase(), "lib/runtime") {
			return imp
		}
	}
	return nil
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
		println("---------------")
	}
	pkg.parsed = true
	return nil
}

// IsExecutable ...
func (pkg *Package) IsExecutable() bool {
	return pkg.MainFunc() != nil
}

// MainFunc ...
func (pkg *Package) MainFunc() *Func {
	el := pkg.Scope.HasElement("Main")
	if el == nil {
		return nil
	}
	f, ok := el.(*Func)
	if !ok {
		return nil
	}
	if f.Type.Target != nil {
		return nil
	}
	return f
}

// Variables ...
func (pkg *Package) Variables() []*Variable {
	var vars []*Variable
	for _, e := range pkg.Scope.Elements {
		if v, ok := e.(*Variable); ok {
			vars = append(vars, v)
		}
	}
	return vars
}

// FullPath returns the absolute file systems path to the package's directory.
func (pkg *Package) FullPath() string {
	return filepath.Join(pkg.RepoPath, pkg.Path)
}

// IsInFyrPath returns true if the package is located in a repository mentioned in FYRPATH.
func (pkg *Package) IsInFyrPath() bool {
	if pkg.inFyrPath != 0 {
		return pkg.inFyrPath == 1
	}
	repo := config.FyrPath()
	if repo != "" {
		repos := strings.Split(repo, string(filepath.ListSeparator))
		for _, repoPath := range repos {
			r, err := filepath.Rel(repoPath, pkg.FullPath())
			if err == nil && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
				pkg.inFyrPath = 1
				return true
			}
		}
	}
	pkg.inFyrPath = -1
	return false
}

// IsInFyrBase ...
func (pkg *Package) IsInFyrBase() bool {
	if pkg.inFyrBase != 0 {
		return pkg.inFyrBase == 1
	}
	base := config.FyrBase()
	if base != "" {
		base = filepath.Join(base, "lib")
		r, err := filepath.Rel(base, pkg.FullPath())
		if err == nil && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			pkg.inFyrBase = 1
			return true
		}
	}
	pkg.inFyrBase = -1
	return false
}

func (pkg *Package) lookupGenericInstanceType(typesig string) (*GenericInstanceType, bool) {
	inst, ok := pkg.genericTypeInstances[typesig]
	if ok {
		return inst, true
	}
	return nil, false
}

func (pkg *Package) registerGenericInstanceType(typesig string, t *GenericInstanceType) {
	pkg.genericTypeInstances[typesig] = t
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
	base := config.FyrBase()
	if base != "" {
		base = filepath.Clean(base)
		if p, err := lookupPackage(filepath.Join(base, "lib"), path2, rootScope, loc, lmap, log); err == nil {
			p.inFyrPath = 1
			return p, nil
		}
	}
	repo := config.FyrPath()
	if repo != "" {
		repos := strings.Split(repo, string(filepath.ListSeparator))
		for _, repoPath := range repos {
			repoPath = filepath.Clean(repoPath)
			if p, err := lookupPackage(filepath.Join(repoPath, "src"), path2, rootScope, loc, lmap, log); err == nil {
				p.inFyrPath = 1
				return p, nil
			}
		}
	}
	return nil, log.AddError(errlog.ErrorPackageNotFound, loc, path)
}

func lookupPackage(repoPath string, path string, rootScope *Scope, loc errlog.LocationRange, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
	dir := filepath.Join(repoPath, path)
	if p, ok := packages[dir]; ok {
		if !p.parsed {
			return nil, log.AddError(errlog.ErrorCircularImport, loc, dir)
		}
		return p, nil
	}
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
