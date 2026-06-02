package kdl_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
	"github.com/calico32/kdl-go/internal/test"
)

// FuzzParse asserts that ParseWithDiagnostics never panics on any input,
// seeding from the upstream KDL test suite and a few hand-picked edge cases. It
// also asserts that the returned Document is non-nil, even on invalid input.
func FuzzParse(f *testing.F) {
	seedCorpus(f, test.Kdl1Tests, "kdl1/tests/test_cases")
	seedCorpus(f, test.Kdl2Tests, "kdl2/tests/test_cases")

	f.Add("")
	f.Add("\x00")
	f.Add("node \"unterminated")
	f.Add("(type")
	f.Add("/*")
	f.Add("node {")
	f.Add("/- kdl-version 1\nnode")
	f.Add("/- kdl-version 2\nnode")

	f.Fuzz(func(t *testing.T, src string) {
		result, err := kdl.ParseWithDiagnostics(strings.NewReader(src))
		if err != nil {
			t.Fatalf("ParseWithDiagnostics returned error: %v", err)
		}
		if result == nil {
			t.Fatal("ParseWithDiagnostics returned nil result")
		}
		if result.Document == nil {
			t.Fatal("ParseWithDiagnostics returned nil Document")
		}
	})
}

func seedCorpus(f *testing.F, fsys fs.ReadDirFS, root string) {
	f.Helper()
	_ = fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".kdl") {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil
		}
		f.Add(string(data))
		return nil
	})
}
