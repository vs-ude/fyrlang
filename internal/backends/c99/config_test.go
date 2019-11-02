package c99

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetConfigNamePositive(t *testing.T) {
	// Given
	paths := []string{
		"gcc",
		"clang",
	}
	var expectedPrefix string
	if runtime.GOARCH == "amd64" {
		expectedPrefix = "x86_64-"
	} else {
		expectedPrefix = ""
	}

	for _, name := range paths {
		// When
		resultName := getConfigName(name)

		// Then
		if !(strings.HasPrefix(resultName, expectedPrefix) && strings.HasSuffix(resultName, "-"+name+".json")) {
			t.Errorf("getConfigName(" + name + ") returned a malformed result:\n" + resultName)
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

func TestCheckConfigPositive(t *testing.T) {
	// Given
	config := Config{}
	config.Default()

	// When
	warn, err := config.CheckConfig()

	// Then
	if warn != nil || err != nil {
		t.Errorf("CheckConfig() should not return warnings or errors on the default config.")
	}
}

func TestCheckConfigWarning(t *testing.T) {
	// Given
	config := Config{}
	config.Default()
	config.Compiler.RequiredFlags = "invalid"

	// When
	warn, err := config.CheckConfig()

	// Then
	if err != nil {
		t.Errorf("CheckConfig() should not return an error on soft issues in valid configurations.")
	}
	if warn == nil {
		t.Errorf("CheckConfig() should return warnings on soft issues in configurations.")
	}
}
