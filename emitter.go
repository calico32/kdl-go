package kdl

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

// Emit writes the KDL representation of the given Document to the provided
// writer. By default, the emitter uses an indent of four spaces and standard
// float formatting. Options can be provided to customize the output.
//   - [WithIndent] to set a custom indent string (default: four spaces).
//   - [WithStringAlwaysQuote] to always quote strings (default: false).
//   - [WithFloatCapitalExponent] to use capital 'E' for exponents (default: false).
//   - [WithFloatMinExponent] to set the minimum exponent for using scientific notation (default: 10).
//   - [WithFloatPlus] to include '+' for positive floats (default: false).
//   - [WithFloatDecimalPoint] to always include a decimal point in floats (default: false).
//   - [WithFloatExponentPlus] to include '+' for positive exponents (default: false).
//   - [WithFloatDecimalOrExponent] to always include either a decimal point or exponent part in floats (default: true).
//     Without this option, integer-like floating points may be reparsed as integers.
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
	}
	for _, opt := range opts {
		opt(e)
	}
	return e.emitDocument(d)
}

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
}

type emitterHints struct {
	// EmitEmptyChildren controls whether to emit an empty children block when
	// the node has no children.
	EmitEmptyChildren bool
}

type EmitterOption func(*emitter)

// WithIndent sets the indent string for the emitter.
func WithIndent(s string) EmitterOption {
	return func(e *emitter) { e.indent = s }
}

// WithStringAlwaysQuote sets whether to always quote strings.
func WithStringAlwaysQuote(v bool) EmitterOption {
	return func(e *emitter) { e.stringAlwaysQuote = v }
}

// WithFloatCapitalExponent sets whether to use capital 'E' for exponents.
func WithFloatCapitalExponent(v bool) EmitterOption {
	return func(e *emitter) { e.floatCapitalExponent = v }
}

// WithFloatMinExponent sets the minimum exponent for using scientific notation.
func WithFloatMinExponent(v int) EmitterOption {
	return func(e *emitter) { e.floatMinExponent = v }
}

// WithFloatPlus sets whether to include '+' for positive floats.
func WithFloatPlus(v bool) EmitterOption {
	return func(e *emitter) { e.floatPlus = v }
}

// WithFloatDecimalPoint sets whether to always include a decimal point in floats.
func WithFloatDecimalPoint(v bool) EmitterOption {
	return func(e *emitter) { e.floatDecimalPoint = v }
}

// WithFloatExponentPlus sets whether to include '+' for positive exponents.
func WithFloatExponentPlus(v bool) EmitterOption {
	return func(e *emitter) { e.floatExponentPlus = v }
}

// WithFloatDecimalOrExponent sets whether to always include either a decimal
// point or exponent part in floats.
func WithFloatDecimalOrExponent(v bool) EmitterOption {
	return func(e *emitter) { e.floatDecimalOrExponent = v }
}

// WithTestSuiteFloatOptions applies float emission options that match the
// upstream KDL test suite expectations. Specifically:
//   - Capital 'E' for exponents
//   - Minimum exponent of 2 (i.e., use scientific notation for exponents >= 2)
//   - Always include a decimal point
//   - Always include a '+' for positive exponents
func WithTestSuiteFloatOptions() EmitterOption {
	return func(e *emitter) {
		e.floatCapitalExponent = true
		e.floatMinExponent = 2
		e.floatDecimalPoint = true
		e.floatExponentPlus = true
	}
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
	if n.TypeAnnotation != nil {
		if err := e.emit("("); err != nil {
			return err
		}
		if err := e.emitString(*n.TypeAnnotation); err != nil {
			return err
		}
		if err := e.emit(")"); err != nil {
			return err
		}
	}

	if err := e.emitString(n.Name); err != nil {
		return err
	}

	for _, a := range n.Arguments {
		if err := e.emit(" "); err != nil {
			return err
		}
		if err := e.emitValue(a); err != nil {
			return err
		}
	}

	props := slices.Clone(n.PropertyOrder)
	slices.Sort(props)
	for _, p := range props {
		if err := e.emit(" "); err != nil {
			return err
		}
		if err := e.emitString(p); err != nil {
			return err
		}
		if err := e.emit("="); err != nil {
			return err
		}
		if err := e.emitValue(n.Properties[p]); err != nil {
			return err
		}
	}

	if len(n.Children.Nodes) > 0 || n.Hints.EmitEmptyChildren {
		if err := e.emit(" {\n"); err != nil {
			return err
		}
		e.indentLevel++
		if err := e.emitDocument(&n.Children); err != nil {
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

func (e *emitter) emitString(s string) error {
	needsQuoting := s == "" || e.stringAlwaysQuote
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
		return e.emit(`"` + escapeString(s) + `"`)
	} else {
		return e.emit(s)
	}
}

func (e *emitter) emitValue(v Value) error {
	if v.TypeAnnotation() != nil {
		if err := e.emit("("); err != nil {
			return err
		}
		if err := e.emitString(*v.TypeAnnotation()); err != nil {
			return err
		}
		if err := e.emit(")"); err != nil {
			return err
		}
	}
	switch val := v.(type) {
	case String:
		return e.emitString(val.value)
	case Integer:
		return e.emit(strconv.FormatInt(val.value, 10))
	case BigInt:
		return e.emit(val.value.String())
	case Float:
		if math.IsNaN(val.value) {
			return e.emit("#nan")
		}
		return e.emitFloat(new(big.Float).SetFloat64(val.value))
	case BigFloat:
		return e.emitFloat(val.value)
	case Boolean:
		if val.value {
			return e.emit("#true")
		} else {
			return e.emit("#false")
		}
	case Null:
		return e.emit("#null")
	default:
		return fmt.Errorf("unknown value type: %T", v)
	}
}

func (e *emitter) emitFloat(f *big.Float) error {
	if f.IsInf() {
		if f.Sign() > 0 {
			return e.emit("#inf")
		} else {
			return e.emit("#-inf")
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
		return fmt.Errorf("failed to format float: %s", s)
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
