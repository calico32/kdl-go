package kdl

import "io"

// ======================== interfaces ========================

// A DecodeOption is an option for the [Decode] family of functions. As decoding
// is the composition of parsing and unmarshaling, DecodeOptions are the union
// of ParseOptions and UnmarshalOptions.
type DecodeOption interface{ decodeOption() }

// An UnmarshalOption is an option for the [Unmarshal] family of functions. It
// can also be passed to a [Decode] function.
type UnmarshalOption interface {
	DecodeOption
	applyUnmarshaler(*decoder)
}

type unmarshalOptionFunc func(*decoder)

func (f unmarshalOptionFunc) applyUnmarshaler(d *decoder) { f(d) }
func (f unmarshalOptionFunc) decodeOption()               {}

// A ParseOption is an option for the [Parse] family of functions. It can also
// be passed to a [Decode] function.
type ParseOption interface {
	DecodeOption
	applyParser(*parser)
}

type parseOptionFunc func(*parser)

func (f parseOptionFunc) applyParser(p *parser) { f(p) }
func (f parseOptionFunc) decodeOption()         {}

type EncodeOption interface{ encodeOption() }

type MarshalOption interface {
	EncodeOption
	applyMarshaler(*encoder)
}

type marshalOptionFunc func(*encoder)

func (f marshalOptionFunc) applyMarshaler(e *encoder) { f(e) }
func (f marshalOptionFunc) encodeOption()             {}

type EmitOption interface {
	EncodeOption
	applyEmitter(*emitter)
}

type emitterOptionFunc func(*emitter)

func (f emitterOptionFunc) applyEmitter(e *emitter) { f(e) }
func (f emitterOptionFunc) encodeOption()           {}

/// ======================== common options ========================

type versionOption Version

// WithVersion sets the KDL version to parse/emit.
func WithVersion(v Version) versionOption       { return versionOption(v) }
func (v versionOption) applyParser(p *parser)   { p.version = Version(v) }
func (v versionOption) applyEmitter(e *emitter) { e.version = Version(v) }
func (v versionOption) decodeOption()           {}
func (v versionOption) encodeOption()           {}

type traceOption struct{ io.Writer }

// WithTrace enables tracing of the parsing or marshaling process to the provided writer.
func WithTrace(w io.Writer) traceOption         { return traceOption{w} }
func (o traceOption) applyParser(p *parser)     { p.trace = o.Writer }
func (o traceOption) applyMarshaler(e *encoder) { e.traceWriter = o.Writer }
func (o traceOption) decodeOption()             {}
func (o traceOption) encodeOption()             {}

// ======================== parse options ========================

type sourceNameOption string

// WithSourceName specifies the name of the input source (e.g. filename) for use in
// error messages and node locations. The default is "<input>".
func WithSourceName(name string) sourceNameOption { return sourceNameOption(name) }
func (o sourceNameOption) applyParser(p *parser)  { p.lexer.File().name = string(o) }
func (o sourceNameOption) decodeOption()          {}

// WithParseTrace is a ParseOption that enables tracing of the parsing process
// to the provided writer.
//
// Deprecated: Use [WithTrace] instead, which can be passed to both parsing and
// marshaling functions.
func WithParseTrace(w io.Writer) ParseOption { return WithTrace(w) }

// WithLocations is a ParseOption that controls whether the parser populates
// location information on nodes and in diagnostics. Default: true (locations
// enabled).
func WithLocations(v bool) ParseOption {
	return parseOptionFunc(func(p *parser) { p.withLocations = v })
}

// DupMode controls how the parser reacts to repeated property keys on a node.
type DupMode uint8

const (
	// DupAllow silently accepts duplicate properties, applying the KDL spec's
	// last-wins semantics. Earlier occurrences remain accessible via
	// [Node.PropertyEntries]. This is the default.
	DupAllow DupMode = iota
	// DupWarn emits a Warning diagnostic for each repeated occurrence after
	// the first. Parsing still succeeds.
	DupWarn
	// DupError emits an Error diagnostic for each repeated occurrence after
	// the first. The duplicate is still recorded on the node so the AST is
	// complete.
	DupError
)

// WithDuplicateProperties controls how the parser reacts when a node has the
// same property key more than once. The KDL spec says the rightmost value
// wins; this option lets callers surface duplicates as warnings or errors.
func WithDuplicateProperties(mode DupMode) ParseOption {
	return parseOptionFunc(func(p *parser) {
		p.duplicateProps = mode
	})
}

// ======================== unmarshal options ========================

// WithStrict specifies whether strict mode should be enabled globally for the
// decode operation. In strict mode, the decoder returns an error if any KDL
// nodes, properties, or arguments cannot be mapped to the target value. It also
// disables any canonical conversions when unmarshaling values.
func WithStrict(strict bool) UnmarshalOption {
	return unmarshalOptionFunc(func(d *decoder) { d.strict = strict })
}

// ======================== marshal options ========================

// (none yet)

// ======================== emit options ========================

// WithIndent sets the indent string for the emitter.
func WithIndent(s string) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.indent = s })
}

// WithStringAlwaysQuote sets whether to always quote strings.
func WithStringAlwaysQuote(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.stringAlwaysQuote = v })
}

// WithFloatCapitalExponent sets whether to use capital 'E' for exponents.
func WithFloatCapitalExponent(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatCapitalExponent = v })
}

// WithFloatMinExponent sets the minimum exponent for using scientific notation.
func WithFloatMinExponent(v int) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatMinExponent = v })
}

// WithFloatPlus sets whether to include '+' for positive floats.
func WithFloatPlus(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatPlus = v })
}

// WithFloatDecimalPoint sets whether to always include a decimal point in floats.
func WithFloatDecimalPoint(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatDecimalPoint = v })
}

// WithFloatExponentPlus sets whether to include '+' for positive exponents.
func WithFloatExponentPlus(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatExponentPlus = v })
}

// WithFloatDecimalOrExponent sets whether to always include either a decimal
// point or exponent part in floats.
func WithFloatDecimalOrExponent(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.floatDecimalOrExponent = v })
}

// WithTestSuiteFloatOptions applies float emission options that match the
// upstream KDL 2.0.0 test suite expectations. Specifically:
//   - Capital 'E' for exponents
//   - Minimum exponent of 2 (i.e., use scientific notation for exponents >= 2)
//   - Always include a decimal point
//   - Always include a '+' for positive exponents
func WithTestSuiteFloatOptions() EmitOption {
	return emitterOptionFunc(func(e *emitter) {
		e.floatCapitalExponent = true
		e.floatMinExponent = 2
		e.floatDecimalPoint = true
		e.floatExponentPlus = true
	})
}

// WithIntegerFormat sets the format to use for integers.
func WithIntegerFormat(f IntegerFormat) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.integerFormat = f })
}

// WithEmitEmptyChildren sets whether to emit an empty children block when
// a node has no children.
func WithEmitEmptyChildren(v bool) EmitOption {
	return emitterOptionFunc(func(e *emitter) { e.emitEmptyChildren = v })
}

// ======================== utils ========================

func splitDecodeOptions(opts []DecodeOption) ([]ParseOption, []UnmarshalOption) {
	parseOpts := []ParseOption{}
	for _, opt := range opts {
		if p, ok := opt.(ParseOption); ok {
			parseOpts = append(parseOpts, p)
		}
	}
	unmarshalOpts := []UnmarshalOption{}
	for _, opt := range opts {
		if u, ok := opt.(UnmarshalOption); ok {
			unmarshalOpts = append(unmarshalOpts, u)
		}
	}
	return parseOpts, unmarshalOpts
}

func splitEncodeOptions(opts []EncodeOption) ([]MarshalOption, []EmitOption) {
	marshalOpts := []MarshalOption{}
	for _, opt := range opts {
		if m, ok := opt.(MarshalOption); ok {
			marshalOpts = append(marshalOpts, m)
		}
	}
	emitOpts := []EmitOption{}
	for _, opt := range opts {
		if e, ok := opt.(EmitOption); ok {
			emitOpts = append(emitOpts, e)
		}
	}
	return marshalOpts, emitOpts
}
