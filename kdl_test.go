package kdl_test

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
)

//go:embed kdl1/tests/test_cases/*
var kdl1Tests embed.FS

//go:embed kdl2/tests/test_cases/*
var kdl2Tests embed.FS

func TestKdl1Suite(t *testing.T) {
	testKdlSuite(kdl.KdlVersion1, kdl1Tests, "kdl1/tests/test_cases", t)
}

func TestKdl2Suite(t *testing.T) {
	testKdlSuite(kdl.KdlVersion2, kdl2Tests, "kdl2/tests/test_cases", t)
}

func testKdlSuite(ver kdl.KdlVersion, fs embed.FS, dir string, t *testing.T) {

	cases, err := fs.ReadDir(filepath.Join(dir, "input"))
	if err != nil {
		panic(err)
	}

	for _, caseFile := range cases {
		input, err := fs.Open(filepath.Join(dir, "input", caseFile.Name()))
		if err != nil {
			panic(err)
		}
		t.Run(caseFile.Name(), func(t *testing.T) {
			fmt.Printf("TEST %s\n", caseFile.Name())
			debug := new(bytes.Buffer)
			p := kdl.NewParser(ver, input)
			p.SetDebug(debug)
			doc, err := p.ParseDocument()

			expected, errExpected := fs.ReadFile(filepath.Join(dir, "expected_kdl", caseFile.Name()))
			shouldFail := strings.HasSuffix(caseFile.Name(), "_fail.kdl") || errors.Is(errExpected, os.ErrNotExist)
			if shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none\n\n%s", caseFile.Name(), debug.String())
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Error parsing %s: %v\n\n%s", caseFile.Name(), err, debug.String())
				return
			}

			actual := new(bytes.Buffer)
			e := kdl.NewEmitter(ver, actual)
			err = e.EmitDocument(doc)
			if err != nil {
				t.Errorf("Error emitting %s: %v\n\n%s", caseFile.Name(), err, debug.String())
				return
			}

			if strings.TrimSpace(actual.String()) != strings.TrimSpace(string(expected)) {
				t.Errorf("Mismatch for %s:\nExpected:\n%s\nGot:\n%s\n\n%s", caseFile.Name(), expected, actual.String(), debug.String())
			}
		})
	}
}
