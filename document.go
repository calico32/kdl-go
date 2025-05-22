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
)

type Document struct {
	Nodes []*Node
}

type Node struct {
	Name           string
	TypeAnnotation *string
	Arguments      []Value
	Properties     map[string]Value
	PropertyOrder  []string
	Children       []*Node
}

type Value interface {
	c() (C.kdl_value, func())
	withTypeAnnotation(ty *string) Value

	Value() any
	TypeAnnotation() *string
}

type String struct {
	typeAnnot *string
	value     string
}
type Integer struct {
	typeAnnot *string
	value     int64
}
type Float struct {
	typeAnnot *string
	value     float64
}
type BigInt struct {
	typeAnnot *string
	value     *big.Int
}
type BigFloat struct {
	typeAnnot *string
	value     *big.Float
}
type Boolean struct {
	typeAnnot *string
	value     bool
}
type Null struct {
	typeAnnot *string
}

func (x String) Value() any   { return x.value }
func (x Integer) Value() any  { return x.value }
func (x Float) Value() any    { return x.value }
func (x BigInt) Value() any   { return x.value }
func (x BigFloat) Value() any { return x.value }
func (x Boolean) Value() any  { return x.value }
func (x Null) Value() any     { return nil }

func (x String) String() string {
	return fmt.Sprintf("%sstring(%q)", formatTypeAnnot(x.typeAnnot), x.value)
}
func (x Integer) String() string {
	return fmt.Sprintf("%sinteger(%d)", formatTypeAnnot(x.typeAnnot), x.value)
}
func (x Float) String() string {
	return fmt.Sprintf("%sfloat(%f)", formatTypeAnnot(x.typeAnnot), x.value)
}
func (x BigInt) String() string {
	return fmt.Sprintf("%sbigint(%s)", formatTypeAnnot(x.typeAnnot), x.value.String())
}
func (x BigFloat) String() string {
	return fmt.Sprintf("%sbigfloat(%s)", formatTypeAnnot(x.typeAnnot), x.value.String())
}
func (x Boolean) String() string {
	return fmt.Sprintf("%sboolean(%t)", formatTypeAnnot(x.typeAnnot), x.value)
}
func (n Null) String() string { return formatTypeAnnot(n.typeAnnot) + "null" }

func formatTypeAnnot(ty *string) string {
	if ty == nil {
		return ""
	}
	if *ty == "" {
		return `("")`
	}
	return fmt.Sprintf("(%s)", *ty)
}

func (s String) withTypeAnnotation(ty *string) Value {
	s.typeAnnot = ty
	return s
}
func (n Integer) withTypeAnnotation(ty *string) Value {
	n.typeAnnot = ty
	return n
}
func (f Float) withTypeAnnotation(ty *string) Value {
	f.typeAnnot = ty
	return f
}
func (b BigInt) withTypeAnnotation(ty *string) Value {
	b.typeAnnot = ty
	return b
}
func (b BigFloat) withTypeAnnotation(ty *string) Value {
	b.typeAnnot = ty
	return b
}
func (b Boolean) withTypeAnnotation(ty *string) Value {
	b.typeAnnot = ty
	return b
}
func (n Null) withTypeAnnotation(ty *string) Value {
	n.typeAnnot = ty
	return n
}

func (s String) TypeAnnotation() *string   { return s.typeAnnot }
func (n Integer) TypeAnnotation() *string  { return n.typeAnnot }
func (f Float) TypeAnnotation() *string    { return f.typeAnnot }
func (b BigInt) TypeAnnotation() *string   { return b.typeAnnot }
func (b BigFloat) TypeAnnotation() *string { return b.typeAnnot }
func (b Boolean) TypeAnnotation() *string  { return b.typeAnnot }
func (n Null) TypeAnnotation() *string     { return n.typeAnnot }

func newKdlValue(v C.kdl_value) (Value, error) {
	var x Value
	switch v._type {
	case C.KDL_TYPE_NULL:
		x = Null{}
	case C.KDL_TYPE_STRING:
		x = String{value: GoString((*C.kdl_str)(unsafe.Pointer(&v.anon0)))}
	case C.KDL_TYPE_NUMBER:
		kdlNum := (*C.kdl_number)(unsafe.Pointer(&v.anon0))
		ptr := unsafe.Pointer(&kdlNum.anon0)
		switch kdlNum._type {
		case C.KDL_NUMBER_TYPE_INTEGER:
			x = Integer{value: int64(*(*C.int64_t)(ptr))}
		case C.KDL_NUMBER_TYPE_FLOATING_POINT:
			x = Float{value: float64(*(*C.double)(ptr))}
		case C.KDL_NUMBER_TYPE_STRING_ENCODED:
			s := GoString((*C.kdl_str)(ptr))
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
	v.type_annotation, freeTypeAnnot = KdlString(*annot)
	return v, func() {
		if free != nil {
			free()
		}
		freeTypeAnnot()
	}
}

func (s String) c() (C.kdl_value, func()) {
	str, freeStr := KdlString(s.value)
	v := C.kdl_value{_type: C.KDL_TYPE_STRING}
	*(*C.kdl_str)(unsafe.Pointer(&v.anon0)) = str

	return withTypeAnnotation(v, s.typeAnnot, freeStr)
}

func (n Integer) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_INTEGER}
	*(*C.int64_t)(unsafe.Pointer(&number.anon0)) = C.int64_t(n.value)

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, n.typeAnnot, nil)
}

func (n Float) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_FLOATING_POINT}
	*(*C.double)(unsafe.Pointer(&number.anon0)) = C.double(n.value)

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, n.typeAnnot, nil)
}

func (b BigInt) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_STRING_ENCODED}
	str, free := KdlString(b.value.String())
	*(*C.kdl_str)(unsafe.Pointer(&number.anon0)) = str

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, b.typeAnnot, free)
}

func (b BigFloat) c() (C.kdl_value, func()) {
	number := C.kdl_number{_type: C.KDL_NUMBER_TYPE_STRING_ENCODED}
	str, free := KdlString(b.value.Text('G', 10))
	*(*C.kdl_str)(unsafe.Pointer(&number.anon0)) = str

	v := C.kdl_value{_type: C.KDL_TYPE_NUMBER}
	*(*C.kdl_number)(unsafe.Pointer(&v.anon0)) = number

	return withTypeAnnotation(v, b.typeAnnot, free)
}

func (b Boolean) c() (C.kdl_value, func()) {
	v := C.kdl_value{_type: C.KDL_TYPE_BOOLEAN}
	*(*C.bool)(unsafe.Pointer(&v.anon0)) = C.bool(b.value)

	return withTypeAnnotation(v, b.typeAnnot, nil)
}

func (n Null) c() (C.kdl_value, func()) {
	kdlNull := C.kdl_value{_type: C.KDL_TYPE_NULL}

	return withTypeAnnotation(kdlNull, n.typeAnnot, nil)
}
