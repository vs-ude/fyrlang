package backend

import (
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
	return "TestConfig"
}

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
