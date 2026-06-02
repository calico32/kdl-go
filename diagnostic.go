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

// Diagnostic codes. These are stable, machine-readable identifiers attached to
// the [Diagnostic.Code] field. They follow a "kdl/<category>/<name>" shape for
// filtering/grouping purposes by editors, tests, and other tools.
//
// New codes may be added in minor releases; existing codes will not be renamed
// without a major version bump.
const (
	// lexer

	DiagLexError = "kdl/lex/error"

	// parser (syntax)

	DiagSyntaxUnexpectedToken             = "kdl/syntax/unexpected-token"
	DiagSyntaxExpectedString              = "kdl/syntax/expected-string"
	DiagSyntaxExpectedValue               = "kdl/syntax/expected-value"
	DiagSyntaxExpectedNumber              = "kdl/syntax/expected-number"
	DiagSyntaxExpectedNodeTerminator      = "kdl/syntax/expected-node-terminator"
	DiagSyntaxExpectedNodeSpace           = "kdl/syntax/expected-node-space"
	DiagSyntaxExpectedLineSpace           = "kdl/syntax/expected-line-space"
	DiagSyntaxExpectedEOLAfterEscline     = "kdl/syntax/expected-eol-after-escline"
	DiagSyntaxExpectedRBrace              = "kdl/syntax/expected-rbrace"
	DiagSyntaxExpectedValueAfterSlashdash = "kdl/syntax/expected-value-after-slashdash"
	DiagSyntaxUnterminatedComment         = "kdl/syntax/unterminated-comment"
	DiagSyntaxInvalidFloat                = "kdl/syntax/invalid-float"
	DiagSyntaxInvalidInteger              = "kdl/syntax/invalid-integer"
	DiagSyntaxV1UnquotedIdent             = "kdl/syntax/v1-unquoted-identifier"
	DiagSyntaxDuplicateProperty           = "kdl/syntax/duplicate-property"

	// parser (meta - diagnostics about the parse itself)

	DiagParseVersionAutoFallback   = "kdl/parse/version-auto-fallback"
	DiagParseVersionMarkerInvalid  = "kdl/parse/version-marker-invalid"
	DiagParseVersionMarkerMismatch = "kdl/parse/version-marker-mismatch"

	// schema validation

	DiagSchemaUnexpectedNode          = "kdl/schema/unexpected-node"
	DiagSchemaUnexpectedProperty      = "kdl/schema/unexpected-property"
	DiagSchemaMissingRequiredProperty = "kdl/schema/missing-required-property"
	DiagSchemaNodeCountMin            = "kdl/schema/node-count-min"
	DiagSchemaNodeCountMax            = "kdl/schema/node-count-max"
	DiagSchemaValueCountMin           = "kdl/schema/value-count-min"
	DiagSchemaValueCountMax           = "kdl/schema/value-count-max"
	DiagSchemaTypeMismatch            = "kdl/schema/type-mismatch"
	DiagSchemaEnumMismatch            = "kdl/schema/enum-mismatch"
	DiagSchemaPatternMismatch         = "kdl/schema/pattern-mismatch"
	DiagSchemaLengthMin               = "kdl/schema/length-min"
	DiagSchemaLengthMax               = "kdl/schema/length-max"
	DiagSchemaBoundGt                 = "kdl/schema/bound-gt"
	DiagSchemaBoundGte                = "kdl/schema/bound-gte"
	DiagSchemaBoundLt                 = "kdl/schema/bound-lt"
	DiagSchemaBoundLte                = "kdl/schema/bound-lte"
	DiagSchemaModulo                  = "kdl/schema/modulo"
)
