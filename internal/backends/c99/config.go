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
	}

	// CompilerConf contains the configuration of the c99 compiler.
	CompilerConf struct {
		Bin          string
		DebugFlags   string
		ReleaseFlags string
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
	c.Compiler = &CompilerConf{Bin: "gcc", DebugFlags: "-g", ReleaseFlags: "-O3"}
	c.Archiver = &ArchiverConf{Bin: "ar", Flags: "rcs"}
	c.PlatformHosted = true
	c.IntSizeBit = 32
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
