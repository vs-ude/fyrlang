package types

import (
	"encoding/json"
	"errors"
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
	ComponentsUsed []*ComponentUsage
	// Might be nil.
	Config *PackageConfig
	// Files of the form `name_<sys>.fyr` are parsed only of `<sys>` is equivalent to systemLabel.
	systemLabel string
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

// PackageConfig represents the data stored in `package.json`.
type PackageConfig struct {
	Proxies []*PackageProxy `json:"proxies"`
	// Lists all targets for which this package can be used.
	// If empty, `Proxies` applies.
	// If empty and `Proxies` is empty as well, then the package can be used on every system
	Targets []*PackageTarget `json:"targets"`
}

// PackageSystem describes a system in `package.json`
type PackageSystem struct {
	Target string `json:"target"`
	OS     string `json:"os"`
	OSMin  string `json:"os-min"`
	OSMax  string `json:"os-max"`
	Arch   string `json:"arch"`
}

// PackageTarget selects a build target.
type PackageTarget struct {
	PackageSystem
	Label string `json:"label"`
}

// PackageProxy redirects to another package depending on the system as used by `package.json`
type PackageProxy struct {
	PackageSystem
	Path string `json:"path"`
}

var errPackageFilterDoesNotMatch = errors.New("Package filter does not match build target")

// List of packages that are either parsed or imported.
// This is used to avoid loading a package twice.
var packages = make(map[string]*Package)

func newPackage(repoPath string, path string, rootScope *Scope, cfg *PackageConfig, loc errlog.LocationRange) *Package {
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
	// Turn `srcPath` into an absolute path
	fullSrcPath := srcPath
	if !filepath.IsAbs(srcPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		fullSrcPath = filepath.Join(wd, srcPath)
	}
	// The directory part is the repoPath
	repoPath := filepath.Dir(fullSrcPath)
	// The last component of `fullSrcPath` is the package path
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
	// Search for `package.json`
	var cfg *PackageConfig
	cfg, err = parsePackageJSON(filepath.Join(fullSrcPath, "package.json"), log, lmap)
	if err != nil {
		return nil, err
	}
	redirect, label, err := matchPackageFilters(cfg)
	if err == errPackageFilterDoesNotMatch {
		return nil, log.AddError(errlog.ErrorPackageNotForTarget, errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0), repoPath)
	}
	if err != nil {
		panic("Oooops")
	}
	// Create the package
	ploc := errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0)
	p, err := newPackage(repoPath, path, rootScope, cfg, ploc), nil
	if err != nil {
		return nil, err
	}
	// If proxy only, find the targe-specific package instead
	if redirect != "" {
		p, err = LookupPackage(redirect, p, ploc, lmap, log)
		if err != nil {
			return nil, err
		}
	}
	p.systemLabel = label
	return p, err
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
	return pkg.parse(dir, lmap, log)
}

func (pkg *Package) parse(dir string, lmap *errlog.LocationMap, log *errlog.ErrorLog) error {
	fh, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer fh.Close()
	// Determine all *.fyr files
	stats, err := fh.Readdir(0)
	if err != nil {
		return err
	}
	var files []*file
	// Parse all *.fyr files
	var parseError error
	for _, stat := range stats {
		if stat.IsDir() {
			continue
		}
		if !strings.HasSuffix(stat.Name(), ".fyr") {
			continue
		}
		name := stat.Name()
		if idx := strings.IndexByte(name, '_'); idx >= 0 {
			label := name[idx+1 : len(name)-4]
			if label != pkg.systemLabel {
				println("SKIP", label, name)
				continue
			}
		}
		filePath := filepath.Join(dir, name)
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
		f := newFile(pkg, n, lmap, log)
		files = append(files, f)
		err = f.parseAndDeclare()
		if err != nil {
			if parseError == nil {
				parseError = err
			}
			continue
		}
		println("---------------")
	}
	if parseError != nil {
		return parseError
	}
	parseError = nil
	for _, f := range files {
		err = f.declareComponents()
		if err != nil && parseError == nil {
			parseError = err
		}
	}
	for _, f := range files {
		err = f.defineTypes()
		if err != nil && parseError == nil {
			parseError = err
		}
	}
	for _, f := range files {
		err = f.defineGlobalVars()
		if err != nil && parseError == nil {
			parseError = err
		}
	}
	for _, f := range files {
		err = f.defineComponents()
		if err != nil && parseError == nil {
			parseError = err
		}
	}
	for _, f := range files {
		err = f.defineFuns()
		if err != nil && parseError == nil {
			parseError = err
		}
	}
	// Check all templates instantiated on behalf of this package
	for _, g := range pkg.genericTypeInstances {
		err := checkFuncs(g, pkg, log)
		if err != nil {
			parseError = err
		}
	}
	pkg.parsed = true
	return parseError
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
	// A relative path?
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		path2 = filepath.Join(from.Path, path2)
		return lookupPackage(from.RepoPath, path2, rootScope, from, loc, lmap, log)
	}
	// The package is in FYRBASE?
	base := config.FyrBase()
	if base != "" {
		base = filepath.Clean(base)
		if p, err := lookupPackage(filepath.Join(base, "lib"), path2, rootScope, from, loc, lmap, log); err == nil {
			p.inFyrPath = 1
			return p, nil
		}
	}
	// The package is in a directory listed in FYRPATH?
	repo := config.FyrPath()
	if repo != "" {
		repos := strings.Split(repo, string(filepath.ListSeparator))
		for _, repoPath := range repos {
			repoPath = filepath.Clean(repoPath)
			if p, err := lookupPackage(filepath.Join(repoPath, "src"), path2, rootScope, from, loc, lmap, log); err == nil {
				p.inFyrPath = 1
				return p, nil
			}
		}
	}
	return nil, log.AddError(errlog.ErrorPackageNotFound, loc, path)
}

func lookupPackage(repoPath string, path string, rootScope *Scope, from *Package, loc errlog.LocationRange, lmap *errlog.LocationMap, log *errlog.ErrorLog) (*Package, error) {
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
	// Search for `package.json`
	var cfg *PackageConfig
	cfg, err = parsePackageJSON(filepath.Join(dir, "package.json"), log, lmap)
	if err != nil {
		return nil, err
	}
	redirect, label, err := matchPackageFilters(cfg)
	if err != nil {
		return nil, err
	}
	f := errlog.NewSourceFile(dir)
	fileNumber := lmap.AddFile(f)
	ploc := errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0)
	p := newPackage(repoPath, path, rootScope, cfg, ploc)
	if redirect != "" {
		// TODO: Avoid loops
		p, err = LookupPackage(redirect, p, ploc, lmap, log)
	}
	p.systemLabel = label
	return p, nil
}

func parsePackageJSON(filename string, log *errlog.ErrorLog, lmap *errlog.LocationMap) (*PackageConfig, error) {
	cfg := &PackageConfig{}
	stat, err := os.Stat(filename)
	if err != nil {
		return cfg, nil
	}
	if stat.IsDir() {
		return cfg, nil
	}
	f := errlog.NewSourceFile(filename)
	fileNumber := lmap.AddFile(f)
	loc := errlog.EncodeLocationRange(fileNumber, 0, 0, 0, 0)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, log.AddError(errlog.ErrorMalformedPackageConfig, loc, err.Error())
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, log.AddError(errlog.ErrorMalformedPackageConfig, loc, err.Error())
	}
	return cfg, nil
}

func matchPackageFilters(cfg *PackageConfig) (redirect string, label string, err error) {
	if cfg == nil {
		return
	}
	// Check whether the package can be used on this system
	for _, sys := range cfg.Targets {
		if matchPackageTarget(&sys.PackageSystem) {
			label = sys.Label
			return
		}
	}
	for _, r := range cfg.Proxies {
		if matchPackageTarget(&r.PackageSystem) {
			redirect = r.Path
			return
		}
	}
	if len(cfg.Targets) > 0 || len(cfg.Proxies) > 0 {
		// None of the specified systems matches. Does not work
		err = errPackageFilterDoesNotMatch
		return
	}
	// Nothing has been specified. Seems to be a universal package. Proceed.
	return
}

func matchPackageTarget(sys *PackageSystem) bool {
	if sys.Target != "" && sys.Target != config.BuildTarget().Name {
		return false
	}
	if sys.OS != "" && sys.OS != config.BuildTarget().OperatingSystem {
		return false
	}
	if sys.Arch != "" && sys.Arch != config.BuildTarget().HardwareArchitecture {
		return false
	}
	return true
}
