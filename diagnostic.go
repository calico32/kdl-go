package kdl

import "fmt"

// A DiagnosticSeverity is the severity level of a Diagnostic.
type DiagnosticSeverity int

const (
	SeverityError DiagnosticSeverity = iota
	SeverityWarning
	SeverityInfo
	SeverityHint
)

// A Diagnostic represents a parse/validation problem with a source range and
// message.
type Diagnostic struct {
	Start    Location
	End      Location
	Severity DiagnosticSeverity
	Message  string
	Code     string // optional machine-readable code
	// Related points at the schema rule(s) that produced this diagnostic.
	// Locations in Related refer to the schema source file, not the source
	// file the diagnostic was raised in.
	Related []DiagnosticRelated
}

// DiagnosticRelated describes a related source location (typically the schema
// rule that produced a validation diagnostic).
type DiagnosticRelated struct {
	Start   Location
	End     Location
	Message string
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s: %s", d.Start, d.Message)
}

// ParseResult holds a (possibly partial) document and the diagnostics
// produced while parsing it. Document is non-nil even when errors are present,
// containing whatever nodes were successfully parsed.
type ParseResult struct {
	Document    *Document
	Diagnostics []Diagnostic
	// Version is the KDL spec version the parser settled on. Set even when
	// the input was originally parsed under VersionAuto.
	Version Version
}

// HasErrors returns true if any diagnostic has SeverityError.
func (r *ParseResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
