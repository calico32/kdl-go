package kdl

// A FormatOption configures the behavior of [Format] and [FormatToString]. See
// the various WithFormat* functions for specific options.
type FormatOption interface {
	applyFormatter(*formatter)
}

type formatOptionFunc func(*formatter)

func (fn formatOptionFunc) applyFormatter(f *formatter) { fn(f) }

func (v versionOption) applyFormatter(f *formatter) { f.version = Version(v) }

// WithFormatMaxLineLen sets the soft maximum line length before escline-wrapping.
// Default: 100.
func WithFormatMaxLineLen(n int) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.maxLineLen = n })
}

// WithFormatIndentStr sets the string used for each indentation level.
// Default: "\t".
func WithFormatIndentStr(s string) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.indentStr = s })
}

// WithFormatPreserveBlankLines controls whether blank lines from the parsed
// source are preserved in formatted output (capped at one consecutive blank
// line). Default: true.
func WithFormatPreserveBlankLines(v bool) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.preserveBlankLines = v })
}

// WithFormatSortProperties controls whether properties are emitted in
// alphabetical order. Default: false (source/insertion order preserved).
func WithFormatSortProperties(v bool) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.sortProperties = v })
}

// WithFormatSlashdashNodeSpace controls whether a space is emitted between /-
// and the node name in top-level slashdash comments. Default: true (spaced
// form "/- nodename"). Set false for "/-nodename".
func WithFormatSlashdashNodeSpace(v bool) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.slashdashNodeSpace = v })
}

// WithFormatSlashdashArgSpace controls whether a space is emitted between /-
// and the following arg, property, or children block inside a node. Default:
// false (compact form "/-value"). Set true for "/- value".
func WithFormatSlashdashArgSpace(v bool) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.slashdashArgSpace = v })
}

// ArgPropOrder controls how a node's arguments and properties are interleaved
// in formatted output.
type ArgPropOrder uint8

const (
	// ArgPropOrderPreserve emits arguments and properties in the order they
	// appeared in the source (or, for programmatically-built nodes, the order
	// in which they were added). This is the default.
	ArgPropOrderPreserve ArgPropOrder = iota
	// ArgPropOrderArgsFirst emits all arguments before all properties.
	ArgPropOrderArgsFirst
	// ArgPropOrderPropsFirst emits all properties before all arguments.
	ArgPropOrderPropsFirst
)

// WithFormatArgPropOrder controls how a node's arguments and properties are
// interleaved in formatted output. Default: [ArgPropOrderPreserve].
func WithFormatArgPropOrder(v ArgPropOrder) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.argPropOrder = v })
}
