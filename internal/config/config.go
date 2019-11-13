package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// config holds the Fyr compiler configuration. It is initialized
// by init() and cannot be accessed in other packages.
// Please use the exported functions below to access the configuration values.
var config struct {
	FyrBase      string `json:"FYRBASE"`
	FyrPath      string `json:"FYRPATH"` // may be empty
	CacheDirPath string
	ConfDirPath  string
	Verbose      bool `json:"-"` // this field is governed by a run flag
}

func init() {
	config.FyrBase = os.Getenv("FYRBASE")
	config.FyrPath = os.Getenv("FYRPATH")
	config.CacheDirPath = getFyrDirectory(os.UserCacheDir())
	config.ConfDirPath = getFyrDirectory(os.UserConfigDir())
	config.Verbose = false
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
