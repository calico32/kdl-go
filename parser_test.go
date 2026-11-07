package kdl

import "testing"

func TestParseWithSourceName(t *testing.T) {
	result := ParseStringWithDiagnostics("node", WithSourceName("test.kdl"))
	if len(result.Diagnostics) != 0 {
		t.Errorf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.Document == nil {
		t.Errorf("expected document, got nil")
	} else if result.Document.Nodes[0].Location().Filename != "test.kdl" {
		t.Errorf("expected source name 'test.kdl', got '%s'", result.Document.Nodes[0].Location().Filename)
	}

	result = ParseStringWithDiagnostics("{", WithSourceName("test.kdl"))
	if len(result.Diagnostics) == 0 {
		t.Errorf("expected diagnostics, got none")
	} else if result.Diagnostics[0].Start.Filename != "test.kdl" {
		t.Errorf("expected diagnostic source name 'test.kdl', got '%s'", result.Diagnostics[0].Start.Filename)
	}

	result = ParseStringWithDiagnostics("/- kdl-version 1\nnode #true", WithSourceName("test.kdl"))
	if len(result.Diagnostics) == 0 {
		t.Errorf("expected diagnostics, got none")
	} else if result.Diagnostics[0].Start.Filename != "test.kdl" {
		t.Errorf("expected diagnostic source name 'test.kdl', got '%s'", result.Diagnostics[0].Start.Filename)
	}
}

func TestParseMultiLineStrings(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"LF", "note \"\"\"\n  a\n  b\n  \"\"\"\n", "a\nb"},
		{"CRLF", "note \"\"\"\r\n  a\r\n  b\r\n  \"\"\"\r\n", "a\nb"},
		{"raw CRLF", "note #\"\"\"\r\n  a\r\n  b\r\n  \"\"\"#\r\n", "a\nb"},
		{"CRLF blank line", "note \"\"\"\r\n  a\r\n\r\n  b\r\n  \"\"\"\r\n", "a\n\nb"},
		{"escaped newline", "note \"\"\"\n  \\u{41}\\n\n  \"\"\"\n", "A\n"},
		{"escaped newline mid-string", "note \"\"\"\n  a\\nb\n  c\n  \"\"\"\n", "a\nb\nc"},
		{"whitespace escape", "note \"\"\"\n  a \\\n     b\n  \"\"\"\n", "a b"},
		{"escaped backslash at end of line", "note \"\"\"\n  a\\\\\n  b\n  \"\"\"\n", "a\\\nb"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseStringWithDiagnostics(tc.src, WithVersion(Version2))
			if len(result.Diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
			}
			if got := result.Document.Nodes[0].Arg(0).String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseBareKeywords(t *testing.T) {
	for _, keyword := range []string{"true", "false", "null", "inf", "-inf", "nan"} {
		t.Run(keyword, func(t *testing.T) {
			result := ParseStringWithDiagnostics("note "+keyword+"\n", WithVersion(Version2))
			if len(result.Diagnostics) == 0 {
				t.Errorf("expected diagnostics for bare %s, got none", keyword)
			}
		})
	}
}
