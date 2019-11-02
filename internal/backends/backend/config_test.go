package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfig is a mock struct
type TestConfig struct {
	TestData1 string
	TestData2 bool
	TestData3 int
}

// Default mock function
func (c *TestConfig) Default() {}

// Name mock function
func (c *TestConfig) Name() string {
	return "c99"
}

// CheckConfig mock function
func (c *TestConfig) CheckConfig() (warnings []string, err error) { return }

func TestValidJSON(t *testing.T) {
	// Given
	testString := `
	{
		"TestData1": "bla",
		"TestData2": true,
		"TestData3": 4
	}
	`
	r := strings.NewReader(testString)
	c := &TestConfig{}

	// When
	readConfig(r, c)

	// Then
	if c.TestData1 != "bla" ||
		c.TestData2 != true ||
		c.TestData3 != 4 {
		t.Errorf("The TestConfig struct does not contain the expected values.")
	}
}

func TestInvalidJSON(t *testing.T) {
	// Given
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The parsing of invalid JSON did not panic")
		}
	}()

	testString := `
	{
		"TestData1": bla,
	}
	`
	r := strings.NewReader(testString)
	c := &TestConfig{}

	// When
	readConfig(r, c)

	// Then
}

func TestValidPath1(t *testing.T) {
	// Given
	testPath := "config_test.go"
	c := &TestConfig{}

	// When
	res := expandConfigPath(testPath, c)

	// Then
	_, err := os.Stat(res)
	if !filepath.IsAbs(res) || err != nil {
		t.Errorf("The returned path is invalid.")
	}
}

func TestValidPath2(t *testing.T) {
	// Given
	testPath := "x86_64-pc-linux-gnu-clang.json"
	c := &TestConfig{}

	// When
	res := expandConfigPath(testPath, c)

	// Then
	_, err := os.Stat(res)
	if !filepath.IsAbs(res) || err != nil {
		t.Errorf("The returned path is invalid.")
	}
}

func TestInvalidPath(t *testing.T) {
	// Given
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The expansion of an invalid path did not panic")
		}
	}()

	testPath := "thisIsAnInvalidPath"
	c := &TestConfig{}

	// When
	expandConfigPath(testPath, c)

	//Then
}
