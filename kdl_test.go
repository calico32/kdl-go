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

// isAcceptableMismatch checks for specific known mismatches between expected
// and actual output for certain test cases, where the mismatch is due to
// differences in formatting that are not tracked by the parser. This is a bit
// of a hack to avoid having to add special emitter options or tracking just for
// these cases, and should be used sparingly.
func isAcceptableMismatch(caseName string, version kdl.Version, actual string) bool {
	// There's no good way to output the correct document for this case for the reasons
	// mentioned above, so we just check for a specific output. It's equivalent to
	// the expected output, but with the numbers formatted differently.
	if caseName == "parse_all_arg_types.kdl" && version == kdl.Version1 {
		// True expected: `node 1 1.0 1.0E+10 1.0E-10 0x1 0o7 0b10 "arg" "arg\\\\" true false null`
		return actual == `node 1 1.0 1.0E+10 1.0E-10 1 7 2 "arg" "arg\\\\" true false null`
	}

	// Escaping of / is allowed in v1 but not required. We avoid escaping it
	// because it makes the emitted KDL more readable, but that means we have to
	// allow the mismatch in this test case if we don't want to add a special
	// emitter option just for this one case.
	if caseName == "all_escapes.kdl" && version == kdl.Version1 {
		// True expected: `node "\"\\\/\b\f\n\r\t"`
		return actual == `node "\"\\/\b\f\n\r\t"`
	}

	return false
}

func TestNodeEndLocation(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantEnd int // byte offset of EndLocation
	}{
		{"name only", "mynode\n", len("mynode")},
		{"string arg", `node "val"` + "\n", len(`node "val"`)},
		{"number arg", "node 42\n", len("node 42")},
		{"bool arg", "node #true\n", len("node #true")},
		{"prop", "node k=42\n", len("node k=42")},
		{"children", "node {\n    child\n}\n", len("node {\n    child\n}")},
		{"inline children", "node { child }\n", len("node { child }")},
		{"slashed arg last", `node "real" /- "slashed"` + "\n", len(`node "real" /- "slashed"`)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := kdl.Parse(strings.NewReader(tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(doc.Nodes) == 0 {
				t.Fatal("no nodes")
			}
			node := doc.Nodes[0]
			got := int(node.EndLocation().Offset)
			if got != tc.wantEnd {
				t.Errorf("EndLocation().Offset = %d, want %d (src: %q)", got, tc.wantEnd, tc.src)
			}
		})
	}
}

func TestValueEndLocation(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantEnd int // byte offset of EndLocation
	}{
		{"string", `node "val"` + "\n", len(`node "val"`)},
		{"number", "node 42\n", len("node 42")},
		{"bool", "node #true\n", len("node #true")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := kdl.Parse(strings.NewReader(tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(doc.Nodes) == 0 {
				t.Fatal("no nodes")
			}
			node := doc.Nodes[0]
			if len(node.Arguments()) == 0 {
				t.Fatal("no arguments")
			}
			value := node.Arg(0)
			got := int(value.EndLocation().Offset)
			if got != tc.wantEnd {
				t.Errorf("EndLocation().Offset = %d, want %d (src: %q)", got, tc.wantEnd, tc.src)
			}
		})
	}
}

// TestFormatRoundtrip asserts that for every parseable input in the v2
// conformance suite, Parse -> Format -> Parse yields a document that is
// canonically equivalent to the original.
func TestFormatRoundtrip(t *testing.T) {
	dir := "kdl2/tests/test_cases/input"
	cases, err := test.Kdl2Tests.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, caseFile := range cases {
		name := caseFile.Name()
		if strings.HasSuffix(name, "_fail.kdl") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f, err := test.Kdl2Tests.Open(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			src, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			doc1, err := kdl.Parse(strings.NewReader(string(src)), kdl.WithVersion(kdl.Version2))
			if err != nil {
				// Input doesn't parse cleanly; skip rather than fail — the
				// conformance suite contains inputs without matching
				// expected_kdl that this test isn't meant to evaluate.
				t.Skipf("input does not parse: %v", err)
				return
			}

			formatted := new(bytes.Buffer)
			if err := kdl.Format(doc1, formatted); err != nil {
				t.Fatalf("format: %v", err)
			}

			doc2, err := kdl.Parse(strings.NewReader(formatted.String()), kdl.WithVersion(kdl.Version2))
			if err != nil {
				t.Fatalf("re-parse of formatted output failed: %v\nFormatted:\n%s", err, formatted.String())
			}

			canon := func(d *kdl.Document) string {
				var b strings.Builder
				if err := kdl.Emit(d, &b, kdl.WithVersion(kdl.Version2), kdl.WithTestSuiteFloatOptions()); err != nil {
					t.Fatalf("emit: %v", err)
				}
				return b.String()
			}
			before, after := canon(doc1), canon(doc2)
			if before != after {
				t.Errorf("round-trip changed canonical output\nBefore:\n%s\nAfter:\n%s\nFormatted source:\n%s", before, after, formatted.String())
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
