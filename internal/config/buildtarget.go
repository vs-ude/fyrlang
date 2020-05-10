package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

// BuildTargetConfig ...
type BuildTargetConfig struct {
	Name                 string          `json:"name"`
	HardwareArchitecture string          `json:"arch"`
	OperatingSystem      string          `json:"os"`
	C99                  *BuildTargetC99 `json:"c99"`
}

// BuildTargetC99 ...
type BuildTargetC99 struct {
	Compiler *BuildTargetC99Compiler `json:"compiler"`
	Archiver *BuildTargetC99Archiver `json:"archiver"`
	Linker   *BuildTargetC99Linker   `json:"linker"`
	Flash    *BuildTargetC99Flash    `json:"flash"`
}

// BuildTargetC99Compiler ...
type BuildTargetC99Compiler struct {
	Command string   `json:"command"`
	Flags   []string `json:"flags"`
}

// BuildTargetC99Archiver ...
type BuildTargetC99Archiver struct {
	Command string   `json:"command"`
	Flags   []string `json:"flags"`
}

// BuildTargetC99Linker ...
type BuildTargetC99Linker struct {
	Command string   `json:"command"`
	Flags   []string `json:"flags"`
}

// BuildTargetC99Flash ...
type BuildTargetC99Flash struct {
	Commands []*BuildTargetC99FlashCommand `json:"commands"`
}

// BuildTargetC99FlashCommand ...
type BuildTargetC99FlashCommand struct {
	Command string   `json:"command"`
	Flags   []string `json:"flags"`
}

func locateBuildTarget() (string, error) {
	filename := filepath.Join(config.FyrBase, "build_targets", EncodedPlatformName()+".json")
	_, err := os.Stat(filename)
	if err != nil {
		return "", errors.New("Failed to find built target file for target " + config.BuildTargetName + "\nLooking for " + EncodedPlatformName() + ".json")
	}
	return filename, nil
}

func loadBuildTarget(filename string) (*BuildTargetConfig, error) {
	cfg := &BuildTargetConfig{}
	stat, err := os.Stat(filename)
	if err != nil {
		return cfg, nil
	}
	if stat.IsDir() {
		return cfg, nil
	}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("Failed to read build target file: " + filename + "\n" + err.Error())
	}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, errors.New("Failed to parse build target file: " + filename + "\n" + err.Error())
	}
	cfg.Name = EncodedPlatformName()
	return cfg, nil
}
