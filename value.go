package kdl

import (
	"fmt"
	"math"
	"math/big"
)

type ValueKind uint8

const (
	// Invalid represents an invalid/zero Value. Most functions never return
	// invalid Values (if the associated error is nil), but it may be produced
	// by certain operations like accessing missing properties/out-of-bounds
	// arguments (in these cases, the function's documentation will clearly
	// state so). Most operations on an Invalid Value will panic or cause
	// unpredictable behavior; check associated error return values or use
	// [Value.IsValid] before working on a Value that may be invalid.
	Invalid ValueKind = iota
	// A Null is a KDL null value with an optional type annotation.
	Null
	// A String is a KDL string value with an optional type annotation.
	String
	// An Int is a KDL integer value with an optional type annotation. Integers
	// that cannot be represented as a 64-bit integer will be represented as a
	// [BigInt] instead.
	Int
	// A Float is a KDL floating-point value with an optional type annotation.
	// Floats that cannot be represented as a 64-bit float will be represented as a
	// [BigFloat] instead.
	Float
	// A BigInt is a KDL integer value that can represent arbitrarily large integers,
	// used when the integer value cannot be represented as a 64-bit integer.
	BigInt
	// A BigFloat is a KDL floating-point value that can represent arbitrarily
	// large/precise floating-point numbers, used when the floating-point value
	// cannot be represented as a 64-bit float.
	BigFloat
	// A Bool is a KDL boolean value with an optional type annotation.
	Bool
)

var valueKindNames = [...]string{
	Invalid:  "Invalid",
	Null:     "Null",
	String:   "String",
	Int:      "Int",
	Float:    "Float",
	BigInt:   "BigInt",
	BigFloat: "BigFloat",
	Bool:     "Bool",
}

func (k ValueKind) String() string {
	if int(k) < 0 || int(k) >= len(valueKindNames) {
		return fmt.Sprintf("ValueKind(%d)", int(k))
	}
	return valueKindNames[k]
}

// A Value is a KDL value that can be one of several kinds, including [String],
// [Int], [Float], [BigInt], [BigFloat], [Bool], or [Null]. A value
// may also have an optional type annotation, which can be used to provide
// additional context about the value's type.
type Value struct {
	typ       string
	typeValid bool
	kind      ValueKind
	raw       any
	src       *valueSourceInfo
}

type valueSourceInfo struct {
	location       Location
	endLocation    Location
	typeAnnotStart Location
	typeAnnotEnd   Location
	// literal holds the exact source text for parsed string values
	// (quoted, raw, or multi-line, including delimiters). Empty for
	// programmatically created values.
	literal string
}

// IsValid reports whether this Value is valid. Most functions never return
// invalid Values (if the associated error is nil), but it may be produced by
// certain operations like accessing missing properties/out-of-bounds arguments
// (in these cases, the function's documentation will clearly state so). Most
// operations on an invalid Value will panic or cause unpredictable behavior;
// check associated error return values or call IsValid before working on a
// Value that may be invalid.
func (v Value) IsValid() bool                  { return v.kind != Invalid && (v.kind == Null || v.raw != nil) }
func (v Value) TypeAnnotation() (string, bool) { return v.typ, v.typeValid }
func (v Value) Kind() ValueKind                { return v.kind }
func (v Value) RawValue() any                  { return v.raw }

// Location returns the source location of the value token, not including any
// type annotation. Returns a zero Location when location tracking is off.
func (v Value) Location() Location {
	if v.src != nil {
		return v.src.location
	}
	return Location{}
}

// EndLocation returns the location of the end (exclusive) of the value token,
// not including any type annotation. Returns a zero Location when location
// tracking is off.
func (v Value) EndLocation() Location {
	if v.src != nil {
		return v.src.endLocation
	}
	return Location{}
}

// TypeAnnotationRange returns the source range of the type annotation content
// (the identifier inside the parentheses, not the parens themselves). ok is
// false when no type annotation is present or location tracking is off.
func (v Value) TypeAnnotationRange() (start, end Location, ok bool) {
	if !v.typeValid || v.src == nil || v.src.typeAnnotStart.Line == 0 {
		return
	}
	return v.src.typeAnnotStart, v.src.typeAnnotEnd, true
}

func (v Value) WithTypeAnnotation(ty string, valid bool) Value {
	v.typ = ty
	v.typeValid = valid
	return v
}

// Literal returns the original source text of the string/numeric literal, if
// this value was produced by parsing a KDL document and the literal text
// contained meaningful syntax that should be preserved when round-tripping
// (e.g. raw/multiline strings or different integer formats). It returns ("",
// false) if no such literal is available or applicable for this value.
func (v Value) Literal() (string, bool) {
	if v.src == nil || v.src.literal == "" {
		return "", false
	}
	return v.src.literal, true
}

// WithLiteral returns a new Value with the original source text of a
// string/numeric literal that this Value should round-trip to when formatting.
// For non-string and non-numeric values, the literal is ignored and does
// nothing.
func (v Value) WithLiteral(s string) Value {
	// copy (or alloc) src info
	var src valueSourceInfo
	if v.src != nil {
		src = *v.src
	}
	src.literal = s
	v.src = &src
	return v
}

// String returns the underlying string value if this value is of kind [String].
// Unlike the other typed accessor methods on Value, it does not panic on a kind
// mismatch to safely implement [fmt.Stringer] for all kinds; instead it returns
// a debug representation in the format "<kdl.KIND %v>", where KIND is the
// [ValueKind] and %v is the raw value formatted by fmt.Printf.
func (v Value) String() string {
	if v.kind == String {
		return v.raw.(string)
	}

	return fmt.Sprintf("<kdl.%s %v>", v.kind, v.raw)
}

// Int returns the underlying int64 value if this value is of kind [Int]. It
// panics if the Value is not of kind Int.
func (v Value) Int() int {
	if v.kind != Int {
		panic("kdl.Value: Int called on non-integer Value")
	}
	return v.raw.(int)
}

// Float returns the underlying float64 value if this value is of kind [Float].
// It panics if the Value is not of kind Float.
func (v Value) Float() float64 {
	if v.kind != Float {
		panic("kdl.Value: Float called on non-float Value")
	}
	return v.raw.(float64)
}

// BigInt returns the underlying *big.Int value if this value is of kind
// [BigInt]. It panics if the Value is not of kind BigInt.
func (v Value) BigInt() *big.Int {
	if v.kind != BigInt {
		panic("kdl.Value: BigInt called on non-BigInt Value")
	}
	return new(big.Int).Set(v.raw.(*big.Int))
}

// BigFloat returns the underlying *big.Float value if this value is of kind
// [BigFloat]. It panics if the Value is not of kind BigFloat.
func (v Value) BigFloat() *big.Float {
	if v.kind != BigFloat {
		panic("kdl.Value: BigFloat called on non-BigFloat Value")
	}
	return new(big.Float).Set(v.raw.(*big.Float))
}

// Bool returns the underlying bool value if this value is of kind [Bool]. It
// panics if the Value is not of kind Bool.
func (v Value) Bool() bool {
	if v.kind != Bool {
		panic("kdl.Value: Bool called on non-bool Value")
	}
	return v.raw.(bool)
}

// Equal returns whether this Value is equal to another Value, comparing only
// the kind, raw value, and type annotation (if present) for equality (using ==,
// or .Cmp for big.Int/big.Float). It does not consider source location or
// literal information. Invalid Values are also considered equal to each other,
// but not equal to any valid Value.
func (v Value) Equal(other Value) bool {
	if v.kind != other.kind {
		return false
	}
	if v.typeValid != other.typeValid {
		return false
	}
	if v.typeValid && v.typ != other.typ {
		return false
	}
	switch v.kind {
	case Invalid:
		return true
	case Null:
		return true
	case String, Int, Float, Bool:
		return v.raw == other.raw
	case BigInt:
		return v.raw.(*big.Int).Cmp(other.raw.(*big.Int)) == 0
	case BigFloat:
		return v.raw.(*big.Float).Cmp(other.raw.(*big.Float)) == 0
	default:
		panic(fmt.Sprintf("kdl.Value: invalid ValueKind in Equal: %s", v.kind))
	}
}

// NewString creates a new KDL string Value.
func NewString(s string) Value {
	return Value{kind: String, raw: s}
}

// NewInt creates a new KDL integer Value.
func NewInt(i int) Value {
	return Value{kind: Int, raw: i}
}

// NewFloat creates a new KDL floating-point Value.
func NewFloat(f float64) Value {
	return Value{kind: Float, raw: f}
}

// NewBigInt creates a new KDL big integer Value. If bi is nil, the value will
// be initialized to 0.
func NewBigInt(bi *big.Int) Value {
	v := new(big.Int)
	if bi != nil {
		v.Set(bi)
	}
	return Value{kind: BigInt, raw: v}
}

// NewBigFloat creates a new KDL big floating-point Value. If bf is nil, the
// value will be initialized to 0.
func NewBigFloat(bf *big.Float) Value {
	v := new(big.Float)
	if bf != nil {
		v.Set(bf)
	}
	return Value{kind: BigFloat, raw: v}
}

// NewBool creates a new KDL boolean Value.
func NewBool(b bool) Value {
	if b {
		return trueValue
	} else {
		return falseValue
	}
}

// NewNull creates a new KDL null Value.
func NewNull() Value {
	return nullValue
}

// Internal predefined values for common constants.
var (
	nullValue   = Value{kind: Null, raw: nil}
	trueValue   = Value{kind: Bool, raw: true}
	falseValue  = Value{kind: Bool, raw: false}
	infValue    = Value{kind: Float, raw: math.Inf(1)}
	negInfValue = Value{kind: Float, raw: math.Inf(-1)}
	nanValue    = Value{kind: Float, raw: math.NaN()}
)
