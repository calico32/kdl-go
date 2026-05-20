package kdl

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Format writes a formatted KDL representation of d to w. Default style:
//   - indentation with tabs
//   - soft 100-char lines, escline wrapping
//   - inline children when they fit
//   - preserved blank lines (≤1)
//   - no space around = in properties
//   - property order preserved from source
//   - space after /- for node comments, no space after /- for inline arg/prop/children comments
//   - arguments and properties interleaved in source order
//
// These defaults can be overridden with [FormatOption]s, such as
// [WithFormatMaxLineLen], [WithFormatIndentStr],
// [WithFormatPreserveBlankLines], [WithFormatSortProperties],
// [WithFormatSlashdashNodeSpace], [WithFormatSlashdashArgSpace], and
// [WithFormatArgPropOrder].
func Format(d *Document, w io.Writer, opts ...FormatOption) error {
	f := newFormatter(opts)
	f.formatDocument(d)
	_, err := io.WriteString(w, f.b.String())
	return err
}

// FormatToString is like [Format] but returns the result as a string.
func FormatToString(d *Document, opts ...FormatOption) (string, error) {
	f := newFormatter(opts)
	f.formatDocument(d)
	return f.b.String(), nil
}

func newFormatter(opts []FormatOption) *formatter {
	f := &formatter{
		version:            Version2,
		maxLineLen:         100,
		indentStr:          "\t",
		preserveBlankLines: true,
		sortProperties:     false,
		slashdashNodeSpace: true,
		slashdashArgSpace:  false,
		argPropOrder:       ArgPropOrderPreserve,
	}
	for _, o := range opts {
		o.applyFormatter(f)
	}
	return f
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

// A FormatOption configures the behavior of [Format] and [FormatToString]. See
// the various WithFormat* functions for specific options.
type FormatOption interface {
	applyFormatter(*formatter)
}

type formatOptionFunc func(*formatter)

func (fn formatOptionFunc) applyFormatter(f *formatter) { fn(f) }

func (v versionOption) applyFormatter(f *formatter) { f.version = v.v }

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

// WithFormatArgPropOrder controls how a node's arguments and properties are
// interleaved in formatted output. Default: [ArgPropOrderPreserve].
func WithFormatArgPropOrder(v ArgPropOrder) FormatOption {
	return formatOptionFunc(func(f *formatter) { f.argPropOrder = v })
}

type formatter struct {
	b strings.Builder

	version            Version
	maxLineLen         int
	indentStr          string
	preserveBlankLines bool
	sortProperties     bool
	slashdashNodeSpace bool // emit "/- " before node name (default: true, "/- ")
	slashdashArgSpace  bool // emit "/- " before inline arg/prop/children (default: "/-", no space)
	argPropOrder       ArgPropOrder

	indentLevel int
	lineLen     int // bytes written since last \n
}

// bodyPlan returns the sequence of nodeEntryKind slots describing how to
// interleave n's arguments and properties for the active argPropOrder. The
// returned slice has len == len(n.args) + len(n.propEntries) — one prop slot
// per occurrence, including duplicates.
func (f *formatter) bodyPlan(n *Node) []nodeEntryKind {
	nArgs := len(n.Arguments())
	nProps := len(n.propEntries)
	switch f.argPropOrder {
	case ArgPropOrderPreserve:
		if n.entriesConsistent() {
			return n.entries
		}
		return argPropPlan(nArgs, nProps, true)
	case ArgPropOrderPropsFirst:
		return argPropPlan(nArgs, nProps, false)
	case ArgPropOrderArgsFirst:
		return argPropPlan(nArgs, nProps, true)
	default:
		panic(fmt.Sprintf("invalid ArgPropOrder: %v", f.argPropOrder))
	}
}

// argPropPlan returns a plan that emits all args before all props (argsFirst=true)
// or all props before all args (argsFirst=false).
func argPropPlan(nArgs, nProps int, argsFirst bool) []nodeEntryKind {
	plan := make([]nodeEntryKind, 0, nArgs+nProps)
	if argsFirst {
		for range nArgs {
			plan = append(plan, nodeEntryArg)
		}
		for range nProps {
			plan = append(plan, nodeEntryProp)
		}
	} else {
		for range nProps {
			plan = append(plan, nodeEntryProp)
		}
		for range nArgs {
			plan = append(plan, nodeEntryArg)
		}
	}
	return plan
}

// slashdashNodePrefix returns the token string that precedes a slashdash node name.
func (f *formatter) slashdashNodePrefix() string {
	if f.slashdashNodeSpace {
		return "/- "
	}
	return "/-"
}

// slashdashArgPrefix returns the token string that precedes a slashdash inline arg/prop/children.
func (f *formatter) slashdashArgPrefix() string {
	if f.slashdashArgSpace {
		return "/- "
	}
	return "/-"
}

func (f *formatter) write(s string) {
	f.b.WriteString(s)
	if lastNewline := strings.LastIndexByte(s, '\n'); lastNewline >= 0 {
		f.lineLen = len(s) - lastNewline - 1
	} else {
		f.lineLen += len(s)
	}
}

func (f *formatter) writeIndent() {
	if f.indentLevel > 0 {
		f.write(strings.Repeat(f.indentStr, f.indentLevel))
	}
}

// writeContinuation emits an escline and indents one level deeper than current.
func (f *formatter) writeContinuation() {
	f.write(" \\\n")
	f.write(strings.Repeat(f.indentStr, f.indentLevel+1))
}

func (f *formatter) formatDocument(d *Document) {
	for i, n := range d.Nodes {
		for j, c := range n.leadingComments {
			if (i > 0 || j > 0) && f.preserveBlankLines && c.blankLineBefore {
				f.write("\n")
			}
			f.writeComment(c)
		}
		if (i > 0 || len(n.leadingComments) > 0) && f.preserveBlankLines && n.blankLineBefore {
			f.write("\n")
		}
		f.formatNode(n)
	}
	for j, c := range d.TrailingComments {
		if (len(d.Nodes) > 0 || j > 0) && f.preserveBlankLines && c.blankLineBefore {
			f.write("\n")
		}
		f.writeComment(c)
	}
}

func (f *formatter) formatNode(n *Node) {
	f.writeIndent()
	f.formatNodeBody(n, "")
}

func (f *formatter) formatNodeBody(n *Node, prefix string) {
	if prefix != "" {
		f.write(prefix)
	}

	if ty, ok := n.TypeAnnotation(); ok {
		f.write("(" + f.identToString(ty) + ")")
	}
	f.write(f.identToString(n.Name()))

	// lookup maps for inline slashdash args and props
	slashedArgAt := make(map[int][]Value)
	slashedPropAt := make(map[int][]KV)
	var slashedChildren []InlineSlashdash
	for _, sd := range n.inlineSlashdashes {
		switch sd.kind {
		case InlineSlashdashArg:
			slashedArgAt[sd.afterArgCount] = append(slashedArgAt[sd.afterArgCount], sd.argValue)
		case InlineSlashdashProp:
			slashedPropAt[sd.afterPropCount] = append(slashedPropAt[sd.afterPropCount], KV{Key: sd.propKey, Value: sd.propVal})
		case InlineSlashdashChildren:
			slashedChildren = append(slashedChildren, sd)
		}
	}

	argPrefix := f.slashdashArgPrefix()
	writeSlashedArgsAt := func(idx int) {
		for _, arg := range slashedArgAt[idx] {
			argStr := fmt.Sprintf(" %s%s", argPrefix, f.valueToString(arg))
			if f.lineLen+len(argStr) > f.maxLineLen {
				f.writeContinuation()
				argStr = argStr[1:]
			}
			f.write(argStr)
		}
	}

	entries := n.propEntries
	if f.sortProperties {
		entries = slices.Clone(entries)
		slices.SortStableFunc(entries, func(a, b propEntry) int {
			if a.key < b.key {
				return -1
			} else if a.key > b.key {
				return 1
			}
			return 0
		})
	}
	writeSlashedPropsAt := func(idx int) {
		for _, prop := range slashedPropAt[idx] {
			propStr := fmt.Sprintf(" %s%s=%s", argPrefix, f.identToString(prop.Key), f.valueToString(prop.Value))
			if f.lineLen+len(propStr) > f.maxLineLen {
				f.writeContinuation()
				propStr = propStr[1:]
			}
			f.write(propStr)
		}
	}

	plan := f.bodyPlan(n)
	var argIndex, propIndex int
	for _, kind := range plan {
		switch kind {
		case nodeEntryArg:
			writeSlashedArgsAt(argIndex)
			a := n.Arguments()[argIndex]
			argStr := " " + f.valueToString(a)
			if f.lineLen+len(argStr) > f.maxLineLen {
				f.writeContinuation()
				argStr = argStr[1:]
			}
			f.write(argStr)
			argIndex++
		case nodeEntryProp:
			writeSlashedPropsAt(propIndex)
			e := entries[propIndex]
			propStr := " " + f.identToString(e.key) + "=" + f.valueToString(e.value)
			if f.lineLen+len(propStr) > f.maxLineLen {
				f.writeContinuation()
				propStr = propStr[1:]
			}
			f.write(propStr)
			propIndex++
		}
	}
	writeSlashedArgsAt(len(n.Arguments()))
	writeSlashedPropsAt(len(entries))

	// emit slashed children blocks before the real children block
	for _, sd := range slashedChildren {
		f.writeSlashedChildrenBlock(sd)
	}

	children := n.Children().Nodes
	_, hasSourceChildren := n.ChildrenInline()
	if len(children) > 0 || hasSourceChildren {
		if len(children) == 0 {
			f.write(" {}")
		} else {
			sourceInline, hasSourceInline := n.ChildrenInline()
			forceMultiline := hasSourceInline && !sourceInline
			inline, canInline := f.inlineChildren(children)
			full := " { " + inline + " }"
			if !forceMultiline && canInline && f.lineLen+len(full) <= f.maxLineLen {
				f.write(full)
			} else {
				f.write(" {\n")
				f.indentLevel++
				f.formatDocument(n.Children())
				f.indentLevel--
				f.writeIndent()
				f.write("}")
			}
		}
	}

	if n.trailingComment != nil {
		// trailingComment.text already includes the trailing newline
		f.write(" ")
		f.write(n.trailingComment.text)
	} else {
		f.write("\n")
	}
}

// writeSlashedChildrenBlock emits a /- { ... } children block in-line at the
// current position (appended to the current node line, not on its own line).
func (f *formatter) writeSlashedChildrenBlock(sd InlineSlashdash) {
	argPrefix := f.slashdashArgPrefix()
	children := sd.children.Nodes
	sourceInline, hasSourceInline := sd.ChildrenInline()
	forceMultiline := hasSourceInline && !sourceInline

	if len(children) == 0 {
		f.write(" " + argPrefix + "{}")
		return
	}

	inlineStr, canInline := f.inlineChildren(children)
	full := " " + argPrefix + "{ " + inlineStr + " }"
	if !forceMultiline && canInline && f.lineLen+len(full) <= f.maxLineLen {
		f.write(full)
		return
	}

	// Multiline: open on the current node line, children indented.
	f.write(" " + argPrefix + "{\n")
	f.indentLevel++
	f.formatDocument(&sd.children)
	f.indentLevel--
	f.writeIndent()
	f.write("}")
}

// writeComment emits a single comment at the current indentation level.
// Single-line comment text already includes a trailing newline.
// Multi-line comment text does not; a newline is appended.
// Slashdash comments re-format the commented-out node as KDL.
func (f *formatter) writeComment(c Comment) {
	switch c.kind {
	case CommentSingleLine:
		f.writeIndent()
		f.write(c.text) // includes trailing \n
	case CommentMultiLine:
		f.writeIndent()
		f.write(c.text)
		f.write("\n")
	case CommentSlashdash:
		f.writeIndent()
		f.writeSlashdashComment(c.node)
	}
}

// writeSlashdashComment formats a slashdash-commented node as "/- <node>" at
// the current indentation level. The node content is re-formatted as KDL.
func (f *formatter) writeSlashdashComment(n *Node) {
	f.formatNodeBody(n, f.slashdashNodePrefix())
}

// inlineChildren returns the "; "-joined inline representation of nodes and
// whether inlining is permitted. Inlining is not permitted when any child has
// children of its own, has leading comments, has a trailing comment, or when
// a blank line separates siblings (and blank line preservation is on).
func (f *formatter) inlineChildren(nodes []*Node) (string, bool) {
	for i, c := range nodes {
		if len(c.Children().Nodes) > 0 {
			return "", false
		}
		if i > 0 && f.preserveBlankLines && c.blankLineBefore {
			return "", false
		}
		if len(c.leadingComments) > 0 || c.trailingComment != nil {
			return "", false
		}
		for _, sd := range c.inlineSlashdashes {
			if sd.kind == InlineSlashdashChildren {
				return "", false
			}
		}
	}
	parts := make([]string, len(nodes))
	for i, c := range nodes {
		parts[i] = f.inlineNode(c)
	}
	return strings.Join(parts, "; "), true
}

// inlineNode returns the inline (no indent, no newline) representation of n,
// including any slashed args/props interleaved in source order.
func (f *formatter) inlineNode(n *Node) string {
	var b strings.Builder
	if ty, ok := n.TypeAnnotation(); ok {
		b.WriteByte('(')
		b.WriteString(f.identToString(ty))
		b.WriteByte(')')
	}
	b.WriteString(f.identToString(n.Name()))

	// slashed-arg/prop lookup maps
	slashedArgAt := make(map[int][]Value)
	slashedPropAt := make(map[int][]KV)
	for _, sd := range n.inlineSlashdashes {
		switch sd.kind {
		case InlineSlashdashArg:
			slashedArgAt[sd.afterArgCount] = append(slashedArgAt[sd.afterArgCount], sd.argValue)
		case InlineSlashdashProp:
			slashedPropAt[sd.afterPropCount] = append(slashedPropAt[sd.afterPropCount], KV{Key: sd.propKey, Value: sd.propVal})
		}
	}

	argPrefix := f.slashdashArgPrefix()
	entries := n.propEntries
	if f.sortProperties {
		entries = slices.Clone(entries)
		slices.SortStableFunc(entries, func(a, b propEntry) int {
			if a.key < b.key {
				return -1
			} else if a.key > b.key {
				return 1
			}
			return 0
		})
	}

	plan := f.bodyPlan(n)
	var argIdx, propIdx int
	for _, kind := range plan {
		switch kind {
		case nodeEntryArg:
			for _, sv := range slashedArgAt[argIdx] {
				b.WriteString(" ")
				b.WriteString(argPrefix)
				b.WriteString(f.valueToString(sv))
			}
			b.WriteString(" ")
			b.WriteString(f.valueToString(n.Arguments()[argIdx]))
			argIdx++
		case nodeEntryProp:
			for _, sp := range slashedPropAt[propIdx] {
				b.WriteString(" ")
				b.WriteString(argPrefix)
				b.WriteString(f.identToString(sp.Key))
				b.WriteString("=")
				b.WriteString(f.valueToString(sp.Value))
			}
			e := entries[propIdx]
			b.WriteString(" ")
			b.WriteString(f.identToString(e.key))
			b.WriteString("=")
			b.WriteString(f.valueToString(e.value))
			propIdx++
		}
	}
	for _, sv := range slashedArgAt[len(n.Arguments())] {
		b.WriteString(" ")
		b.WriteString(argPrefix)
		b.WriteString(f.valueToString(sv))
	}
	for _, sp := range slashedPropAt[len(entries)] {
		b.WriteString(" ")
		b.WriteString(argPrefix)
		b.WriteString(f.identToString(sp.Key))
		b.WriteString("=")
		b.WriteString(f.valueToString(sp.Value))
	}

	return b.String()
}

// identToString returns the KDL identifier/string representation of s,
// quoting it when necessary for the active version.
func (f *formatter) identToString(s string) string {
	if !CanBeBareIdentifier(s, f.version) {
		return `"` + EscapeString(s, f.version) + `"`
	}
	return s
}

// stringToKDL returns the v2 KDL bare or quoted string representation of s.
func (f *formatter) stringToKDL(s string) string {
	if f.version == Version1 || !CanBeBareIdentifier(s, f.version) {
		return `"` + EscapeString(s, f.version) + `"`
	}
	return s
}

// valueToString returns the KDL representation of v.
func (f *formatter) valueToString(v Value) string {
	var b strings.Builder
	if ty, ok := v.TypeAnnotation(); ok {
		b.WriteByte('(')
		b.WriteString(f.stringToKDL(ty))
		b.WriteByte(')')
	}
	switch v.Kind() {
	case String:
		b.WriteString(f.stringToKDL(v.String()))
	case Int:
		if lit, ok := v.NumericLiteral(); ok {
			b.WriteString(lit)
		} else {
			b.WriteString(strconv.FormatInt(int64(v.Int()), 10))
		}
	case BigInt:
		if lit, ok := v.NumericLiteral(); ok {
			b.WriteString(lit)
		} else {
			b.WriteString(v.BigInt().String())
		}
	case Float:
		if lit, ok := v.NumericLiteral(); ok {
			b.WriteString(lit)
		} else if math.IsNaN(v.Float()) {
			if f.version == Version1 {
				b.WriteString(`"nan"`)
			} else {
				b.WriteString("#nan")
			}
		} else {
			b.WriteString(f.floatToString(new(big.Float).SetFloat64(v.Float())))
		}
	case BigFloat:
		if lit, ok := v.NumericLiteral(); ok {
			b.WriteString(lit)
		} else {
			b.WriteString(f.floatToString(v.BigFloat()))
		}
	case Bool:
		if f.version == Version1 {
			if v.Bool() {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
		} else {
			if v.Bool() {
				b.WriteString("#true")
			} else {
				b.WriteString("#false")
			}
		}
	case Null:
		if f.version == Version1 {
			b.WriteString("null")
		} else {
			b.WriteString("#null")
		}
	default:
		panic(errors.Errorf("formatter.valueToString: unknown value kind %v", v.Kind()))
	}
	return b.String()
}

// floatToString formats f using fixed notation, always including a decimal point.
func (f *formatter) floatToString(fl *big.Float) string {
	if fl.IsInf() {
		if f.version == Version1 {
			if fl.Sign() > 0 {
				return `"inf"`
			}
			return `"-inf"`
		}
		if fl.Sign() > 0 {
			return "#inf"
		}
		return "#-inf"
	}
	s := fl.Text('f', -1)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}
