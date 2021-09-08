package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vs-ude/fyrlang/internal/errlog"
)

func TestParser(t *testing.T) {
	err := filepath.Walk("test_data", func(path string, info os.FileInfo, err error) error {
		if path == "test_data" {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		lmap := errlog.NewLocationMap()
		log := errlog.NewErrorLog()
		gen := NewPackageGenerator(log, lmap)
		gen.Run([]string{path})
		if len(log.Errors) != 0 {
			for _, e := range log.Errors {
				t.Log(errlog.ErrorToString(e, lmap))
			}
			t.Fail()
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
