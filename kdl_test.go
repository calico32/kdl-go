package kdl_test

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calico32/kdl-go"
	"github.com/calico32/kdl-go/internal/test"
)

func TestKdl2Suite(t *testing.T) {
	testKdlSuite(test.Kdl2Tests, "kdl2/tests/test_cases", t)
}

func testKdlSuite(fs embed.FS, dir string, t *testing.T) {
	cases, err := fs.ReadDir(filepath.Join(dir, "input"))
	if err != nil {
		panic(err)
	}

	for _, caseFile := range cases {
		f, err := fs.Open(filepath.Join(dir, "input", caseFile.Name()))
		if err != nil {
			panic(err)
		}
		input, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		t.Run(caseFile.Name(), func(t *testing.T) {
			t.Parallel()
			// fmt.Printf("TEST %s\n", caseFile.Name())
			doc, err := timeout(5*time.Second, func() (*kdl.Document, error) {
				return kdl.Parse(strings.NewReader(string(input)))
			})

			expected, errExpected := fs.ReadFile(filepath.Join(dir, "expected_kdl", caseFile.Name()))
			shouldFail := strings.HasSuffix(caseFile.Name(), "_fail.kdl") || errors.Is(errExpected, os.ErrNotExist)
			if shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, but got none\nInput:\n%s", caseFile.Name(), input)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Error parsing %s: \nInput:\n%s\n%+v", caseFile.Name(), input, err)
				return
			}

			opts := []kdl.EmitterOption{
				kdl.WithTestSuiteFloatOptions(),
			}
			// Special case: this test case expects no decimal point when emitting.
			// even though the rest of the test suite expects decimal points.
			if caseFile.Name() == "no_decimal_exponent.kdl" {
				opts = append(opts, kdl.WithFloatDecimalPoint(false))
			}
			actual := new(bytes.Buffer)
			err = kdl.Emit(doc, actual, opts...)
			if err != nil {
				t.Errorf("Error emitting %s: %+v", caseFile.Name(), err)
				return
			}

			if strings.TrimSpace(actual.String()) != strings.TrimSpace(string(expected)) {
				t.Errorf("Mismatch for %s:\nInput:\n%s\nExpected:\n%s\nGot:\n%s", caseFile.Name(), input, expected, actual.String())
			}
		})
	}
}

func timeout[T any](duration time.Duration, f func() (T, error)) (result T, err error) {
	done := make(chan struct{})
	go func() {
		result, err = f()
		close(done)
	}()
	select {
	case <-done:
		return result, err
	case <-time.After(duration):
		var zero T
		return zero, errors.New("operation timed out")
	}
}
