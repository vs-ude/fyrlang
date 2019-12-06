package c99

import (
	commonConfig "github.com/vs-ude/fyrlang/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type (
	// Config contains the complete configuration of the c99 backend.
	Config struct {
		Compiler           *CompilerConf
		Archiver           *ArchiverConf
		PlatformHosted     bool
		IntSizeBit         int
		PointerSizeBit     int
		PackageReplacement map[string]string `json:",omitempty"`
		configTriplet      string
		memConf            *memoryConf
	}

	// CompilerConf contains the configuration of the c99 compiler.
	CompilerConf struct {
		Bin           string
		RequiredFlags string
		DebugFlags    string
		ReleaseFlags  string
	}

	// ArchiverConf contains the configuration of the archiver.
	ArchiverConf struct {
		Bin   string
		Flags string
	}

	// memoryConf is used to determine the memory allocation behavior.
	memoryConf struct {
		libsPath  string
		arch      string
		platform  string
		algorithm string
	}
)

// Name prints the name of the backend.
func (c *Config) Name() string {
	return "c99"
}

// Default configures the backend with the default values.
func (c *Config) Default() {
	c.Compiler = &CompilerConf{Bin: "gcc", RequiredFlags: "-D_FORTIFY_SOURCE=0", DebugFlags: "-g", ReleaseFlags: "-O3"}
	c.Archiver = &ArchiverConf{Bin: "ar", Flags: "rcs"}
	c.PlatformHosted = true
	c.IntSizeBit = 32
	c.PointerSizeBit = 64
	c.PackageReplacement = make(map[string]string, 1) // we don't expect to inject a lot of replacements inside the compiler
	c.PackageReplacement["platform"] = getArchAlias(runtime.GOARCH) + "-" + getPlatformAlias(runtime.GOOS)
	c.configTriplet = "x86_64-default"
	c.setupMemConfig()
}

// CheckConfig checks the validity of the loaded configuration and returns warnings and errors.
func (c *Config) CheckConfig() (warnings []string, err error) {
	err = nil
	if c.isGccOrClang() && !strings.Contains(c.Compiler.RequiredFlags, "-D_FORTIFY_SOURCE=0") {
		warnings = append(warnings, "GCC and Clang should be run with -D_FORTIFY_SOURCE=0!")
	}
	return
}

func (c *Config) isGccOrClang() bool {
	if strings.Contains(c.Compiler.Bin, "gcc") || strings.Contains(c.Compiler.Bin, "clang") {
		return true
	}
	return false
}

func (c *Config) setupMemConfig() {
	var libsPath, arch, platform, algorithm string

	if p, isSet := os.LookupEnv("FYRLIB_NATIVE"); isSet {
		libsPath = p
	} else {
		libsPath = filepath.Join(commonConfig.FyrBase(), "lib", "native")
	}
	switch strings.Split(c.configTriplet, "-")[0] {
	case "x86_64", "amd64":
		arch = "x86_64"
	default:
		arch = "none"
	}
	switch {
	case strings.Contains(c.configTriplet, "linux"):
		platform = "linux"
	case strings.Contains(c.configTriplet, "darwin"):
		platform = "darwin"
	default:
		platform = runtime.GOOS
	}
	if v, isSet := os.LookupEnv("FYR_NATIVE_MALLOC"); isSet {
		algorithm = v
	} else {
		algorithm = ""
	}

	c.memConf = &memoryConf{
		libsPath:  libsPath,
		arch:      arch,
		platform:  platform,
		algorithm: algorithm,
	}
}

// GetConfigName tries to automatically determine the correct configuration for a given compiler.
// Only works with gcc/clang.
func getConfigName(compilerPath string) string {
	compilerBinary := filepath.Base(compilerPath)
	if project := getCompilerProject(compilerBinary); project == "gcc" || project == "clang" {
		res, err := exec.Command(compilerPath, "-dumpmachine").Output()
		if err == nil {
			triplet := strings.Trim(string(res), "\t\n ")
			splitTriplet := strings.SplitN(triplet, "-", 2)
			return getArchAlias(splitTriplet[0]) + "-" +
				getPlatformAlias(splitTriplet[1]) + "-" +
				getCompilerProject(compilerPath) + ".json"
		}
		panic(err)
	} else {
		panic("Autodetection of architecture configuration only works with gcc or clang.")
	}
}

func getCompilerProject(compilerPath string) string {
	compilerBinary := filepath.Base(compilerPath)
	if strings.Contains(compilerBinary, "gcc") {
		return "gcc"
	} else if strings.Contains(compilerBinary, "clang") {
		return "clang"
	}
	panic("Unable to match the compiler to a project (gcc/clang).")
}

func getArchAlias(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	default:
		return arch
	}
}

func getPlatformAlias(platform string) string {
	switch {
	case simpleRegexMatch(platform, `.*linux.*`) ||
		simpleRegexMatch(platform, `.*darwin.*`):
		return "any"
	default:
		return platform
	}
}

func simpleRegexMatch(input, exp string) bool {
	matched, _ := regexp.MatchString(exp, input)
	return matched
}
