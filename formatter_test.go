package kdl

import (
	"math"
	"math/big"
	"strings"
	"testing"
)

// parseDoc is a test helper that parses src and fatals on any error diagnostic.
func parseDoc(t *testing.T, src string) *Document {
	t.Helper()
	result, err := ParseNamedWithDiagnostics("<test>", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == SeverityError {
			t.Fatalf("parse diagnostic: %s", d.Message)
		}
	}
	return result.Document
}

func mustFormat(t *testing.T, doc *Document, opts ...FormatOption) string {
	t.Helper()
	out, err := FormatToString(doc, opts...)
	if err != nil {
		t.Fatalf("FormatToString error: %v", err)
	}
	return out
}

func checkFormat(t *testing.T, src, want string, opts ...FormatOption) {
	t.Helper()
	doc := parseDoc(t, src)
	got := mustFormat(t, doc, opts...)
	if got != want {
		t.Errorf("\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// basic node structure

func TestFormatEmpty(t *testing.T) {
	checkFormat(t, "", "")
}

func TestFormatSimpleNode(t *testing.T) {
	checkFormat(t, "node", "node\n")
}

func TestFormatNodeWithArgs(t *testing.T) {
	checkFormat(t, `node 1 2 3`, "node 1 2 3\n")
}

func TestFormatNodeWithProperties(t *testing.T) {
	checkFormat(t, `node a=1 b=2`, "node a=1 b=2\n")
}

func TestFormatNoSpaceAroundEquals(t *testing.T) {
	// no spaces around = regardless of source
	checkFormat(t, `node   key  =  "val"`, "node key=val\n")
}

func TestFormatArgsAndProps(t *testing.T) {
	checkFormat(t, `node "hello" 42 key=true`, "node hello 42 key=#true\n")
}

func TestFormatMultipleNodes(t *testing.T) {
	checkFormat(t, "a\nb\nc\n", "a\nb\nc\n")
}

// string quoting

func TestFormatBareString(t *testing.T) {
	checkFormat(t, `node "simple"`, "node simple\n")
}

func TestFormatQuotedStringWithSpace(t *testing.T) {
	checkFormat(t, `node "hello world"`, "node \"hello world\"\n")
}

func TestFormatEmptyString(t *testing.T) {
	checkFormat(t, `node ""`, "node \"\"\n")
}

func TestFormatStringWithEscapes(t *testing.T) {
	checkFormat(t, `node "line1\nline2"`, "node \"line1\\nline2\"\n")
}

func TestFormatStringWithInternalQuote(t *testing.T) {
	checkFormat(t, `node "say \"hi\""`, "node \"say \\\"hi\\\"\"\n")
}

func TestFormatBareStringStartsWithSign(t *testing.T) {
	checkFormat(t, `node "+foo"`, "node +foo\n")
	checkFormat(t, `node "-bar"`, "node -bar\n")
}

func TestFormatReservedNodeNameQuoted(t *testing.T) {
	checkFormat(t, `"true"`, "\"true\"\n")
	checkFormat(t, `"false"`, "\"false\"\n")
	checkFormat(t, `"null"`, "\"null\"\n")
	checkFormat(t, `"true"`, "\"true\"\n", WithVersion(Version1))
	checkFormat(t, `"false"`, "\"false\"\n", WithVersion(Version1))
	checkFormat(t, `"null"`, "\"null\"\n", WithVersion(Version1))
}

func TestFormatStringStartingWithDigitQuoted(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewString("123abc")),
	}}
	got := mustFormat(t, doc)
	if got != "node \"123abc\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatPropertyKeyQuoted(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddProperty("key with space", NewInt(1)),
	}}
	got := mustFormat(t, doc)
	if got != "node \"key with space\"=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatEmptyPropertyKey(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddProperty("", NewInt(1)),
	}}
	got := mustFormat(t, doc)
	if got != "node \"\"=1\n" {
		t.Errorf("got %q", got)
	}
}

// value types

func TestFormatIntegerValues(t *testing.T) {
	checkFormat(t, "node 0 42 -7", "node 0 42 -7\n")
}

func TestFormatFloatAlwaysDecimal(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(1.0)),
	}}
	got := mustFormat(t, doc)
	if got != "node 1.0\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatFloatWithDecimal(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(1.5)),
	}}
	got := mustFormat(t, doc)
	if got != "node 1.5\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatFloatNegative(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(-3.14)),
	}}
	got := mustFormat(t, doc)
	if got != "node -3.14\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatFloatZero(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(0.0)),
	}}
	got := mustFormat(t, doc)
	if got != "node 0.0\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatNaNv2(t *testing.T) {
	checkFormat(t, "node #nan", "node #nan\n")
}

func TestFormatInfV2(t *testing.T) {
	checkFormat(t, "node #inf", "node #inf\n")
}

func TestFormatNegInfV2(t *testing.T) {
	checkFormat(t, "node #-inf", "node #-inf\n")
}

func TestFormatNaNv1(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(math.NaN())),
	}}
	got := mustFormat(t, doc, WithVersion(Version1))
	if got != "node \"nan\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatInfV1(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(math.Inf(1))),
	}}
	got := mustFormat(t, doc, WithVersion(Version1))
	if got != "node \"inf\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatNegInfV1(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(math.Inf(-1))),
	}}
	got := mustFormat(t, doc, WithVersion(Version1))
	if got != "node \"-inf\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatBoolV2(t *testing.T) {
	checkFormat(t, "node #true #false", "node #true #false\n")
}

func TestFormatNullV2(t *testing.T) {
	checkFormat(t, "node #null", "node #null\n")
}

func TestFormatBoolV1(t *testing.T) {
	checkFormat(t, "node true false", "node true false\n", WithVersion(Version1))
}

func TestFormatNullV1(t *testing.T) {
	checkFormat(t, "node null", "node null\n", WithVersion(Version1))
}

func TestFormatBigInt(t *testing.T) {
	var n big.Int
	n.SetString("99999999999999999999999999999999", 10)
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewBigInt(&n)),
	}}
	got := mustFormat(t, doc)
	if got != "node 99999999999999999999999999999999\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatBigFloat(t *testing.T) {
	f, _, _ := big.ParseFloat("1.2345678901234567890123456789", 10, 128, big.ToNearestEven)
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewBigFloat(f)),
	}}
	got := mustFormat(t, doc)
	if got != "node 1.2345678901234567890123456789\n" {
		t.Errorf("got %q", got)
	}
}

// numeric literal preservation

func TestFormatHexLiteralPreserved(t *testing.T) {
	checkFormat(t, "node 0xFF", "node 0xFF\n")
}

func TestFormatHexLiteralUpperCase(t *testing.T) {
	checkFormat(t, "node 0xDEADBEEF", "node 0xDEADBEEF\n")
}

func TestFormatHexLiteralWithUnderscores(t *testing.T) {
	checkFormat(t, "node 0xDEAD_BEEF", "node 0xDEAD_BEEF\n")
}

func TestFormatOctalLiteralPreserved(t *testing.T) {
	checkFormat(t, "node 0o17", "node 0o17\n")
}

func TestFormatOctalLiteralWithUnderscores(t *testing.T) {
	checkFormat(t, "node 0o7_7_7", "node 0o7_7_7\n")
}

func TestFormatBinaryLiteralPreserved(t *testing.T) {
	checkFormat(t, "node 0b1010", "node 0b1010\n")
}

func TestFormatBinaryLiteralWithUnderscores(t *testing.T) {
	checkFormat(t, "node 0b1111_0000", "node 0b1111_0000\n")
}

func TestFormatDecimalWithUnderscores(t *testing.T) {
	checkFormat(t, "node 1_000_000", "node 1_000_000\n")
}

func TestFormatFloatScientificNotationPreserved(t *testing.T) {
	// must not be converted to fixed-point form
	checkFormat(t, "node 1.5e2", "node 1.5e2\n")
}

func TestFormatFloatScientificUpperE(t *testing.T) {
	checkFormat(t, "node 1.23E-10", "node 1.23E-10\n")
}

func TestFormatFloatNegativeExponent(t *testing.T) {
	checkFormat(t, "node 6.674e-11", "node 6.674e-11\n")
}

func TestFormatFloatLeadingZero(t *testing.T) {
	checkFormat(t, "node 0.001", "node 0.001\n")
}

func TestFormatHighPrecisionFloatPreserved(t *testing.T) {
	// must not be rounded or converted to scientific notation
	src := "node 3.14159265358979323846264338327950288"
	checkFormat(t, src, "node 3.14159265358979323846264338327950288\n")
}

func TestFormatLargeIntPreserved(t *testing.T) {
	// integer too big for int64
	src := "node 99999999999999999999999999999999"
	checkFormat(t, src, "node 99999999999999999999999999999999\n")
}

func TestFormatNumericLiteralWithTypeAnnotation(t *testing.T) {
	// type annotation must not affect literal preservation
	checkFormat(t, "node (u32)0xFF", "node (u32)0xFF\n")
}

func TestFormatNumericLiteralInProperty(t *testing.T) {
	checkFormat(t, "node key=0xFF", "node key=0xFF\n")
}

func TestFormatNumericLiteralInlineChildren(t *testing.T) {
	checkFormat(t, "parent { child 0xFF }", "parent { child 0xFF }\n")
}

// programmatically created values have no literal — fallback to computed form

func TestFormatProgrammaticIntNoLiteral(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewInt(255)),
	}}
	if got := mustFormat(t, doc); got != "node 255\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatProgrammaticFloatNoLiteral(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewFloat(1.5)),
	}}
	if got := mustFormat(t, doc); got != "node 1.5\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatWithNumericLiteralManual(t *testing.T) {
	// WithNumericLiteral lets callers opt into exact representation for
	// programmatic values too
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewInt(255).WithNumericLiteral("0xFF")),
	}}
	if got := mustFormat(t, doc); got != "node 0xFF\n" {
		t.Errorf("got %q", got)
	}
}

// type annotations

func TestFormatTypeAnnotationOnNode(t *testing.T) {
	checkFormat(t, `(u8)node 255`, "(u8)node 255\n")
}

func TestFormatTypeAnnotationOnArg(t *testing.T) {
	checkFormat(t, `node (u8)255`, "node (u8)255\n")
}

func TestFormatTypeAnnotationOnProperty(t *testing.T) {
	checkFormat(t, `node key=(u8)255`, "node key=(u8)255\n")
}

func TestFormatTypeAnnotationNeedsQuoting(t *testing.T) {
	doc := &Document{Nodes: []*Node{func() *Node {
		n := NewNode("node")
		v := NewInt(1).WithTypeAnnotation("my type", true)
		n.AddArgument(v)
		return n
	}()}}
	got := mustFormat(t, doc)
	if got != "node (\"my type\")1\n" {
		t.Errorf("got %q", got)
	}
}

// property order

func TestFormatPropertyOrderPreserved(t *testing.T) {
	checkFormat(t, "node z=1 a=2 m=3", "node z=1 a=2 m=3\n")
}

func TestFormatPropertyOrderSorted(t *testing.T) {
	checkFormat(t, "node z=1 a=2 m=3", "node a=2 m=3 z=1\n", WithFormatSortProperties(true))
}

func TestFormatPropertyOrderSortedInline(t *testing.T) {
	// sorted order should also apply to inline children
	src := "parent { child z=3 a=1 m=2 }"
	checkFormat(t, src, "parent { child a=1 m=2 z=3 }\n", WithFormatSortProperties(true))
}

// line wrapping

func TestFormatNoWrapAtLimit(t *testing.T) {
	// "node arg1234567890" = 18 chars — no wrap
	checkFormat(t, "node arg1234567890", "node arg1234567890\n", WithFormatMaxLineLen(18))
}

func TestFormatWrapOneOverLimit(t *testing.T) {
	// "node arg1234567890X" = 19 chars. With limit 18 the last arg wraps.
	// "node " = 5, "arg123456789" = 12, total so far = 17 ≤ 18, ok
	// " X" = 2, 17+2 = 19 > 18 → wrap
	checkFormat(t,
		"node arg123456789 X",
		"node arg123456789 \\\n\tX\n",
		WithFormatMaxLineLen(18),
	)
}

func TestFormatWrapArgs(t *testing.T) {
	src := "node aaaa bbbb cccc dddd"
	// "node aaaa bbbb cccc" = 19 ≤ 20, " dddd" = 5, 19+5=24 > 20 → wrap dddd
	want := "node aaaa bbbb cccc \\\n\tdddd\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(20))
}

func TestFormatWrapProps(t *testing.T) {
	src := "node a=1 b=2 c=3 d=4"
	// "node" = 4, " a=1" = 4 → 8, " b=2" = 4 → 12, " c=3" = 4 → 16 > 15 → wrap c
	// after wrap at indent 0+1=1 tab: lineLen=1, "c=3" = 3 → 4, " d=4" = 4 → 8 ≤ 15
	want := "node a=1 b=2 \\\n\tc=3 d=4\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(15))
}

func TestFormatWrapContinuationAlsoWraps(t *testing.T) {
	// continuation line itself can also exceed limit and trigger another wrap
	// limit=10, node + 4 args each 5 chars wide
	// "node" = 4, " aaaaa" = 6 → 10 ≤ 10, " bbbbb" = 6 → 16 > 10 → wrap bbbbb
	// cont: "\t" = 1, "bbbbb" = 5 → 6, " ccccc" = 6 → 12 > 10 → wrap ccccc
	// cont: "\t" = 1, "ccccc" = 5 → 6, " ddddd" = 6 → 12 > 10 → wrap ddddd
	// cont: "\t" = 1, "ddddd" = 5 → 6
	want := "node aaaaa \\\n\tbbbbb \\\n\tccccc \\\n\tddddd\n"
	checkFormat(t, "node aaaaa bbbbb ccccc ddddd", want, WithFormatMaxLineLen(10))
}

func TestFormatWrapCustomIndent(t *testing.T) {
	// with 2-space indent, continuation uses 2 spaces
	src := "node aaaa bbbb cccc dddd"
	want := "node aaaa bbbb cccc \\\n  dddd\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(20), WithFormatIndentStr("  "))
}

func TestFormatWrapNestedIndent(t *testing.T) {
	// continuation at nested level uses indentLevel+1 tabs
	// limit 15, child node inside parent:
	// "\t" + "child aaaa" = 11, " bbbb" = 5 → 16 > 15 → wrap
	// continuation: "\t\t" = 2
	src := "parent {\nchild aaaa bbbb\n}"
	want := "parent {\n\tchild aaaa \\\n\t\tbbbb\n}\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(15))
}

func TestFormatSingleValueExceedsLimit(t *testing.T) {
	// arg is bare (all 'x') but "node " + 120 x's = 125 > 100, so it wraps.
	// after writeContinuation the value still exceeds the limit — that's
	// acceptable: we wrapped as far as we could (soft limit).
	long := strings.Repeat("x", 120)
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewString(long)),
	}}
	got := mustFormat(t, doc, WithFormatMaxLineLen(100))
	expected := "node \\\n\t" + long + "\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

// children

func TestFormatInlineSingleChild(t *testing.T) {
	checkFormat(t, "parent { child }", "parent { child }\n")
}

func TestFormatInlineMultipleChildren(t *testing.T) {
	checkFormat(t, "parent { child1; child2; child3 }", "parent { child1; child2; child3 }\n")
}

func TestFormatInlineChildrenWithArgs(t *testing.T) {
	checkFormat(t, "parent { child1 1 2; child2 foo=bar }", "parent { child1 1 2; child2 foo=bar }\n")
}

func TestFormatInlineChildrenWithTypeAnnotation(t *testing.T) {
	checkFormat(t, "parent { (t)child 1 }", "parent { (t)child 1 }\n")
}

func TestFormatMultilineChildrenSourcePreserved(t *testing.T) {
	// multiline source stays multiline even if it would fit on one line
	checkFormat(t, "parent {\nchild\n}", "parent {\n\tchild\n}\n")
}

func TestFormatInlineChildrenDoNotFit(t *testing.T) {
	// inline form too long → multiline
	src := "parent {\nchild1 aaaa\nchild2 bbbb\n}"
	// " { child1 aaaa; child2 bbbb }" = 30 chars, "parent" = 6, total 36
	// with limit 20: 6 + 30 = 36 > 20 → multiline
	want := "parent {\n\tchild1 aaaa\n\tchild2 bbbb\n}\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(20))
}

func TestFormatChildrenWithGrandchildrenAlwaysMultiline(t *testing.T) {
	// parent goes multiline because child has children; child { grandchild }
	// is inline since it was inline in source
	src := "parent {\nchild { grandchild }\n}"
	want := "parent {\n\tchild { grandchild }\n}\n"
	checkFormat(t, src, want)
}

func TestFormatDeepNesting(t *testing.T) {
	// c { d } is inline (inline in source); b and a go multiline (multiline in source)
	src := "a {\nb {\nc { d }\n}\n}"
	want := "a {\n\tb {\n\t\tc { d }\n\t}\n}\n"
	checkFormat(t, src, want)
}

func TestFormatChildrenAfterWrappedArgs(t *testing.T) {
	// args wrap, then children follow
	src := "node aaaa bbbb {\nchild\n}"
	want := "node aaaa \\\n\tbbbb {\n\tchild\n}\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(12))
}

func TestFormatInlineChildrenSemicolonSeparator(t *testing.T) {
	// verify "; " (not ";" or " ; ") is the separator
	src := "parent { a; b }"
	got := mustFormat(t, parseDoc(t, src))
	if !strings.Contains(got, "a; b") {
		t.Errorf("expected 'a; b' in %q", got)
	}
}

// empty children block preservation

func TestFormatEmptyChildrenInlinePreserved(t *testing.T) {
	checkFormat(t, "node {}", "node {}\n")
}

func TestFormatEmptyChildrenMultilineNormalized(t *testing.T) {
	// multiline empty block normalizes to {}
	checkFormat(t, "node {\n}", "node {}\n")
}

func TestFormatEmptyChildrenWithArgsPreserved(t *testing.T) {
	checkFormat(t, "node 1 2 {}", "node 1 2 {}\n")
}

func TestFormatEmptyChildrenProgrammaticNoBlock(t *testing.T) {
	// programmatic node has no source block → no {} emitted
	doc := &Document{Nodes: []*Node{NewNode("node")}}
	if got := mustFormat(t, doc); got != "node\n" {
		t.Errorf("got %q", got)
	}
}

// blank line preservation

func TestFormatBlankLinePreservedSingle(t *testing.T) {
	src := "a\n\nb\n"
	checkFormat(t, src, "a\n\nb\n")
}

func TestFormatBlankLineDoubleCollapsesToOne(t *testing.T) {
	src := "a\n\n\nb\n"
	checkFormat(t, src, "a\n\nb\n")
}

func TestFormatBlankLineTripleCollapsesToOne(t *testing.T) {
	src := "a\n\n\n\nb\n"
	checkFormat(t, src, "a\n\nb\n")
}

func TestFormatNoBlankLineNotAdded(t *testing.T) {
	src := "a\nb\n"
	checkFormat(t, src, "a\nb\n")
}

func TestFormatBlankLinePreservedInChildren(t *testing.T) {
	src := "parent {\na\n\nb\n}"
	want := "parent {\n\ta\n\n\tb\n}\n"
	checkFormat(t, src, want)
}

func TestFormatPreserveBlankLinesFalse(t *testing.T) {
	src := "a\n\nb\n\nc\n"
	checkFormat(t, src, "a\nb\nc\n", WithFormatPreserveBlankLines(false))
}

func TestFormatFirstNodeNoBlankLine(t *testing.T) {
	// blank line before first node is not tracked (only between nodes)
	src := "\n\na\nb\n"
	checkFormat(t, src, "a\nb\n")
}

func TestFormatBlankLinesBetweenSingleLineComments(t *testing.T) {
	src := "// a\n\n// b\nnode\n"
	checkFormat(t, src, "// a\n\n// b\nnode\n")
}

func TestFormatBlankLinesBetweenMultilineComments(t *testing.T) {
	src := "/* a */\n\n/* b */\nnode\n"
	checkFormat(t, src, "/* a */\n\n/* b */\nnode\n")
}

func TestFormatBlankLinesBeforeSlashdash(t *testing.T) {
	src := "node1\n\n/- node2\nnode3\n"
	checkFormat(t, src, "node1\n\n/- node2\nnode3\n")
}

func TestFormatBlankLinesBetweenSlashdashAndComment(t *testing.T) {
	src := "/- node1\n\n// comment\nnode2\n"
	checkFormat(t, src, "/- node1\n\n// comment\nnode2\n")
}

// version handling

func TestFormatV1Keywords(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").
			AddArgument(NewBool(true)).
			AddArgument(NewBool(false)).
			AddArgument(NewNull()),
	}}
	got := mustFormat(t, doc, WithVersion(Version1))
	if got != "node true false null\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatV1IdentifierQuoting(t *testing.T) {
	doc := &Document{Nodes: []*Node{
		NewNode("node").AddArgument(NewString("123")),
	}}
	v1 := mustFormat(t, doc, WithVersion(Version1))
	v2 := mustFormat(t, doc)
	if v1 != "node \"123\"\n" {
		t.Errorf("v1 got %q", v1)
	}
	if v2 != "node \"123\"\n" {
		t.Errorf("v2 got %q", v2)
	}
}

// options

func TestFormatCustomMaxLineLen(t *testing.T) {
	src := "node a b c"
	want := "node a \\\n\tb c\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(6))
}

func TestFormatCustomIndentStr(t *testing.T) {
	src := "parent {\nchild aaaa bbbb cccc\n}"
	want := "parent {\n  child aaaa bbbb \\\n    cccc\n}\n"
	checkFormat(t, src, want, WithFormatMaxLineLen(20), WithFormatIndentStr("  "))
}

func TestFormatCustomIndentStrChildren(t *testing.T) {
	// inline source: stays inline regardless of indent string
	src := "parent { child1; child2 }"
	got := mustFormat(t, parseDoc(t, src), WithFormatIndentStr("    "))
	if got != "parent { child1; child2 }\n" {
		t.Errorf("got %q", got)
	}
}

// idempotency

func TestFormatIdempotent(t *testing.T) {
	cases := []string{
		"node\n",
		"node 1 2 3\n",
		"node a=1 b=2\n",
		"parent { child1; child2 }\n",
		"parent {\n\tchild1\n\tchild2 { grandchild }\n}\n",
		"a\n\nb\n",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			doc1 := parseDoc(t, c)
			out1 := mustFormat(t, doc1)
			doc2 := parseDoc(t, out1)
			out2 := mustFormat(t, doc2)
			if out1 != out2 {
				t.Errorf("not idempotent:\nfirst:  %q\nsecond: %q", out1, out2)
			}
			// first pass should already equal input for these canonical cases
			if out1 != c {
				t.Errorf("first pass changed canonical input:\ngot:  %q\nwant: %q", out1, c)
			}
		})
	}
}

// misc edge cases

func TestFormatNodeNameNeedsQuoting(t *testing.T) {
	doc := &Document{Nodes: []*Node{NewNode("hello world")}}
	got := mustFormat(t, doc)
	if got != "\"hello world\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatNodeNameEmpty(t *testing.T) {
	doc := &Document{Nodes: []*Node{NewNode("")}}
	got := mustFormat(t, doc)
	if got != "\"\"\n" {
		t.Errorf("got %q", got)
	}
}

func TestFormatMixedBlankLinesAndNested(t *testing.T) {
	src := "a\n\nb { c }\n\nd\n"
	want := "a\n\nb { c }\n\nd\n"
	checkFormat(t, src, want)
}

func TestFormatInlineChildrenEmptyArgs(t *testing.T) {
	src := "parent { leaf }"
	checkFormat(t, src, "parent { leaf }\n")
}

func TestFormatSingleNodeManyPropsPreservesOrder(t *testing.T) {
	doc := &Document{Nodes: []*Node{func() *Node {
		n := NewNode("node")
		for _, k := range []string{"z", "y", "x", "w", "v"} {
			n.AddProperty(k, NewInt(1))
		}
		return n
	}()}}
	got := mustFormat(t, doc)
	if got != "node z=1 y=1 x=1 w=1 v=1\n" {
		t.Errorf("got %q", got)
	}
}

// comment preservation

func TestFormatSingleLineCommentPreserved(t *testing.T) {
	src := "// hello\nnode\n"
	checkFormat(t, src, "// hello\nnode\n")
}

func TestFormatSingleLineCommentTrailing(t *testing.T) {
	src := "node // trailing comment\n"
	checkFormat(t, src, "node // trailing comment\n")
}

func TestFormatSingleLineCommentBetweenNodes(t *testing.T) {
	src := "a\n// comment\nb\n"
	checkFormat(t, src, "a\n// comment\nb\n")
}

func TestFormatMultipleSingleLineComments(t *testing.T) {
	src := "// first\n// second\nnode\n"
	checkFormat(t, src, "// first\n// second\nnode\n")
}

func TestFormatMultilineCommentPreserved(t *testing.T) {
	src := "/* block comment */\nnode\n"
	checkFormat(t, src, "/* block comment */\nnode\n")
}

func TestFormatMultilineCommentMultipleLines(t *testing.T) {
	src := "/*\n * multi\n * line\n */\nnode\n"
	checkFormat(t, src, "/*\n * multi\n * line\n */\nnode\n")
}

func TestFormatMultilineCommentBetweenNodes(t *testing.T) {
	src := "a\n/* mid */\nb\n"
	checkFormat(t, src, "a\n/* mid */\nb\n")
}

func TestFormatMultilineCommentNestedPreserved(t *testing.T) {
	src := "/* outer /* inner */ outer */\nnode\n"
	checkFormat(t, src, "/* outer /* inner */ outer */\nnode\n")
}

func TestFormatSlashdashNodePreserved(t *testing.T) {
	src := "/-node\nother\n"
	checkFormat(t, src, "/- node\nother\n")
}

func TestFormatSlashdashNodeWithArgs(t *testing.T) {
	src := "/-node 1 2\nother\n"
	checkFormat(t, src, "/- node 1 2\nother\n")
}

func TestFormatSlashdashNodeWithProps(t *testing.T) {
	src := "/-node key=1\nother\n"
	checkFormat(t, src, "/- node key=1\nother\n")
}

func TestFormatSlashdashNodeWithInlineChildren(t *testing.T) {
	src := "/-parent { child }\nother\n"
	checkFormat(t, src, "/- parent { child }\nother\n")
}

func TestFormatSlashdashNodeWithMultilineChildren(t *testing.T) {
	src := "/-parent {\nchild\n}\nother\n"
	checkFormat(t, src, "/- parent {\n\tchild\n}\nother\n")
}

func TestFormatSlashdashBeforeFirstNode(t *testing.T) {
	src := "/-disabled\nnode\n"
	checkFormat(t, src, "/- disabled\nnode\n")
}

func TestFormatMultipleSlashdashNodes(t *testing.T) {
	src := "/-first\n/-second\nnode\n"
	checkFormat(t, src, "/- first\n/- second\nnode\n")
}

func TestFormatCommentAndSlashdashMixed(t *testing.T) {
	src := "// reason disabled\n/-node\nother\n"
	checkFormat(t, src, "// reason disabled\n/- node\nother\n")
}

func TestFormatSlashdashNodeSpaceOption(t *testing.T) {
	src := "/- node 1 2\nother\n"
	checkFormat(t, src, "/-node 1 2\nother\n", WithFormatSlashdashNodeSpace(false))
}

func TestFormatSlashdashNodeSpaceOptionMultilineChildren(t *testing.T) {
	src := "/- parent {\nchild\n}\nother\n"
	checkFormat(t, src, "/-parent {\n\tchild\n}\nother\n", WithFormatSlashdashNodeSpace(false))
}

func TestFormatSlashdashTrailingComment(t *testing.T) {
	src := "/-node // reason\nother\n"
	checkFormat(t, src, "/- node // reason\nother\n")
}

// inline slashdash

func TestFormatInlineSlashdashArg(t *testing.T) {
	src := "node /-1 2\n"
	checkFormat(t, src, "node /-1 2\n")
}

func TestFormatInlineSlashdashArgWithSpace(t *testing.T) {
	src := "node /-1 2\n"
	checkFormat(t, src, "node /- 1 2\n", WithFormatSlashdashArgSpace(true))
}

func TestFormatInlineSlashdashArgBeforeFirst(t *testing.T) {
	// slashed arg before real arg
	src := "node /- 1 realarg\n"
	checkFormat(t, src, "node /-1 realarg\n")
}

func TestFormatInlineSlashdashArgAfterLast(t *testing.T) {
	// slashed arg after all real args
	src := "node realarg /- 2\n"
	checkFormat(t, src, "node realarg /-2\n")
}

func TestFormatInlineSlashdashProp(t *testing.T) {
	src := "node /-key=val\n"
	checkFormat(t, src, "node /-key=val\n")
}

func TestFormatInlineSlashdashPropWithSpace(t *testing.T) {
	src := "node /-key=val\n"
	checkFormat(t, src, "node /- key=val\n", WithFormatSlashdashArgSpace(true))
}

func TestFormatInlineSlashdashPropInterleaved(t *testing.T) {
	// slashed prop between real props
	src := "node a=1 /- b=2 c=3\n"
	checkFormat(t, src, "node a=1 /-b=2 c=3\n")
}

func TestFormatInlineSlashdashChildren(t *testing.T) {
	src := "node /- { child }\n"
	checkFormat(t, src, "node /-{ child }\n")
}

func TestFormatInlineSlashdashChildrenWithSpace(t *testing.T) {
	src := "node /- { child }\n"
	checkFormat(t, src, "node /- { child }\n", WithFormatSlashdashArgSpace(true))
}

func TestFormatInlineSlashdashChildrenMultiline(t *testing.T) {
	src := "node /- {\nchild\n}\n"
	checkFormat(t, src, "node /-{\n\tchild\n}\n")
}

func TestFormatInlineSlashdashAndRealChildren(t *testing.T) {
	// slashed children block followed by real children block
	src := "node /- { old } {\nnew\n}\n"
	checkFormat(t, src, "node /-{ old } {\n\tnew\n}\n")
}

func TestFormatInlineSlashdashArgForcesMultilineInlineChild(t *testing.T) {
	// child with slashed-children cannot be inlined in parent
	src := "parent {\nchild /- { old }\n}\n"
	checkFormat(t, src, "parent {\n\tchild /-{ old }\n}\n")
}

func TestFormatInlineSlashdashIdempotent(t *testing.T) {
	cases := []string{
		"node /-1 2\n",
		"node /-key=val\n",
		"node /-{ child }\n",
		"/- node\nother\n",
		"/- node 1 2\n",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			doc1 := parseDoc(t, c)
			out1 := mustFormat(t, doc1)
			doc2 := parseDoc(t, out1)
			out2 := mustFormat(t, doc2)
			if out1 != out2 {
				t.Errorf("not idempotent:\nfirst:  %q\nsecond: %q", out1, out2)
			}
			if out1 != c {
				t.Errorf("first pass changed canonical input:\ngot:  %q\nwant: %q", out1, c)
			}
		})
	}
}

func TestFormatBlankLineThenComment(t *testing.T) {
	src := "a\n\n// comment\nb\n"
	checkFormat(t, src, "a\n\n// comment\nb\n")
}

func TestFormatTrailingCommentWithBlankLine(t *testing.T) {
	src := "a // trailing\n\nb\n"
	checkFormat(t, src, "a // trailing\n\nb\n")
}

func TestFormatCommentInChildrenBlock(t *testing.T) {
	src := "parent {\n// child comment\nchild\n}\n"
	checkFormat(t, src, "parent {\n\t// child comment\n\tchild\n}\n")
}

func TestFormatTrailingCommentInChildrenBlock(t *testing.T) {
	src := "parent {\nchild // trailing\n}\n"
	checkFormat(t, src, "parent {\n\tchild // trailing\n}\n")
}

func TestFormatSlashdashInChildrenBlock(t *testing.T) {
	src := "parent {\n/-disabled\nchild\n}\n"
	checkFormat(t, src, "parent {\n\t/- disabled\n\tchild\n}\n")
}

func TestFormatTrailingCommentOnlyDoc(t *testing.T) {
	// document with only a trailing comment (no nodes)
	src := "// lonely comment\n"
	checkFormat(t, src, "// lonely comment\n")
}

func TestFormatCommentAfterLastNode(t *testing.T) {
	src := "node\n// trailing doc comment\n"
	checkFormat(t, src, "node\n// trailing doc comment\n")
}

func TestFormatMultilineCommentAfterLastNode(t *testing.T) {
	src := "node\n/* end of file */\n"
	checkFormat(t, src, "node\n/* end of file */\n")
}

func TestFormatCommentsForceMultilineChildren(t *testing.T) {
	// children with leading comments cannot be inlined
	src := "parent { // comment\nchild\n}\n"
	checkFormat(t, src, "parent {\n\t// comment\n\tchild\n}\n")
}

func TestFormatSingleLineCommentExactPreservation(t *testing.T) {
	// whitespace inside the comment must be preserved exactly
	src := "//   spaced   comment  \nnode\n"
	checkFormat(t, src, "//   spaced   comment  \nnode\n")
}

func TestFormatMultilineCommentExactPreservation(t *testing.T) {
	src := "  /*  exact   \n  spacing  */\nnode\n"
	checkFormat(t, src, "/*  exact   \n  spacing  */\nnode\n")
}

func TestFormatCommentIdempotent(t *testing.T) {
	cases := []string{
		"// comment\nnode\n",
		"node // trailing\n",
		"a\n// between\nb\n",
		"/* block */\nnode\n",
		"node\n// eof comment\n",
		"/- commented\nnode\n",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			doc1 := parseDoc(t, c)
			out1 := mustFormat(t, doc1)
			doc2 := parseDoc(t, out1)
			out2 := mustFormat(t, doc2)
			if out1 != out2 {
				t.Errorf("not idempotent:\nfirst:  %q\nsecond: %q", out1, out2)
			}
			if out1 != c {
				t.Errorf("first pass changed canonical input:\ngot:  %q\nwant: %q", out1, c)
			}
		})
	}
}

// arg/prop interleaving

func TestFormatArgPropOrderPreserveDefault(t *testing.T) {
	// preserve interleaving of args and props as in source by default
	checkFormat(t, `node a=1 "x" b=2 "y"`, "node a=1 x b=2 y\n")
}

func TestFormatArgPropOrderPreserveExplicit(t *testing.T) {
	checkFormat(t, `node "x" a=1 "y" b=2`, "node x a=1 y b=2\n",
		WithFormatArgPropOrder(ArgPropOrderPreserve))
}

func TestFormatArgPropOrderArgsFirst(t *testing.T) {
	checkFormat(t, `node a=1 "x" b=2 "y"`, "node x y a=1 b=2\n",
		WithFormatArgPropOrder(ArgPropOrderArgsFirst))
}

func TestFormatArgPropOrderPropsFirst(t *testing.T) {
	checkFormat(t, `node "x" a=1 "y" b=2`, "node a=1 b=2 x y\n",
		WithFormatArgPropOrder(ArgPropOrderPropsFirst))
}

func TestFormatArgPropOrderPreserveAllArgs(t *testing.T) {
	checkFormat(t, `node 1 2 3`, "node 1 2 3\n",
		WithFormatArgPropOrder(ArgPropOrderPreserve))
}

func TestFormatArgPropOrderPreserveAllProps(t *testing.T) {
	checkFormat(t, `node a=1 b=2`, "node a=1 b=2\n",
		WithFormatArgPropOrder(ArgPropOrderPreserve))
}

func TestFormatArgPropOrderPropsFirstAllArgs(t *testing.T) {
	checkFormat(t, `node 1 2 3`, "node 1 2 3\n",
		WithFormatArgPropOrder(ArgPropOrderPropsFirst))
}

func TestFormatArgPropOrderArgsFirstAllProps(t *testing.T) {
	checkFormat(t, `node a=1 b=2`, "node a=1 b=2\n",
		WithFormatArgPropOrder(ArgPropOrderArgsFirst))
}

func TestFormatArgPropOrderPreserveChildren(t *testing.T) {
	src := `parent { child a=1 "x" b=2 "y" }`
	checkFormat(t, src, "parent { child a=1 x b=2 y }\n")
}

func TestFormatArgPropOrderArgsFirstChildren(t *testing.T) {
	src := `parent { child a=1 "x" b=2 "y" }`
	checkFormat(t, src, "parent { child x y a=1 b=2 }\n",
		WithFormatArgPropOrder(ArgPropOrderArgsFirst))
}

func TestFormatArgPropOrderPropsFirstChildren(t *testing.T) {
	src := `parent { child "x" a=1 "y" b=2 }`
	checkFormat(t, src, "parent { child a=1 b=2 x y }\n",
		WithFormatArgPropOrder(ArgPropOrderPropsFirst))
}

func TestFormatArgPropOrderWithSortProps(t *testing.T) {
	// sort props but leave interleaving of args and props as in source
	checkFormat(t, `node z=1 "x" a=2 "y" m=3`, "node a=2 x m=3 y z=1\n",
		WithFormatArgPropOrder(ArgPropOrderPreserve),
		WithFormatSortProperties(true))
}

func TestFormatArgPropOrderProgrammaticPreserveInsertionOrder(t *testing.T) {
	// preserve insertion order for programmatically created nodes too
	n := NewNode("node")
	n.AddProperty("a", NewInt(1))
	n.AddArgument(NewString("x"))
	n.AddProperty("b", NewInt(2))
	n.AddArgument(NewString("y"))
	doc := NewDocument()
	doc.AddNode(n)
	got := mustFormat(t, doc)
	want := "node a=1 x b=2 y\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFormatArgPropOrderProgrammaticArgsFirst(t *testing.T) {
	n := NewNode("node")
	n.AddProperty("a", NewInt(1))
	n.AddArgument(NewString("x"))
	n.AddProperty("b", NewInt(2))
	n.AddArgument(NewString("y"))
	doc := NewDocument()
	doc.AddNode(n)
	got := mustFormat(t, doc, WithFormatArgPropOrder(ArgPropOrderArgsFirst))
	want := "node x y a=1 b=2\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFormatArgPropOrderRemoveKeepsInterleave(t *testing.T) {
	doc := parseDoc(t, `node a=1 "x" b=2 "y"`)
	doc.Nodes[0].RemoveArgument(0) // remove "x"
	got := mustFormat(t, doc)
	want := "node a=1 b=2 y\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFormatArgPropOrderRemovePropKeepsInterleave(t *testing.T) {
	doc := parseDoc(t, `node a=1 "x" b=2 "y"`)
	doc.Nodes[0].RemoveProperty("a")
	got := mustFormat(t, doc)
	want := "node x b=2 y\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestFormatWriteToWriter(t *testing.T) {
	doc := parseDoc(t, "node 1 2")
	var sb strings.Builder
	err := Format(doc, &sb)
	if err != nil {
		t.Fatal(err)
	}
	if sb.String() != "node 1 2\n" {
		t.Errorf("got %q", sb.String())
	}
}
