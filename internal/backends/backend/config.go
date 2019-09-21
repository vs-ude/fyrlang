package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Config is the interface backends need to implement in order to be configurable.
type Config interface {
	Default()
	Name() string
}

// PrintConfig prints the given config in formatted JSON.
func PrintConfig(c Config) {
	conf, _ := json.MarshalIndent(c, "", "    ")
	fmt.Println(string(conf))
}

// LoadConfig checks the existence of the given configuration file and parses it into the provided config struct.
// If something prevents the correct loading of the config file, the process will panic() and print an explanation.
func LoadConfig(path string, c Config) {
	if filepath.IsAbs(path) {
		//noop
	} else {
		panic(`The backend configuration file could not be located.
		Please make sure you have provided a correct path or name for the file.
		If no file for your desired target exists, you can create one. Please consult the Fyr documentation on how to do this.
		`)
	}
	// TODO: check if the file exists in $WORKDIR/, $CONFDIR/backend/`c.Name()`/, or $FYRBASE/configs/backend/`c.Name()`, in this order and update it
	if file, err := os.Open(path); err != nil {
		panic("The backend configuration could not be opened!")
	} else {
		defer file.Close()
		readConfig(file, c)
	}
}

func readConfig(r io.Reader, c Config) {
	jsonParser := json.NewDecoder(r)
	if err := jsonParser.Decode(c); err != nil {
		panic("The backend configuration file contains invalid JSON.")
	}
}
