package kdl

import (
	"fmt"
	"math"
	"math/big"
)

type ValueKind int

const (
	// A String is a KDL string value with an optional type annotation.
	String ValueKind = iota
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
	// A Null is a KDL null value with an optional type annotation.
	Null
)

var valueKindNames = [...]string{
	String:   "String",
	Int:      "Int",
	Float:    "Float",
	BigInt:   "BigInt",
	BigFloat: "BigFloat",
	Bool:     "Bool",
	Null:     "Null",
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
	ty        string
	typeValid bool
	kind      ValueKind
	location  Location
	raw       any
}

func (v Value) TypeAnnotation() (string, bool) { return v.ty, v.typeValid }
func (v Value) Kind() ValueKind                { return v.kind }
func (v Value) RawValue() any                  { return v.raw }
func (v Value) Location() Location             { return v.location }

func (v Value) WithTypeAnnotation(ty string, valid bool) Value {
	v.ty = ty
	v.typeValid = valid
	return v
}

func (v Value) WithLocation(loc Location) Value {
	v.location = loc
	return v
}

// String returns the underlying string value if this value is of kind [String].
// Unlike other accessor methods, it does not panic if the Value is not of kind
// String and instead returns a string in the format "<kdl.KIND %v>", where KIND
// is the ValueKind and %v is the raw value formatted by fmt.Printf.
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
		panic("Int() on non-integer Value")
	}
	return v.raw.(int)
}

// Float returns the underlying float64 value if this value is of kind [Float].
// It panics if the Value is not of kind Float.
func (v Value) Float() float64 {
	if v.kind != Float {
		panic("Float() on non-float Value")
	}
	return v.raw.(float64)
}

// BigInt returns the underlying *big.Int value if this value is of kind
// [BigInt]. It panics if the Value is not of kind BigInt.
func (v Value) BigInt() *big.Int {
	if v.kind != BigInt {
		panic("BigInt() on non-BigInt Value")
	}
	return new(big.Int).Set(v.raw.(*big.Int))
}

// BigFloat returns the underlying *big.Float value if this value is of kind
// [BigFloat]. It panics if the Value is not of kind BigFloat.
func (v Value) BigFloat() *big.Float {
	if v.kind != BigFloat {
		panic("BigFloat() on non-BigFloat Value")
	}
	return new(big.Float).Set(v.raw.(*big.Float))
}

// Bool returns the underlying bool value if this value is of kind [Bool]. It
// panics if the Value is not of kind Bool.
func (v Value) Bool() bool {
	if v.kind != Bool {
		panic("Bool() on non-bool Value")
	}
	return v.raw.(bool)
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
