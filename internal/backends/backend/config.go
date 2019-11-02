package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vs-ude/fyrlang/internal/config"
)

// Config is the interface backends need to implement in order to be configurable.
type Config interface {
	Default()
	Name() string
	CheckConfig() ([]string, error)
	// TODO: add ListBackendConfigs() and add an appropriate command for it
}

// PrintConfig prints the given config in formatted JSON.
func PrintConfig(c Config) {
	conf, _ := json.MarshalIndent(c, "", "    ")
	fmt.Println(string(conf))
}

// LoadConfig checks the existence of the given configuration file and parses it into the provided config struct.
// If something prevents the correct loading of the config file, the process will panic() and print an explanation.
func LoadConfig(path string, c Config) {
	path = expandConfigPath(path, c)
	if file, err := os.Open(path); err != nil {
		panic("The backend configuration could not be opened!")
	} else {
		defer file.Close()
		readConfig(file, c)
	}
}

func readConfig(r io.Reader, c Config) {
	jsonParser := json.NewDecoder(r)
	if err := jsonParser.Decode(&c); err != nil {
		panic("The backend configuration file contains invalid JSON.")
	}
	warnings, err := c.CheckConfig()
	if warnings != nil {
		printWarnings(c, warnings)
	}
	if err != nil {
		panic("The backend configuration file contains errors.")
	}
}

// expandConfigPath checks if the file exists in $WORKDIR/, $CONFDIR/backend/`c.Name()`/, or $FYRBASE/configs/backend/`c.Name()`,
// in this order and returns the absolute path to it.
func expandConfigPath(p string, c Config) (absPath string) {
	checkPath := func(p string) (path string, err error) {
		path = p
		_, err = os.Stat(path)
		return
	}

	if _, err := os.Stat(p); err == nil {
		absPath, _ = filepath.Abs(p)
	} else if path, err := checkPath(filepath.Join(config.ConfDirPath(), "backend", c.Name(), p)); err == nil {
		absPath = path
	} else if path, err := checkPath(filepath.Join(config.FyrBase(), "configs", "backend", c.Name(), p)); err == nil {
		absPath = path
	} else {
		panic(`The backend configuration file could not be located.
		Please make sure you have provided a correct path or name for the file.
		If no file for your desired target exists, you can create one. Please consult the Fyr documentation on how to do this.
		`)
	}
	return
}

func printWarnings(c Config, warnings []string) {
	fmt.Println("WARNING: The", c.Name(), "configuration contains possible issues!")
	for _, warning := range warnings {
		fmt.Println("WARNING:", warning)
	}
}
