package c99

import (
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	// Config contains the complete configuration of the c99 backend.
	Config struct {
		Compiler       *CompilerConf
		Archiver       *ArchiverConf
		PlatformHosted bool
		IntSizeBit     int
		PointerSizeBit int
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

// GetConfigName tries to automatically determine the correct configuration for a given compiler.
// Only works with gcc/clang.
// TODO: implement fuzzy matching by using `any` in the triplets
func getConfigName(compilerPath string) string {
	compilerBinary := filepath.Base(compilerPath)
	if strings.Contains(compilerBinary, "gcc") || strings.Contains(compilerBinary, "clang") {
		res, err := exec.Command(compilerPath, "-dumpmachine").Output()
		if err == nil {
			triplet := strings.Trim(string(res), "\t\n ")
			return triplet + "-" + getCompilerProject(compilerPath) + ".json"
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
