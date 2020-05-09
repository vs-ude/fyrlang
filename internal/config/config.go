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
	FyrBase              string `json:"FYRBASE"`
	FyrPath              string `json:"FYRPATH"` // may be empty
	CacheDirPath         string
	ConfDirPath          string
	Verbose              bool   `json:"-"` // this field is governed by a run flag
	HardwareArchitecture string `json:"arch"`
	OperatingSystem      string `json:"os"`
	// Name of this configuration.
	// If omitted, a concatenation of `HardwareArchitecture` and `OperatingSystem` is used.
	Name string `json:"name"`
}

func init() {
	config.FyrBase = os.Getenv("FYRBASE")
	config.FyrPath = os.Getenv("FYRPATH")
	config.CacheDirPath = getFyrDirectory(os.UserCacheDir())
	config.ConfDirPath = getFyrDirectory(os.UserConfigDir())
	config.Verbose = false
	checkConfig()
}

// getFyrDirectory returns the Fyr-specific path of user/system directories if it can be determined.
func getFyrDirectory(path string, err error) string {
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "fyrlang")
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

// OperatingSystem ...
func OperatingSystem() string {
	return config.OperatingSystem
}

// HardwareArchitecture ...
func HardwareArchitecture() string {
	return config.HardwareArchitecture
}

// PlatformName ...
func PlatformName() string {
	if config.Name != "" {
		return config.Name
	}
	return config.HardwareArchitecture + "-" + config.OperatingSystem
}

// EncodedPlatformName returns PlatformName but encoded in a way that it can be used as a filename.
func EncodedPlatformName() string {
	str := PlatformName()
	strings.ReplaceAll(str, "/", "_")
	// TODO: Encode other special characters
	return str
}

// PrintConf prints the current Fyr configuration to stdout in JSON format.
func PrintConf() {
	prettyConf, _ := json.MarshalIndent(config, "", "    ")
	fmt.Println(string(prettyConf))
}

// Set common configuration options. Be careful to use the correct types as value!
func Set(name string, value interface{}) {
	switch name {
	case "verbose":
		config.Verbose = value.(bool)
	}
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
