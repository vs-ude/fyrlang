package c99

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

type malloc struct {
	name,
	prefix string
	linker,
	linkOpts []string
}

func getAlgorithm(name string) (*malloc, error) {
	switch name {
	case "jemalloc":
		return &malloc{
			name:     "jemalloc",
			prefix:   "je_",
			linker:   nil,
			linkOpts: []string{"-lpthread"},
		}, nil
	case "tcmalloc":
		return &malloc{
			name:     "tcmalloc",
			prefix:   "tc_",
			linker:   []string{"gcc", "g++"},
			linkOpts: []string{"-lpthread"},
		}, nil
	default:
		return nil, errors.New("unknown memory allocator")
	}
}

func includeMallocHeader(args []string, config Config) []string {
	if &config.memConf.algorithm != nil {
		return append(args, []string{
			"-I" + filepath.Join(config.memConf.libsPath, "include", config.memConf.algorithm.name),
			"-include",
			"malloc.h",
		}...)
	}
	return args
}

func appendMallocOptions(archives []string, config Config) []string {
	var opts []string
	if &config.memConf.algorithm != nil {
		opts = []string{filepath.Join(
			config.memConf.libsPath,
			"lib",
			config.memConf.algorithm.name,
			config.memConf.platform+"-"+config.memConf.arch,
			"lib"+config.memConf.algorithm.name+".a",
		)}
	}
	opts = append(opts, config.memConf.algorithm.linkOpts...)
	return append(archives, opts...)
}

// getLinker tries to rewrite the provided compiler to point to a valid c++ compiler if the memory allocator requires it.
func getLinker(linker string, config Config) string {
	if config.memConf.algorithm != nil && config.memConf.algorithm.linker != nil {
		// TODO: this might be a bad idea. For prod it might be better to also require the platform conf to contain a c++ compiler.
		linkerBin := filepath.Base(linker)
		for i := 0; i < len(config.memConf.algorithm.linker); i += 2 {
			linkerBin = strings.ReplaceAll(linkerBin, config.memConf.algorithm.linker[i], config.memConf.algorithm.linker[i+1])
		}
		linker = filepath.Join(filepath.Dir(linker), linkerBin)
		if _, err := exec.LookPath(linker); err != nil {
			panic("The " + config.memConf.algorithm.name + " memory allocator requires a c++ compiler, but it could not be found at " + linker)
		}
	}
	return linker
}
