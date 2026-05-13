package kdl_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/calico32/kdl-go"
)

// diagCheck is a partial matcher for a single error-severity Diagnostic.
// Zero-valued fields are not checked.
type diagCheck struct {
	msgContain string
	startLine  int
	startCol   int
}

func (dc diagCheck) match(t *testing.T, d kdl.Diagnostic, label string) {
	t.Helper()
	if dc.msgContain != "" && !strings.Contains(d.Message, dc.msgContain) {
		t.Errorf("%s: message %q does not contain %q", label, d.Message, dc.msgContain)
	}
	if dc.startLine != 0 && d.Start.Line != dc.startLine {
		t.Errorf("%s: Start.Line = %d, want %d", label, d.Start.Line, dc.startLine)
	}
	if dc.startCol != 0 && d.Start.Column != dc.startCol {
		t.Errorf("%s: Start.Column = %d, want %d", label, d.Start.Column, dc.startCol)
	}
}

type recoveryCase struct {
	name          string
	input         string
	opts          []kdl.ParseOption
	wantErrCount  int         // min number of error-severity diagnostics expected
	wantDiags     []diagCheck // partial matchers for error-severity diagnostics, applied in order
	wantNodeNames []string    // expected top-level node names in order
}

var v2 = kdl.WithVersion(kdl.Version2)

func TestParserRecovery(t *testing.T) {
	cases := []recoveryCase{
		// valid cases
		{
			name:          "valid document produces no errors",
			input:         "node1 1\nnode2 2\n",
			wantNodeNames: []string{"node1", "node2"},
		},
		{
			name:          "single valid node",
			input:         "node 42\n",
			wantNodeNames: []string{"node"},
		},
		{
			name:          "empty document",
			input:         "",
			wantNodeNames: nil,
		},
		{
			name:          "comments only",
			input:         "// comment\n/* block */\n",
			wantNodeNames: nil,
		},
		{
			name:          "nested children, no errors",
			input:         "outer {\n  inner {\n    leaf 1\n  }\n}\n",
			wantNodeNames: []string{"outer"},
		},
		{
			name:          "multiple nodes with children",
			input:         "a {\n  x 1\n}\nb {\n  y 2\n}\n",
			wantNodeNames: []string{"a", "b"},
		},
		{
			name:          "node with properties and args",
			input:         "node \"hello\" key=42 flag=#true\n",
			wantNodeNames: []string{"node"},
		},

		// node-name errors → next sibling recovered
		{
			name:         "invalid char at node-name position, sibling recovered",
			input:        "[\nnode2 2\n",
			opts:         []kdl.ParseOption{v2},
			wantErrCount: 1,
			wantDiags: []diagCheck{
				{startLine: 1, startCol: 1},
			},
			wantNodeNames: []string{"node2"},
		},
		{
			name:          "two node-name errors, siblings recovered",
			input:         "[\nnode2 1\n[\nnode4 2\n",
			opts:          []kdl.ParseOption{v2},
			wantErrCount:  2,
			wantNodeNames: []string{"node2", "node4"},
		},
		{
			name:          "bad node name after a valid node",
			input:         "good 1\n[\nalsogood 2\n",
			opts:          []kdl.ParseOption{v2},
			wantErrCount:  1,
			wantNodeNames: []string{"good", "alsogood"},
		},

		// arg/prop errors → partial node, next sibling recovered
		{
			name:          "illegal token in arg position, node and sibling recovered",
			input:         "node1 [\nnode2 2\n",
			opts:          []kdl.ParseOption{v2},
			wantErrCount:  1, // lex error for [
			wantNodeNames: []string{"node1", "node2"},
		},

		// slashdash errors
		{
			// node slashdash
			name:         "slashdash at EOF",
			input:        "/-",
			opts:         []kdl.ParseOption{v2},
			wantErrCount: 1,
			wantDiags: []diagCheck{
				{msgContain: "string", startLine: 1},
			},
			wantNodeNames: nil,
		},
		{
			// inline slashdash
			name:         "trailing slashdash inside a node",
			input:        "node /-",
			opts:         []kdl.ParseOption{v2},
			wantErrCount: 1,
			wantDiags: []diagCheck{
				{msgContain: "slashdash", startLine: 1},
			},
			wantNodeNames: []string{"node"},
		},
		// missing closing brace
		{
			name:         "missing closing brace, parent node recovered",
			input:        "parent {\n  child 1\n",
			opts:         []kdl.ParseOption{v2},
			wantErrCount: 1,
			wantDiags: []diagCheck{
				{msgContain: "}"},
			},
			wantNodeNames: []string{"parent"},
		},

		// error in children
		{
			name:          "error inside children, outer siblings recovered",
			input:         "n1 1\nparent {\n  [\n  child 1\n}\nn3 3\n",
			opts:          []kdl.ParseOption{v2},
			wantErrCount:  1,
			wantNodeNames: []string{"n1", "parent", "n3"},
		},
		// diag ranges
		{
			// '[' appears at column 6 (after "node ")
			name:         "lex error range points at the bad character",
			input:        "node [\n",
			opts:         []kdl.ParseOption{v2},
			wantErrCount: 1,
			wantDiags: []diagCheck{
				{startLine: 1, startCol: 6},
			},
			wantNodeNames: []string{"node"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := kdl.ParseWithDiagnostics(strings.NewReader(tc.input), tc.opts...)
			if err != nil {
				t.Fatalf("ParseWithDiagnostics returned unexpected I/O error: %v", err)
			}
			if result == nil {
				t.Fatal("ParseWithDiagnostics returned nil result")
			}
			if result.Document == nil {
				t.Fatal("ParseResult.Document must be non-nil even when errors are present")
			}

			var errDiags []kdl.Diagnostic
			for _, d := range result.Diagnostics {
				if d.Severity == kdl.SeverityError {
					errDiags = append(errDiags, d)
				}
			}

			if len(errDiags) < tc.wantErrCount {
				t.Errorf("got %d error diagnostics, want at least %d", len(errDiags), tc.wantErrCount)
				for _, d := range result.Diagnostics {
					t.Logf("  [%d,%d] %s", d.Start.Line, d.Start.Column, d.Message)
				}
			}
			if tc.wantErrCount == 0 && len(errDiags) > 0 {
				t.Errorf("got %d unexpected error diagnostics:", len(errDiags))
				for _, d := range result.Diagnostics {
					t.Logf("  [%d,%d] %s", d.Start.Line, d.Start.Column, d.Message)
				}
			}

			for i, dc := range tc.wantDiags {
				if i >= len(errDiags) {
					t.Errorf("diagnostic[%d]: missing (only got %d error diagnostics)", i, len(errDiags))
					continue
				}
				dc.match(t, errDiags[i], fmt.Sprintf("diagnostic[%d]", i))
			}

			var gotNames []string
			for _, n := range result.Document.Nodes {
				gotNames = append(gotNames, n.Name())
			}
			if !stringSlicesEqual(gotNames, tc.wantNodeNames) {
				t.Errorf("top-level nodes = %v, want %v", gotNames, tc.wantNodeNames)
			}
		})
	}
}

// TestParserRecovery_ChildrenRecovered checks that children parsed before an
// error are still present on the recovered parent node.
func TestParserRecovery_ChildrenRecovered(t *testing.T) {
	input := "parent {\n  [\n  child 1\n}\n"
	result, err := kdl.ParseWithDiagnostics(strings.NewReader(input), v2)
	if err != nil {
		t.Fatalf("unexpected I/O error: %v", err)
	}
	if len(result.Document.Nodes) == 0 {
		t.Fatal("no top-level nodes recovered")
	}
	parent := result.Document.Nodes[0]
	if parent.Name() != "parent" {
		t.Fatalf("expected node named 'parent', got %q", parent.Name())
	}
	children := parent.Children()
	if children == nil {
		t.Fatal("parent.Children() is nil")
	}
	var found bool
	for _, c := range children.Nodes {
		if c.Name() == "child" {
			found = true
		}
	}
	if !found {
		var names []string
		for _, c := range children.Nodes {
			names = append(names, c.Name())
		}
		t.Errorf("'child' not found in parent.Children(); got: %v", names)
	}
}

// TestParserRecovery_MissingBraceChildren checks that already-parsed children
// are attached to the parent even when the closing brace is missing.
func TestParserRecovery_MissingBraceChildren(t *testing.T) {
	input := "parent {\n  child 1\n"
	result, err := kdl.ParseWithDiagnostics(strings.NewReader(input), v2)
	if err != nil {
		t.Fatalf("unexpected I/O error: %v", err)
	}
	if !result.HasErrors() {
		t.Fatal("expected HasErrors() == true for missing }")
	}
	if len(result.Document.Nodes) == 0 {
		t.Fatal("no nodes recovered")
	}
	parent := result.Document.Nodes[0]
	if parent.Children() == nil || len(parent.Children().Nodes) == 0 {
		t.Error("children not recovered despite missing }")
	}
}

// TestParserRecovery_DocumentNonNilOnError checks that Document is always
// non-nil from ParseWithDiagnostics, even on totally invalid input.
func TestParserRecovery_DocumentNonNilOnError(t *testing.T) {
	result, err := kdl.ParseWithDiagnostics(strings.NewReader("[\nnode 1\n"), v2)
	if err != nil {
		t.Fatalf("unexpected I/O error: %v", err)
	}
	if result.Document == nil {
		t.Fatal("Document must be non-nil even on error")
	}
	if !result.HasErrors() {
		t.Error("expected HasErrors() == true")
	}
}

// TestParserRecovery_AllDiagnosticsHavePositions checks that every diagnostic
// has a non-zero start line and column.
func TestParserRecovery_AllDiagnosticsHavePositions(t *testing.T) {
	inputs := []string{
		"[\n",
		"node [\n",
		"node {\n  [\n}\n",
	}
	for _, input := range inputs {
		result, err := kdl.ParseWithDiagnostics(strings.NewReader(input), v2)
		if err != nil {
			t.Fatalf("I/O error for %q: %v", input, err)
		}
		for i, d := range result.Diagnostics {
			if d.Start.Line == 0 {
				t.Errorf("input %q diagnostic[%d]: Start.Line == 0", input, i)
			}
			if d.Start.Column == 0 {
				t.Errorf("input %q diagnostic[%d]: Start.Column == 0", input, i)
			}
		}
	}
}

// TestParserRecovery_EndPositionAfterStart checks that End >= Start for all
// diagnostics (i.e. the range is non-degenerate or at least a point).
func TestParserRecovery_EndPositionAfterStart(t *testing.T) {
	result, err := kdl.ParseWithDiagnostics(strings.NewReader("[\nnode [\nparent {\n  [\n}\n"), v2)
	if err != nil {
		t.Fatalf("unexpected I/O error: %v", err)
	}
	for i, d := range result.Diagnostics {
		if d.End.Offset < d.Start.Offset {
			t.Errorf("diagnostic[%d]: End.Offset (%d) < Start.Offset (%d)", i, d.End.Offset, d.Start.Offset)
		}
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
