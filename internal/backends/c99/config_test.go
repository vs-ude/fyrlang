package c99

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigNamePositive(t *testing.T) {
	// Given
	paths := []string{
		"gcc",
		"clang",
	}

	for _, name := range paths {
		// When
		resultName := getConfigName(name)

		// Then
		resultPath := filepath.Join(os.Getenv("FYRBASE"), "configs", "backend", "c99", resultName)
		if _, err := os.Stat(resultPath); err != nil {
			t.Errorf("getConfigName(" + name + ") returned a nonexistent file:\n" + resultPath)
		}
	}
}

func TestGetConfigNameNegative(t *testing.T) {
	// Given
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The construction of a configuration name should have aborted for an invalid compiler!")
		}
	}()
	name := "ThisIsNotAValidGCCName"

	// When
	getConfigName(name)
}

func TestGetCompilerProjectPositive(t *testing.T) {
	// Given
	paths := []string{
		"gcc",
		"clang",
		"/usr/bin/gcc",
		"/usr/bin/clang",
		"arm-linux-gnueabi-gcc",
	}
	expectedResults := []string{
		"gcc",
		"clang",
		"gcc",
		"clang",
		"gcc",
	}

	for i, name := range paths {
		// When
		resultName := getCompilerProject(name)

		// Then
		if resultName != expectedResults[i] {
			t.Errorf("getCompilerProject(" + name + ") returned an incorrect project:\n" + resultName)
		}
	}
}

func TestGetGetCompilerProjectNegative(t *testing.T) {
	// Given
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("getCompilerProject() should only match supported compilers!")
		}
	}()
	name := "ThisIsNotAValidCompilerName"

	// When
	getCompilerProject(name)
}
