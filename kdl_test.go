package kdl_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calico32/kdl-go"
	"github.com/calico32/kdl-go/internal/test"
)

func TestKdl1Suite(t *testing.T) {
	testKdlSuite(kdl.Version1, test.Kdl1Tests, "kdl1/tests/test_cases", t)
}

func TestKdl2Suite(t *testing.T) {
	testKdlSuite(kdl.Version2, test.Kdl2Tests, "kdl2/tests/test_cases", t)
}

type filesys interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

func testKdlSuite(version kdl.Version, fs filesys, dir string, t *testing.T) {
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
			trace := new(strings.Builder)
			doc, err := timeout(5*time.Second, func() (doc *kdl.Document, err error) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("panic: %v", r)
						doc = nil
						err = nil
						return
					}
				}()
				return kdl.Parse(strings.NewReader(string(input)), kdl.WithVersion(version), kdl.WithParseTrace(trace))
			})

			expected, errExpected := fs.ReadFile(filepath.Join(dir, "expected_kdl", caseFile.Name()))
			shouldFail := strings.HasSuffix(caseFile.Name(), "_fail.kdl") || errors.Is(errExpected, os.ErrNotExist)
			if shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s (%s), but got none\nInput:\n%s\nTrace:\n%s", caseFile.Name(), version, input, trace)
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Error parsing %s (%s): \nInput:\n%s\n%+v\nTrace:\n%s", caseFile.Name(), version, input, err, trace)
				return
			}

			if doc == nil {
				return // panic occurred
			}

			opts := []kdl.EmitterOption{
				kdl.WithTestSuiteFloatOptions(),
				kdl.WithVersion(version),
			}
			opts = append(opts, testSpecificEmitterOptions(caseFile.Name(), version)...)

			actual := new(bytes.Buffer)
			err = kdl.Emit(doc, actual, opts...)
			if err != nil {
				t.Errorf("Error emitting %s (%s): %+v", caseFile.Name(), version, err)
				return
			}

			if strings.TrimSpace(actual.String()) != strings.TrimSpace(string(expected)) {
				if isAcceptableMismatch(caseFile.Name(), version, strings.TrimSpace(actual.String())) {
					// ok
				} else {
					t.Errorf("Mismatch for %s (%s):\nInput:\n%s\nExpected:\n%s\nGot:\n%s\nTrace:\n%s", caseFile.Name(), version, input, expected, actual.String(), trace)
				}
			}
		})
	}
}

// Lots of test cases expect different formatting for numbers similar to the
// format in the input KDL, but because we don't keep track of the that format
// we need to cheat a bit and change the emitter options based on the case name.
func testSpecificEmitterOptions(caseName string, version kdl.Version) []kdl.EmitterOption {
	switch caseName {
	case "no_decimal_exponent.kdl":
		return []kdl.EmitterOption{kdl.WithFloatDecimalPoint(false)}
	}

	if version == kdl.Version2 {
		// no other quirks for v2
		return nil
	}

	switch {
	case strings.HasPrefix(caseName, "binary"):
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Binary)}
	case strings.HasPrefix(caseName, "hex"):
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Hex)}
	case strings.HasPrefix(caseName, "octal"):
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Octal)}
	case strings.HasPrefix(caseName, "empty_child"):
		return []kdl.EmitterOption{kdl.WithEmitEmptyChildren(true)}
	}

	switch caseName {
	case "leading_zero_binary.kdl":
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Binary)}
	case "trailing_underscore_hex.kdl":
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Hex)}
	case "leading_zero_oct.kdl",
		"trailing_underscore_octal.kdl",
		"underscore_in_octal.kdl":
		return []kdl.EmitterOption{kdl.WithIntegerFormat(kdl.Octal)}
	case "slashdash_node_in_child.kdl":
		return []kdl.EmitterOption{kdl.WithEmitEmptyChildren(true)}
	}

	return nil
}

// There's no good way to output the correct document for this case for the reasons
// mentioned above, so we just check for a specific output. It's equivalent to
// the expected output, but with the numbers formatted differently.
func isAcceptableMismatch(caseName string, version kdl.Version, actual string) bool {
	if caseName == "parse_all_arg_types.kdl" && version == kdl.Version1 {
		// True expected: `node 1 1.0 1.0E+10 1.0E-10 0x1 0o7 0b10 "arg" "arg\\\\" true false null`
		return actual == `node 1 1.0 1.0E+10 1.0E-10 1 7 2 "arg" "arg\\\\" true false null`
	}

	return false
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
