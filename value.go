package kdl

import (
	"fmt"
	"math/big"
)

func NewString(value string) String         { return String{value: value} }
func NewInteger(value int64) Integer        { return Integer{value: value} }
func NewFloat(value float64) Float          { return Float{value: value} }
func NewBigInt(value *big.Int) BigInt       { return BigInt{value: value} }
func NewBigFloat(value *big.Float) BigFloat { return BigFloat{value: value} }
func NewBoolean(value bool) Boolean         { return Boolean{value: value} }
func NewNull() Null                         { return Null{} }

// A Value is a KDL value that can be one of several types, including [String],
// [Integer], [Float], [BigInt], [BigFloat], [Boolean], or [Null]. Each value
// may also have an optional type annotation, which can be used to provide
// additional context about the value's type.
//
// Values are immutable. It is an error to modify a Value or
// its attached type annotation after it has been created.
//
// One can cast a Value to a specific type for using helpers like [AsString],
// [AsInt], and [AsPointer].
type Value interface {
	WithTypeAnnotation(ty *string) Value

	// RawValue returns the underlying value of the KDL value as an any. It may
	// be more useful to first cast the Value to a specific KDL type (e.g.,
	// [String], [Integer], etc.) and then call the appropriate Value method.
	RawValue() any
	// TypAnnotation returns the KDL type annotation of the value, if any.
	TypeAnnotation() *string
	// String returns the string representation of the underlying value. For
	// example, for a [String], it returns the string itself, while for an
	// [Integer], it returns a base-10 representation of the integer.
	String() string
}

// A String is a KDL string value with an optional type annotation.
type String struct {
	typeAnnotation *string
	value          string
}

// An Integer is a KDL integer value with an optional type annotation. Integers
// that cannot be represented as a 64-bit integer will be represented as a
// [BigInt] instead.
type Integer struct {
	typeAnnotation *string
	value          int64
}

// A Float is a KDL floating-point value with an optional type annotation.
// Floats that cannot be represented as a 64-bit float will be represented as a
// [BigFloat] instead.
type Float struct {
	typeAnnotation *string
	value          float64
}

// A BigInt is a KDL integer value that can represent arbitrarily large integers,
// used when the integer value cannot be represented as a 64-bit integer.
type BigInt struct {
	typeAnnotation *string
	value          *big.Int
}

// A BigFloat is a KDL floating-point value that can represent arbitrarily
// large/precise floating-point numbers, used when the floating-point value
// cannot be represented as a 64-bit float.
type BigFloat struct {
	typeAnnotation *string
	value          *big.Float
}

// A Boolean is a KDL boolean value with an optional type annotation.
type Boolean struct {
	typeAnnotation *string
	value          bool
}

// A Null is a KDL null value with an optional type annotation.
type Null struct {
	typeAnnotation *string
}

func (x String) RawValue() any   { return x.value }
func (x Integer) RawValue() any  { return x.value }
func (x Float) RawValue() any    { return x.value }
func (x BigInt) RawValue() any   { return x.value }
func (x BigFloat) RawValue() any { return x.value }
func (x Boolean) RawValue() any  { return x.value }
func (x Null) RawValue() any     { return nil }

func (x String) Value() string       { return x.value }
func (x Integer) Value() int64       { return x.value }
func (x Float) Value() float64       { return x.value }
func (x BigInt) Value() *big.Int     { return x.value }
func (x BigFloat) Value() *big.Float { return x.value }
func (x Boolean) Value() bool        { return x.value }

func (x String) TypeString() string {
	return fmt.Sprintf("%sstring(%q)", formatTypeAnnotation(x.typeAnnotation), x.value)
}
func (x Integer) TypeString() string {
	return fmt.Sprintf("%sinteger(%d)", formatTypeAnnotation(x.typeAnnotation), x.value)
}
func (x Float) TypeString() string {
	return fmt.Sprintf("%sfloat(%f)", formatTypeAnnotation(x.typeAnnotation), x.value)
}
func (x BigInt) TypeString() string {
	return fmt.Sprintf("%sbigint(%s)", formatTypeAnnotation(x.typeAnnotation), x.value.String())
}
func (x BigFloat) TypeString() string {
	return fmt.Sprintf("%sbigfloat(%s)", formatTypeAnnotation(x.typeAnnotation), x.value.String())
}
func (x Boolean) TypeString() string {
	return fmt.Sprintf("%sboolean(%t)", formatTypeAnnotation(x.typeAnnotation), x.value)
}
func (n Null) TypeString() string { return formatTypeAnnotation(n.typeAnnotation) + "null" }

func (x String) String() string   { return x.value }
func (x Integer) String() string  { return fmt.Sprintf("%d", x.value) }
func (x Float) String() string    { return fmt.Sprintf("%f", x.value) }
func (x BigInt) String() string   { return x.value.String() }
func (x BigFloat) String() string { return x.value.String() }
func (x Boolean) String() string  { return fmt.Sprintf("%t", x.value) }
func (n Null) String() string     { return "null" }

func formatTypeAnnotation(ty *string) string {
	if ty == nil {
		return ""
	}
	if *ty == "" {
		return `("")`
	}
	return fmt.Sprintf("(%s)", *ty)
}

func (s String) WithTypeAnnotation(ty *string) Value {
	s.typeAnnotation = ty
	return s
}
func (n Integer) WithTypeAnnotation(ty *string) Value {
	n.typeAnnotation = ty
	return n
}
func (f Float) WithTypeAnnotation(ty *string) Value {
	f.typeAnnotation = ty
	return f
}
func (b BigInt) WithTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (b BigFloat) WithTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (b Boolean) WithTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (n Null) WithTypeAnnotation(ty *string) Value {
	n.typeAnnotation = ty
	return n
}

func (s String) TypeAnnotation() *string   { return s.typeAnnotation }
func (n Integer) TypeAnnotation() *string  { return n.typeAnnotation }
func (f Float) TypeAnnotation() *string    { return f.typeAnnotation }
func (b BigInt) TypeAnnotation() *string   { return b.typeAnnotation }
func (b BigFloat) TypeAnnotation() *string { return b.typeAnnotation }
func (b Boolean) TypeAnnotation() *string  { return b.typeAnnotation }
func (n Null) TypeAnnotation() *string     { return n.typeAnnotation }
