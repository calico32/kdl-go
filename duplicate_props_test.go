package kdl

import (
	"strings"
	"testing"
)

func TestDuplicateProps_PreservedInAST(t *testing.T) {
	doc := parseDoc(t, "node a=1 a=2 a=3\n")
	if len(doc.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(doc.Nodes))
	}
	n := doc.Nodes[0]

	entries := n.PropertyEntries()
	if len(entries) != 3 {
		t.Fatalf("PropertyEntries() len = %d, want 3", len(entries))
	}
	for i, want := range []int{1, 2, 3} {
		if got := entries[i].Value.Int(); got != want {
			t.Errorf("entries[%d].Value = %v, want %d", i, entries[i].Value, want)
		}
		if entries[i].Key != "a" {
			t.Errorf("entries[%d].Key = %q, want %q", i, entries[i].Key, "a")
		}
	}

	// Properties() is last-wins
	if v := n.Properties()["a"].Int(); v != 3 {
		t.Errorf("Properties()[a] = %v, want 3", n.Properties()["a"])
	}

	// PropertyOrder is unique
	if got := n.PropertyOrder(); len(got) != 1 || got[0] != "a" {
		t.Errorf("PropertyOrder() = %v, want [a]", got)
	}

	// Per-entry locations
	for i := range entries {
		start, end, ok := n.PropertyEntryKeyLocation(i)
		if !ok {
			t.Errorf("PropertyEntryKeyLocation(%d) ok=false, want true", i)
		}
		if start.Line == 0 || end.Line == 0 {
			t.Errorf("PropertyEntryKeyLocation(%d) zero loc: start=%v end=%v", i, start, end)
		}
	}
}

func TestDuplicateProps_FormatterPreservesAll(t *testing.T) {
	src := "node a=1 a=2 a=3\n"
	doc := parseDoc(t, src)
	got := mustFormat(t, doc)
	want := "node a=1 a=2 a=3\n"
	if got != want {
		t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDuplicateProps_FormatterInterleavedWithArgs(t *testing.T) {
	src := "node 1 a=1 2 a=2 3\n"
	checkFormat(t, src, "node 1 a=1 2 a=2 3\n")
}

func TestDuplicateProps_EmitterDropsEarlier(t *testing.T) {
	doc := parseDoc(t, "node a=1 a=2\n")
	got, err := EmitToString(doc)
	if err != nil {
		t.Fatalf("EmitToString: %v", err)
	}
	want := "node a=2\n"
	if got != want {
		t.Errorf("emit: got %q want %q", got, want)
	}
}

func TestDuplicateProps_DupWarn(t *testing.T) {
	result, err := ParseNamedWithDiagnostics("<test>",
		strings.NewReader("node a=1 a=2 a=3\n"),
		WithDuplicateProperties(DupWarn),
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var warns int
	for _, d := range result.Diagnostics {
		if d.Severity == SeverityWarning && strings.Contains(d.Message, "duplicate property") {
			warns++
		}
		if d.Severity == SeverityError {
			t.Errorf("unexpected error diagnostic: %s", d.Message)
		}
	}
	if warns != 2 {
		t.Errorf("got %d duplicate warnings, want 2", warns)
	}
}

func TestDuplicateProps_DupError(t *testing.T) {
	result, err := ParseNamedWithDiagnostics("<test>",
		strings.NewReader("node a=1 a=2\n"),
		WithDuplicateProperties(DupError),
	)
	if err != nil {
		t.Fatalf("ParseNamedWithDiagnostics IO error: %v", err)
	}
	if !result.HasErrors() {
		t.Fatal("want HasErrors()=true with DupError")
	}
	var dups int
	for _, d := range result.Diagnostics {
		if d.Severity == SeverityError && strings.Contains(d.Message, "duplicate property") {
			dups++
		}
	}
	if dups != 1 {
		t.Errorf("got %d duplicate errors, want 1", dups)
	}
	// AST still has both entries even with DupError
	if len(result.Document.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1", len(result.Document.Nodes))
	}
	if got := len(result.Document.Nodes[0].PropertyEntries()); got != 2 {
		t.Errorf("PropertyEntries len = %d, want 2", got)
	}
}

func TestDuplicateProps_DupAllowSilent(t *testing.T) {
	result, err := ParseNamedWithDiagnostics("<test>",
		strings.NewReader("node a=1 a=2\n"),
		WithDuplicateProperties(DupAllow),
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "duplicate") {
			t.Errorf("unexpected diagnostic with DupAllow: %+v", d)
		}
	}
}

func TestDuplicateProps_RemoveProperty(t *testing.T) {
	doc := parseDoc(t, "node a=1 b=2 a=3 c=4\n")
	n := doc.Nodes[0]
	n.RemoveProperty("a")
	entries := n.PropertyEntries()
	if len(entries) != 2 {
		t.Fatalf("PropertyEntries len = %d, want 2", len(entries))
	}
	wantKeys := []string{"b", "c"}
	for i, k := range wantKeys {
		if entries[i].Key != k {
			t.Errorf("entries[%d].Key = %q, want %q", i, entries[i].Key, k)
		}
	}
	if _, ok := n.Properties()["a"]; ok {
		t.Errorf("Properties()[a] still present after RemoveProperty")
	}
	if !n.entriesConsistent() {
		t.Errorf("entries inconsistent after RemoveProperty")
	}
}

func TestDuplicateProps_Clone(t *testing.T) {
	doc := parseDoc(t, "node a=1 a=2\n")
	n := doc.Nodes[0]
	c := n.Clone()
	if got := len(c.PropertyEntries()); got != 2 {
		t.Errorf("clone PropertyEntries len = %d, want 2", got)
	}
	// mutate clone, original untouched
	c.RemoveProperty("a")
	if got := len(n.PropertyEntries()); got != 2 {
		t.Errorf("original mutated: PropertyEntries len = %d, want 2", got)
	}
}

func TestDuplicateProps_SortPreservesDupOrder(t *testing.T) {
	src := "node b=1 a=2 a=3 b=4\n"
	doc := parseDoc(t, src)
	got := mustFormat(t, doc, WithFormatSortProperties(true))
	// Stable sort: among duplicates, original source order kept.
	want := "node a=2 a=3 b=1 b=4\n"
	if got != want {
		t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDuplicateProps_SetUpdatesLastInPlace(t *testing.T) {
	doc := parseDoc(t, "node a=1 a=2\n")
	n := doc.Nodes[0]
	Set(n, "a", NewInt(99))
	entries := n.PropertyEntries()
	if len(entries) != 2 {
		t.Fatalf("PropertyEntries len = %d, want 2", len(entries))
	}
	if v := entries[0].Value.Int(); v != 1 {
		t.Errorf("entries[0] = %v, want 1", entries[0].Value)
	}
	if v := entries[1].Value.Int(); v != 99 {
		t.Errorf("entries[1] = %v, want 99", entries[1].Value)
	}
	if v := n.Properties()["a"].Int(); v != 99 {
		t.Errorf("Properties()[a] = %v, want 99", n.Properties()["a"])
	}
}
