package kdl

import (
	"fmt"
	"io"
)

var (
	// ErrStrict is the base error type for strict mode errors. It can be
	// used with [errors.Is] to check for strict mode errors.
	ErrStrict = fmt.Errorf("strict mode error")
)

// Decode reads a KDL document from r and unmarshals it into v. If v implements
// the [DocumentUnmarshaler] interface, that is used to unmarshal the document.
// Otherwise, v must be a pointer to a struct, interface, or map, and the
// document's nodes are unmarshaled into v's fields, properties, or map entries.
//
// # Structs
//
// For struct targets, nodes are mapped to struct fields by name or tag. If a
// struct field's type F implements the [Unmarshaler] interface, that is used to
// unmarshal the node. Otherwise, F must be a pointer to:
//   - a struct, in which case the node's arguments, properties, and children are
//     mapped to struct fields
//   - any, in which case the node is unmarshaled into a map[string]any
//   - a [value type] (see below), in which case the node's first argument is
//     unmarshaled into the value
//   - a map[T]U where T is string/any integer type/any unsigned integer type,
//     and U is a [value type], in which case the node's arguments, properties,
//     and children are unmarshaled into map entries
//   - a []T where T is a [value type], in which case the node's arguments are
//     unmarshaled into slice elements
//   - an array [N]T where T is a [value type], in which case the node's
//     arguments are unmarshaled into array elements (the node must have exactly
//     N elements)
//
// # Struct Tags
//
// Unmarshaling behavior for struct fields can be customized using the
// `kdl:"..."` struct tag. The tag commonly specifies the lowercase name of the
// node that maps to it (e.g., `kdl:"host"`). Additionally, the following flags
// can be used:
//   - multiple: indicates that the node can appear multiple times and each
//     should be mapped to a single slice element (without it, the first node's
//     arguments are each unmarshaled into slice elements). Valid only on slice
//     fields, ignored otherwise.
//   - strict: indicates that strict mode should be enabled for this field,
//     requiring a value to be present when unmarshaling and disallowing any
//     implicit conversions for values.
//   - arg/argument: indicates that the field should receive a single argument by
//     position. Multiple fields can be marked with this flag to receive multiple
//     arguments. Valid only on non-slice, non-map types.
//   - args/arguments: indicates that the field should receive any arguments not
//     mapped to other fields. Valid only on slice, array, or map types. Can only
//     be used once per struct.
//   - prop/property: indicates that the field should only match a property, not a
//     child node. Valid only on scalar fields.
//   - props/properties: indicates that the field should receive any properties not
//     mapped to other fields. Valid only on map or struct types. Can only be used
//     once per struct.
//   - child: indicates that the field should only match child nodes, not
//     properties.
//   - children: indicates that the field should receive any child nodes not
//     mapped to other fields. Valid only on slice, array, or map types. Can only
//     be used once per struct.
//   - presence: indicates that for bool fields, the presence of a child node
//     with no arguments is interpreted as true. Valid only on bool fields.
//
// An additional tag, omitzero, can be used to control marshaling behavior but
// is ignored during unmarshaling.
//
// For example:
//
//	type Config struct {
//	    Hosts []Host `kdl:"host,multiple"`
//	}
//	type Host struct {
//	    Id       string            `kdl:",arg,strict"`
//	    Network  string            `kdl:"network,prop"`
//	    User     string            `kdl:"user,child"`
//	    Hostname string            `kdl:"hostname,child,strict"`
//	    Port     int               `kdl:"port,child"`
//	    Extra    map[string]string `kdl:",children"`
//	    Internal bool              `kdl:"internal,presence"`
//	}
//
// # Values
//
// A [value type] is any of:
//   - string,
//   - int/int8/int16/int32/int64/[*big.Int],
//   - uint/uint8/uint16/uint32/uint64,
//   - float32/float64/[*big.Float],
//   - bool, ("1", "t", "T", "true", "TRUE", "True", "y", "Y", "yes", "YES", "Yes" for true;
//     "0", "f", "F", "false", "FALSE", "False", "n", "N", "no", "NO", "No" for false)
//   - [time.Time],
//   - [time.Duration],
//   - any (in which case the result of [Value.RawValue] is used except for KDL
//     integers, which are by default int instead of int64),
//   - or a type implementing [ValueUnmarshaler] (in which case
//     [ValueUnmarshaler.UnmarshalKDLValue] is used to unmarshal the value).
//
// Values will be formatted or parsed for canonical conversions, like string to
// int, string to bool, etc. Use [WithStrict] to disable such conversions.
//
// An error is returned if the KDL value cannot be converted to the target type
// because of a type mismatch or overflow.
//
// # Unused Data
//
// Decode ignores any nodes, properties, or arguments that cannot be mapped to
// the target value. To enable strict mode, which returns an error if any data
// cannot be mapped, pass the [WithStrict] option to [Decode].
func Decode(r io.Reader, v any, opts ...DecodeOption) error {
	parseOpts, unmarshalOpts := splitDecodeOptions(opts)
	doc, err := Parse(r, parseOpts...)
	if err != nil {
		return err
	}
	return UnmarshalDocument(doc, v, unmarshalOpts...)
}

// DecodeStrict is like [Decode] but enables strict mode, which returns an error
// if any nodes, properties, or arguments cannot be mapped to the target value.
// It also disables any canonical conversions when unmarshaling values.
//
// Deprecated: Use [Decode] with the [WithStrict] option instead.
func DecodeStrict(r io.Reader, v any) error {
	return Decode(r, v, WithStrict(true))
}

// DecodeNamed is like [Decode] but allows specifying the name of the input
// source, which is used in error messages and locations.
//
// Deprecated: Use [Decode] with the [WithSourceName] option instead.
func DecodeNamed(name string, r io.Reader, v any) error {
	return Decode(r, v, WithSourceName(name))
}

// DecodeNamedStrict is like [DecodeStrict] but allows specifying the name of
// the input source, which is used in error messages and locations.
//
// Deprecated: Use [Decode] with the [WithSourceName] and [WithStrict] options
// instead.
func DecodeNamedStrict(name string, r io.Reader, v any) error {
	return Decode(r, v, WithSourceName(name), WithStrict(true))
}

// DecodeString is a convenience wrapper around [Decode] for string inputs.
func DecodeString(src string, v any, opts ...DecodeOption) error {
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
	doc, err := ParseString(src, parseOpts...)
	if err != nil {
		return err
	}
	return UnmarshalDocument(doc, v, unmarshalOpts...)
}

// Unmarshal unmarshals n into v. See [Decode] for details.
func Unmarshal(n *Node, v any, opts ...UnmarshalOption) error {
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalKDL(n)
	}
	d := &decoder{strict: false}
	for _, opt := range opts {
		opt.applyUnmarshaler(d)
	}
	target, err := unwrapPointerOrInterface(v)
	if err != nil {
		return err
	}

	return d.unmarshalNode(n, structTag{}, target)
}

// UnmarshalStrict unmarshals n into v in strict mode, which returns an error if
// any nodes, properties, or arguments cannot be mapped to the target value. It
// also disables any canonical conversions when unmarshaling values. See
// [DecodeStrict] for details.
//
// Deprecated: Use [Unmarshal] with the [WithStrict] option instead.
func UnmarshalStrict(n *Node, v any) error {
	return Unmarshal(n, v, WithStrict(true))
}

// UnmarshalDocument unmarshals the given KDL [Document] into v. See [Decode]
// for details.
func UnmarshalDocument(doc *Document, v any, opts ...UnmarshalOption) error {
	return unmarshalDocument(doc, v, opts...)
}

// UnmarshalDocumentStrict unmarshals the given KDL [Document] into v in strict
// mode, which returns an error if any nodes, properties, or arguments cannot
// be mapped to the target value. It also disables any canonical conversions
// when unmarshaling values. See [DecodeStrict] for details.
//
// Deprecated: Use [UnmarshalDocument] with the [WithStrict] option instead.
func UnmarshalDocumentStrict(doc *Document, v any) error {
	return UnmarshalDocument(doc, v, WithStrict(true))
}

// Located[T] is a wrapper type that can be used to unmarshal a T along with its
// source location. When unmarshaling into a Located[T], the decoder will
// unmarshal the value into the Value field and set the Start and End fields to
// the value's source range (either the Node or Value range, depending on the
// context). Located[T] is transparent when marshaling.
type Located[T any] struct {
	Value      T
	Start, End Location
}

// internal interface to detect Located[T] - cannot reflect type eq generics
type locatedType interface {
	locatedType()
}

var _ locatedType = Located[any]{}

func (Located[T]) locatedType() {}
