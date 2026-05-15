package kdl

import (
	"testing"
)

// kinds extracts the TokenKind sequence, optionally dropping whitespace and
// newlines so structural assertions stay readable.
func kinds(toks []Token, keepTrivia bool) []TokenKind {
	out := make([]TokenKind, 0, len(toks))
	for _, t := range toks {
		if !keepTrivia && (t.Kind == TokenWhitespace || t.Kind == TokenNewline) {
			continue
		}
		out = append(out, t.Kind)
	}
	return out
}

func eqKinds(t *testing.T, got []TokenKind, want ...TokenKind) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("kind count = %d %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("kind[%d] = %v, want %v (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestTokenize_Basic(t *testing.T) {
	toks := Tokenize([]byte("node arg key=1\n"))
	if len(toks) == 0 || toks[len(toks)-1].Kind != TokenEOF {
		t.Fatalf("stream must end in EOF, got %v", kinds(toks, true))
	}
	// node | arg | key | = | 1
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenIdent, TokenIdent,
		TokenEqual, TokenNumber, TokenEOF)
}

func TestTokenize_FullFidelityCoversInput(t *testing.T) {
	src := "node  arg \\\n  more\n"
	toks := Tokenize([]byte(src))
	// Every byte of src must be covered exactly once by consecutive tokens
	// (EOF excluded).
	off := 0
	for _, tk := range toks {
		if tk.Kind == TokenEOF {
			break
		}
		if int(tk.Start.Offset) != off {
			t.Fatalf("gap/overlap: token %v starts at %d, expected %d",
				tk.Kind, tk.Start.Offset, off)
		}
		off = int(tk.End.Offset)
	}
	if off != len(src) {
		t.Fatalf("coverage ended at %d, want %d", off, len(src))
	}
}

func TestTokenize_Escline(t *testing.T) {
	toks := Tokenize([]byte("node \\\n  arg\n"))
	var sawCont, sawNL bool
	for _, tk := range toks {
		if tk.Kind == TokenLineContinuation {
			sawCont = true
		}
		if sawCont && tk.Kind == TokenNewline {
			sawNL = true
		}
	}
	if !sawCont {
		t.Fatal("expected a TokenLineContinuation for the escline backslash")
	}
	if !sawNL {
		t.Fatal("expected the escline newline still emitted after the backslash")
	}
	// LineContinuation is significant (not trivia), so it survives the
	// whitespace/newline filter and sits between the two idents.
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenLineContinuation, TokenIdent, TokenEOF)
}

func TestTokenize_MultiLineString(t *testing.T) {
	src := "node \"\"\"\nhello\nworld\n\"\"\"\n"
	toks := Tokenize([]byte(src))
	var str *Token
	for i := range toks {
		if toks[i].Kind == TokenString {
			str = &toks[i]
			break
		}
	}
	if str == nil {
		t.Fatal("expected a TokenString for the multi-line string")
	}
	if str.Start.Line != 1 {
		t.Fatalf("string Start.Line = %d, want 1", str.Start.Line)
	}
	if str.End.Line <= str.Start.Line {
		t.Fatalf("multi-line string should span lines: start %d end %d",
			str.Start.Line, str.End.Line)
	}
}

func TestTokenize_NestedChildren(t *testing.T) {
	toks := Tokenize([]byte("a {\n  b c\n}\n"))
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenLBrace,
		TokenIdent, TokenIdent,
		TokenRBrace, TokenEOF)
}

func TestTokenize_Slashdash(t *testing.T) {
	toks := Tokenize([]byte("/- node arg\n"))
	if k := kinds(toks, false); k[0] != TokenSlashdash {
		t.Fatalf("first kind = %v, want Slashdash (full %v)", k[0], k)
	}
}

func TestTokenize_Comments(t *testing.T) {
	single := Tokenize([]byte("node // hi\n"))
	var sawComment bool
	for _, tk := range single {
		if tk.Kind == TokenComment {
			sawComment = true
		}
	}
	if !sawComment {
		t.Fatal("expected a TokenComment for single-line comment")
	}

	multi := Tokenize([]byte("node /* a\n b */ arg\n"))
	sawComment = false
	for _, tk := range multi {
		if tk.Kind == TokenComment {
			sawComment = true
		}
	}
	if !sawComment {
		t.Fatal("expected TokenComment spans for multi-line comment")
	}
}

func TestTokenize_KeywordsAndNumbers(t *testing.T) {
	toks := Tokenize([]byte("node #true #null 0xff 1.5\n"))
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenKeyword, TokenKeyword,
		TokenNumber, TokenNumber, TokenEOF)
}

func TestTokenize_TypeAnnotation(t *testing.T) {
	toks := Tokenize([]byte("node (u8)1\n"))
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenLParen, TokenIdent, TokenRParen,
		TokenNumber, TokenEOF)
}

func TestTokenize_Empty(t *testing.T) {
	toks := Tokenize([]byte(""))
	if len(toks) != 1 || toks[0].Kind != TokenEOF {
		t.Fatalf("empty input must yield exactly [EOF], got %v", kinds(toks, true))
	}
}

func TestTokenize_Locations(t *testing.T) {
	// Offsets/lines/cols are 1-based line, 1-based byte column, 0-based offset.
	toks := Tokenize([]byte("ab cd\n"))
	first := toks[0]
	if first.Kind != TokenIdent || first.Text != "ab" {
		t.Fatalf("first token = %+v", first)
	}
	if first.Start.Offset != 0 || first.Start.Line != 1 || first.Start.Column != 1 {
		t.Fatalf("first Start = %+v", first.Start)
	}
	if first.End.Offset != 2 {
		t.Fatalf("first End.Offset = %d, want 2", first.End.Offset)
	}
	// Find "cd".
	var cd *Token
	for i := range toks {
		if toks[i].Text == "cd" {
			cd = &toks[i]
		}
	}
	if cd == nil || cd.Start.Offset != 3 || cd.Start.Column != 4 {
		t.Fatalf("cd token = %+v", cd)
	}
}

func TestTokenize_Illegal(t *testing.T) {
	// An unterminated string is malformed; Tokenize must still terminate and
	// produce a complete stream ending in EOF (no panic / no hang).
	toks := Tokenize([]byte("node \"unterminated\n"))
	if toks[len(toks)-1].Kind != TokenEOF {
		t.Fatalf("stream must end in EOF even for malformed input: %v",
			kinds(toks, true))
	}
}

func TestTokenize_VersionOption(t *testing.T) {
	// Test that the Version option is accepted and doesn't break basic
	// tokenization.
	toks := Tokenize([]byte("node true\n"), WithVersion(Version1))
	eqKinds(t, kinds(toks, false),
		TokenIdent, TokenKeyword, TokenEOF)
}
