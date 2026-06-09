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
