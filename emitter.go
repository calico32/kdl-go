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

// Emit writes the KDL representation of the given Document to the provided
// writer. By default, the emitter uses an indent of four spaces and standard
// float formatting. Options can be provided to customize the output.
//   - [WithVersion] to set the KDL version to emit (default: [Version2]).
//   - [WithIndent] to set a custom indent string (default: four spaces).
//   - [WithStringAlwaysQuote] to always quote strings (default: false).
//   - [WithFloatCapitalExponent] to use capital 'E' for exponents (default: false).
//   - [WithFloatMinExponent] to set the minimum exponent for using scientific notation (default: 10).
//   - [WithFloatPlus] to include '+' for positive floats (default: false).
//   - [WithFloatDecimalPoint] to always include a decimal point in floats (default: false).
//   - [WithFloatExponentPlus] to include '+' for positive exponents (default: false).
//   - [WithFloatDecimalOrExponent] to always include either a decimal point or exponent part in floats (default: true).
//     Without this option, integer-like floating points may be reparsed as integers.
//   - [WithEmitEmptyChildren] to emit an empty children block when a node has no children (default: false). Also
//     configurable at the node level via [Node.Hints].
//   - [WithIntegerFormat] to set the format to use for integers (default: [Decimal]).
func Emit(d *Document, w io.Writer, opts ...EmitterOption) error {
	e := &emitter{
		w:      w,
		indent: "    ",

		stringAlwaysQuote:      false,
		floatCapitalExponent:   false,
		floatMinExponent:       10,
		floatPlus:              false,
		floatDecimalPoint:      false,
		floatExponentPlus:      false,
		floatDecimalOrExponent: true,
		version:                Version2,
		integerFormat:          Decimal,
		emitEmptyChildren:      false,
	}
	for _, opt := range opts {
		opt.applyEmitter(e)
	}
	return e.emitDocument(d)
}

// EmitToString is like [Emit] but returns the emitted KDL as a string; see
// [Emit] for details.
func EmitToString(d *Document, opts ...EmitterOption) (string, error) {
	var buf strings.Builder
	err := Emit(d, &buf, opts...)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// IntegerFormat specifies the format to use for emitting integers.
type IntegerFormat int

const (
	Decimal IntegerFormat = iota
	Hex
	Octal
	Binary
)

type emitter struct {
	w           io.Writer
	indent      string
	indentLevel int

	stringAlwaysQuote      bool
	floatCapitalExponent   bool
	floatMinExponent       int
	floatPlus              bool
	floatDecimalPoint      bool
	floatExponentPlus      bool
	floatDecimalOrExponent bool
	version                Version
	integerFormat          IntegerFormat
	emitEmptyChildren      bool
}

type emitterHints struct {
	// EmitEmptyChildren controls whether to emit an empty children block when
	// the node has no children.
	EmitEmptyChildren bool
}

type EmitterOption interface {
	applyEmitter(*emitter)
}

type emitterOptionFunc func(*emitter)

func (f emitterOptionFunc) applyEmitter(e *emitter) {
	f(e)
}

// WithIndent sets the indent string for the emitter.
func WithIndent(s string) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.indent = s })
}

// WithStringAlwaysQuote sets whether to always quote strings.
func WithStringAlwaysQuote(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.stringAlwaysQuote = v })
}

// WithFloatCapitalExponent sets whether to use capital 'E' for exponents.
func WithFloatCapitalExponent(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatCapitalExponent = v })
}

// WithFloatMinExponent sets the minimum exponent for using scientific notation.
func WithFloatMinExponent(v int) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatMinExponent = v })
}

// WithFloatPlus sets whether to include '+' for positive floats.
func WithFloatPlus(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatPlus = v })
}

// WithFloatDecimalPoint sets whether to always include a decimal point in floats.
func WithFloatDecimalPoint(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatDecimalPoint = v })
}

// WithFloatExponentPlus sets whether to include '+' for positive exponents.
func WithFloatExponentPlus(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatExponentPlus = v })
}

// WithFloatDecimalOrExponent sets whether to always include either a decimal
// point or exponent part in floats.
func WithFloatDecimalOrExponent(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.floatDecimalOrExponent = v })
}

// WithTestSuiteFloatOptions applies float emission options that match the
// upstream KDL 2.0.0 test suite expectations. Specifically:
//   - Capital 'E' for exponents
//   - Minimum exponent of 2 (i.e., use scientific notation for exponents >= 2)
//   - Always include a decimal point
//   - Always include a '+' for positive exponents
func WithTestSuiteFloatOptions() EmitterOption {
	return emitterOptionFunc(func(e *emitter) {
		e.floatCapitalExponent = true
		e.floatMinExponent = 2
		e.floatDecimalPoint = true
		e.floatExponentPlus = true
	})
}

// WithIntegerFormat sets the format to use for integers.
func WithIntegerFormat(f IntegerFormat) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.integerFormat = f })
}

// WithEmitEmptyChildren sets whether to emit an empty children block when
// a node has no children.
func WithEmitEmptyChildren(v bool) EmitterOption {
	return emitterOptionFunc(func(e *emitter) { e.emitEmptyChildren = v })
}

type versionOption struct {
	v Version
}

func (v versionOption) applyParser(p *parser)   { p.version = v.v }
func (v versionOption) applyEmitter(e *emitter) { e.version = v.v }

// WithVersion sets the KDL version to parse/emit.
func WithVersion(v Version) versionOption {
	return versionOption{v}
}

func (e *emitter) emit(s string) error {
	_, err := io.WriteString(e.w, s)
	return err
}

func (e *emitter) emitIndent() error {
	if e.indentLevel > 0 && e.indent != "" {
		return e.emit(strings.Repeat(e.indent, e.indentLevel))
	}
	return nil
}

func (e *emitter) emitDocument(d *Document) error {
	for _, n := range d.Nodes {
		if err := e.emitNode(n); err != nil {
			return err
		}
	}
	return nil
}

func (e *emitter) emitNode(n *Node) error {
	if err := e.emitIndent(); err != nil {
		return err
	}
	if ty, ok := n.TypeAnnotation(); ok {
		if err := e.emit("("); err != nil {
			return err
		}
		if err := e.emitIdentifier(ty); err != nil {
			return err
		}
		if err := e.emit(")"); err != nil {
			return err
		}
	}

	if err := e.emitIdentifier(n.name); err != nil {
		return err
	}

	for _, a := range n.args {
		if err := e.emit(" "); err != nil {
			return err
		}
		if err := e.emitValue(a); err != nil {
			return err
		}
	}

	props := slices.Clone(n.propOrder)
	slices.Sort(props)
	for _, p := range props {
		if err := e.emit(" "); err != nil {
			return err
		}
		if err := e.emitIdentifier(p); err != nil {
			return err
		}
		if err := e.emit("="); err != nil {
			return err
		}
		if err := e.emitValue(n.props[p]); err != nil {
			return err
		}
	}

	if len(n.children.Nodes) > 0 || n.hints.EmitEmptyChildren || e.emitEmptyChildren {
		if err := e.emit(" {\n"); err != nil {
			return err
		}
		e.indentLevel++
		if err := e.emitDocument(&n.children); err != nil {
			return err
		}
		e.indentLevel--
		if err := e.emitIndent(); err != nil {
			return err
		}
		if err := e.emit("}"); err != nil {
			return err
		}
	}

	if err := e.emit("\n"); err != nil {
		return err
	}

	return nil
}

func (e *emitter) emitIdentifier(s string) error {
	if e.version != Version1 {
		return e.emitString(s)
	}

	needsQuoting := s == "" || e.stringAlwaysQuote
	if !needsQuoting {
		runes := []rune(s)
		allowDash := len(runes) == 1 || !isDigit(runes[1])
		for i, r := range runes {
			if i == 0 {
				if !isV1IdentStartChar(r, allowDash) {
					needsQuoting = true
					break
				}
			} else if !isV1IdentChar(r) {
				needsQuoting = true
				break
			}
		}
	}

	if needsQuoting {
		return e.emitString(s)
	} else {
		return e.emit(s)
	}
}

func (e *emitter) emitString(s string) error {
	needsQuoting := s == "" || e.stringAlwaysQuote || e.version == Version1
	if !needsQuoting {
		for i, r := range s {
			if i == 0 {
				if !isIdentStartChar(r) && !isSign(r) && r != '.' {
					needsQuoting = true
					break
				}
			} else if !isIdentChar(r) {
				needsQuoting = true
				break
			}
		}
	}

	if needsQuoting {
		return e.emit(`"` + escapeString(e.version, s) + `"`)
	} else {
		return e.emit(s)
	}
}

func (e *emitter) emitValue(v Value) error {
	if ty, ok := v.TypeAnnotation(); ok {
		if err := e.emit("("); err != nil {
			return err
		}
		if err := e.emitString(ty); err != nil {
			return err
		}
		if err := e.emit(")"); err != nil {
			return err
		}
	}
	switch v.Kind() {
	case String:
		return e.emitString(v.String())
	case Int:
		switch e.integerFormat {
		case Decimal:
			return e.emit(strconv.FormatInt(int64(v.Int()), 10))
		case Hex:
			if v.Int() < 0 {
				return e.emit(fmt.Sprintf("-0x%x", -v.Int()))
			}
			return e.emit(fmt.Sprintf("0x%x", v.Int()))
		case Octal:
			if v.Int() < 0 {
				return e.emit(fmt.Sprintf("-0o%o", -v.Int()))
			}
			return e.emit(fmt.Sprintf("0o%o", v.Int()))
		case Binary:
			if v.Int() < 0 {
				return e.emit(fmt.Sprintf("-0b%b", -v.Int()))
			}
			return e.emit(fmt.Sprintf("0b%b", v.Int()))
		default:
			panic("kdl.Emit: invalid integer format")
		}
	case BigInt:
		if e.integerFormat == Decimal {
			return e.emit(v.BigInt().String())
		}
		var base int
		var prefix string
		switch e.integerFormat {
		case Hex:
			base = 16
			prefix = "0x"
		case Octal:
			base = 8
			prefix = "0o"
		case Binary:
			base = 2
			prefix = "0b"
		default:
			panic("kdl.Emit: invalid integer format")
		}
		s := prefix + v.BigInt().Text(base)
		if v.BigInt().Sign() < 0 {
			s = "-" + s
		}
		return e.emit(s)
	case Float:
		if math.IsNaN(v.Float()) {
			if e.version == Version1 {
				// v1 doesn't have NaN... guess we can emit a string
				return e.emit(`"nan"`)
			} else {
				return e.emit("#nan")
			}
		}
		return e.emitFloat(new(big.Float).SetFloat64(v.Float()))
	case BigFloat:
		return e.emitFloat(v.BigFloat())
	case Bool:
		if e.version == Version1 {
			if v.Bool() {
				return e.emit("true")
			} else {
				return e.emit("false")
			}
		} else {
			if v.Bool() {
				return e.emit("#true")
			} else {
				return e.emit("#false")
			}
		}
	case Null:
		if e.version == Version1 {
			return e.emit("null")
		} else {
			return e.emit("#null")
		}
	default:
		return errors.Errorf("unknown value type: %T", v)
	}
}

func (e *emitter) emitFloat(f *big.Float) error {
	if f.IsInf() {
		if e.version == Version1 {
			// v1 doesn't have Inf... guess we can emit a string
			if f.Sign() > 0 {
				return e.emit(`"inf"`)
			} else {
				return e.emit(`"-inf"`)
			}
		} else {
			if f.Sign() > 0 {
				return e.emit("#inf")
			} else {
				return e.emit("#-inf")
			}
		}
	}

	if e.floatPlus && f.Sign() > 0 {
		if err := e.emit("+"); err != nil {
			return err
		}
	}

	s := f.Text('e', -1)
	parts := strings.Split(s, "e")
	if len(parts) != 2 {
		return errors.Errorf("failed to format float: %s", s)
	}

	exponent, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}

	useScientific := true
	if exponent >= 0 {
		if exponent < e.floatMinExponent {
			useScientific = false
		}
	} else {
		if -exponent < e.floatMinExponent {
			useScientific = false
		}
	}

	if !useScientific {
		s = f.Text('f', -1)
	} else {
		formatChar := byte('e')
		if e.floatCapitalExponent {
			formatChar = 'E'
		}
		s = f.Text(formatChar, -1)

		if !e.floatExponentPlus {
			idx := strings.IndexByte(s, formatChar)
			if idx != -1 && idx+1 < len(s) && s[idx+1] == '+' {
				s = s[:idx+1] + s[idx+2:]
			}
		}
	}

	if e.floatDecimalPoint {
		if !strings.Contains(s, ".") {
			idx := strings.IndexAny(s, "eE")
			if idx != -1 {
				s = s[:idx] + ".0" + s[idx:]
			} else {
				s = s + ".0"
			}
		}
	} else if e.floatDecimalOrExponent {
		if !strings.ContainsAny(s, ".eE") {
			s = s + ".0"
		}
	}

	return e.emit(s)
}
