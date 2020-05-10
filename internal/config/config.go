package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// config holds the Fyr compiler configuration. It is initialized
// by init() and cannot be accessed in other packages.
// Please use the exported functions below to access the configuration values.
var config struct {
	FyrBase              string
	FyrPath              string
	CacheDirPath         string
	ConfDirPath          string
	Verbose              bool
	HardwareArchitecture string
	OperatingSystem      string
	Flash                string
	// Name of the build-target
	BuildTargetName   string
	BuildTargetConfig *BuildTargetConfig
}

func init() {
	config.FyrBase = os.Getenv("FYRBASE")
	config.FyrPath = os.Getenv("FYRPATH")
	config.CacheDirPath = getFyrDirectory(os.UserCacheDir())
	config.ConfDirPath = getFyrDirectory(os.UserConfigDir())
	config.Verbose = false

	// Infer FYRBASE if not specified
	if config.FyrBase == "" {
		_, b, _, _ := runtime.Caller(0)
		srcpath := filepath.Dir(b)
		basepath := filepath.Dir(filepath.Dir(srcpath))
		_, err := os.Stat(filepath.Join(basepath, "fyrc"))
		if err == nil {
			os.Setenv("FYRBASE", basepath)
			config.FyrBase = basepath
		} else {
			panic("FYRBASE must be set!")
		}
	}
	if config.OperatingSystem == "" {
		if runtime.GOOS == "linux" {
			config.OperatingSystem = "GNU/Linux"
		} else if runtime.GOOS == "darwin" {
			config.OperatingSystem = "Darwin"
		} else {
			// TODO: Use `uname` to find out
			panic("Unknown operating system")
		}
	}
	if config.HardwareArchitecture == "" {
		if runtime.GOARCH == "amd64" {
			config.HardwareArchitecture = "x86_64"
		} else {
			// TODO: Use `uname` to find out
			panic("Unknown hardware platform")
		}
	}
	config.BuildTargetName = config.HardwareArchitecture + "-" + config.OperatingSystem
}

// getFyrDirectory returns the Fyr-specific path of user/system directories if it can be determined.
func getFyrDirectory(path string, err error) string {
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "fyrlang")
}

// LoadBuildTarget ...
func LoadBuildTarget(name string, debug bool) error {
	// Override the default_
	if name != "" {
		config.BuildTargetName = name
	}
	if debug {
		config.BuildTargetName += "-debug"
	}
	println(name, config.BuildTargetName)
	f, err := locateBuildTarget()
	if err != nil {
		return err
	}
	b, err := loadBuildTarget(f)
	if err != nil {
		return err
	}
	config.BuildTargetConfig = b
	return nil
}

// BuildTarget ...
func BuildTarget() *BuildTargetConfig {
	return config.BuildTargetConfig
}

// FyrBase returns the path specified in the FYRBASE environment variable.
// It should usually point to the installation path of the Fyr library and runtime.
func FyrBase() string {
	return config.FyrBase
}

// FyrPath returns the path specified in the FYRPATH environment variable.
// Be aware that this might not be set, in which case the returned string is empty.
func FyrPath() string {
	return config.FyrPath
}

// CacheDirPath returns the path used for storing Fyr-specific cache files.
func CacheDirPath() string {
	return config.CacheDirPath
}

// ConfDirPath returns the path used for storing Fyr-specific configuration files.
func ConfDirPath() string {
	return config.ConfDirPath
}

// OperatingSystem used by the host computer
func OperatingSystem() string {
	return config.OperatingSystem
}

// HardwareArchitecture used by the host computer
func HardwareArchitecture() string {
	return config.HardwareArchitecture
}

// BuildTargetName ...
func BuildTargetName() string {
	return config.BuildTargetName
}

// EncodedPlatformName returns PlatformName but encoded in a way that it can be used as a filename.
func EncodedPlatformName() string {
	str := config.BuildTargetName
	str = strings.ReplaceAll(str, "/", "_")
	// TODO: Encode other special characters
	str = strings.ToLower(str)
	return str
}

// Flash returns a string that is relevant to the programmer when flashing a microcontroller.
// If not empty, the compiler will flash the program after compiling it.
func Flash() string {
	return config.Flash
}

// SetFlash ...
func SetFlash(options string) {
	config.Flash = options
}

// PrintConf prints the current Fyr configuration to stdout in JSON format.
func PrintConf() {
	prettyConf, _ := json.MarshalIndent(config, "", "    ")
	fmt.Println(string(prettyConf))
}

// SetVerbose ...
func SetVerbose() {
	config.Verbose = true
}

// Verbose returns the Verbose setting.
func Verbose() bool {
	return config.Verbose
}

func checkConfig() {
	// Infer FYRBASE if not specified
	if config.FyrBase == "" {
		_, b, _, _ := runtime.Caller(0)
		srcpath := filepath.Dir(b)
		basepath := filepath.Dir(filepath.Dir(srcpath))
		_, err := os.Stat(filepath.Join(basepath, "fyrc"))
		if err == nil {
			os.Setenv("FYRBASE", basepath)
			config.FyrBase = basepath
		} else {
			panic("FYRBASE must be set!")
		}
	}
	if config.OperatingSystem == "" {
		if runtime.GOOS == "linux" {
			config.OperatingSystem = "GNU/Linux"
		} else if runtime.GOOS == "darwin" {
			config.OperatingSystem = "Darwin"
		} else {
			// TODO: Use `uname` to find out
			panic("Unknown operating system")
		}
	}
	if config.HardwareArchitecture == "" {
		if runtime.GOARCH == "amd64" {
			config.HardwareArchitecture = "x86_64"
		} else {
			// TODO: Use `uname` to find out
			panic("Unknown hardware platform")
		}
	}
}
