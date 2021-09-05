package parser

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/vs-ude/fyrlang/internal/errlog"
)

func TestParser(t *testing.T) {
	lmap := errlog.NewLocationMap()
	log := errlog.NewErrorLog()
	err := filepath.Walk("test_data", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		p := NewParser(log)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		f := errlog.NewSourceFile(path)
		number := lmap.AddFile(f)
		_, err = p.Parse(number, string(data))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	if len(log.Errors) != 0 {
		for _, e := range log.Errors {
			t.Log(errlog.ErrorToString(e, lmap))
		}
		t.Fail()
	}
}
