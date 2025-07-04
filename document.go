package kdl

// #cgo CFLAGS: -I/usr/local/include
// #cgo LDFLAGS: -L/usr/local/lib -lkdl
// #include "kdl.h"
// #include <stdlib.h>
// #include <string.h>
import "C"

import (
	"fmt"
	"math/big"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("no such key")
)

// A Document is a collection of nodes.
type Document struct {
	Nodes []*Node
}

func NewDocument(nodes ...*Node) *Document {
	return &Document{Nodes: nodes}
}

// AddNode adds a node to the document and returns the document.
func (d *Document) AddNode(node *Node) *Document {
	d.Nodes = append(d.Nodes, node)
	return d
}

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
// One can cast a Value to a specific type for using helpers like [AsString], [AsInt], and [AsPointer].
type Value interface {
	c() (C.kdl_value, func())
	withTypeAnnotation(ty *string) Value

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

func (s String) withTypeAnnotation(ty *string) Value {
	s.typeAnnotation = ty
	return s
}
func (n Integer) withTypeAnnotation(ty *string) Value {
	n.typeAnnotation = ty
	return n
}
func (f Float) withTypeAnnotation(ty *string) Value {
	f.typeAnnotation = ty
	return f
}
func (b BigInt) withTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (b BigFloat) withTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (b Boolean) withTypeAnnotation(ty *string) Value {
	b.typeAnnotation = ty
	return b
}
func (n Null) withTypeAnnotation(ty *string) Value {
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

func newKdlValue(v C.kdl_value) (Value, error) {
	var x Value
	switch v._type {
	case C.KDL_TYPE_NULL:
		x = Null{}
	case C.KDL_TYPE_STRING:
		x = String{value: goString((*C.kdl_str)(unsafe.Pointer(&v.anon0)))}
	case C.KDL_TYPE_NUMBER:
		kdlNum := (*C.kdl_number)(unsafe.Pointer(&v.anon0))
		ptr := unsafe.Pointer(&kdlNum.anon0)
		switch kdlNum._type {
		case C.KDL_NUMBER_TYPE_INTEGER:
			x = Integer{value: int64(*(*C.int64_t)(ptr))}
		case C.KDL_NUMBER_TYPE_FLOATING_POINT:
			x = Float{value: float64(*(*C.double)(ptr))}
		case C.KDL_NUMBER_TYPE_STRING_ENCODED:
			s := goString((*C.kdl_str)(ptr))
			if _, ok := new(big.Int).SetString(s, 10); ok {
				i := new(big.Int)
				i.SetString(s, 10)
				x = BigInt{value: i}
			} else {
				f := new(big.Float)
				f.Parse(s, 10)
				x = BigFloat{value: f}
			}
		}
	case C.KDL_TYPE_BOOLEAN:
		b := (*C.bool)(unsafe.Pointer(&v.anon0))
		x = Boolean{value: bool(*b)}
	}

	if v.type_annotation.data != nil {
		bytes := C.GoBytes(unsafe.Pointer(v.type_annotation.data), C.int(v.type_annotation.len))
		str := string(bytes)
		x = x.withTypeAnnotation(&str)
	}

	return x, nil
}

func withTypeAnnotation(v C.kdl_value, annot *string, free func()) (C.kdl_value, func()) {
	if annot == nil {
		return v, func() {
			if free != nil {
				free()
			}
		}
	}

	var freeTypeAnnot func()
	v.type_annotation, freeTypeAnnot = kdlString(*annot)
	return v, func() {
		if free != nil {
			free()
		}
		freeTypeAnnot()
	}
}

func (s String) c() (C.kdl_value, func()) {
	str, freeStr := kdlString(s.value)
	v := C.kdl_value{_type: C.KDL_TYPE_STRING}
	*(*C.kdl_str)(unsafe.Pointer(&v.anon0)) = str

	return withTypeAnnotation(v, s.typeAnnotation, freeStr)
}

func (n Integer) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_INTEGER}
	*(*C.int64_t)(unsafe.Pointer(&number.anon0)) = C.int64_t(n.value)

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, n.typeAnnotation, nil)
}

func (n Float) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_FLOATING_POINT}
	*(*C.double)(unsafe.Pointer(&number.anon0)) = C.double(n.value)

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, n.typeAnnotation, nil)
}

func (b BigInt) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_STRING_ENCODED}
	str, free := kdlString(b.value.String())
	*(*C.kdl_str)(unsafe.Pointer(&number.anon0)) = str

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, b.typeAnnotation, free)
}

func (b BigFloat) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_STRING_ENCODED}
	str, free := kdlString(b.value.Text('G', 10))
	*(*C.kdl_str)(unsafe.Pointer(&number.anon0)) = str

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, b.typeAnnotation, free)
}

func (b Boolean) c() (C.kdl_value, func()) {
	v := C.kdl_value{_type: C.KDL_TYPE_BOOLEAN}
	*(*C.bool)(unsafe.Pointer(&v.anon0)) = C.bool(b.value)

	return withTypeAnnotation(v, b.typeAnnotation, nil)
}

func (n Null) c() (C.kdl_value, func()) {
	kdlNull := C.kdl_value{_type: C.KDL_TYPE_NULL}

	return withTypeAnnotation(kdlNull, n.typeAnnotation, nil)
}
