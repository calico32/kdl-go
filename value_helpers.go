package kdl

import (
	"fmt"
	"math"
	"math/big"
)

type keyType interface{ ~string | ~int }

// Get gets an argument or property from a KDL node, depending on the type of
// the key (integer index for arguments, string for properties).
//
// If the key is missing, Get returns nil.
//
// Get panics if node is nil, or if K resolves to a type other than ~string or
// ~int at runtime.
//
// Deprecated: use [Node.Arg] and [Node.Prop] instead, which provide more
// explicit APIs for accessing arguments and properties, respectively, and avoid
// the need for runtime type checks on the key.
func Get[K keyType](node *Node, key K) *Value {
	if node == nil {
		panic("kdl.Get: nil node")
	}

	switch key := any(key).(type) {
	case string:
		v := node.Prop(key)
		if !v.IsValid() {
			return nil
		}
		return &v
	case int:
		v := node.Arg(key)
		if !v.IsValid() {
			return nil
		}
		return &v
	default:
		panic(fmt.Sprintf("kdl.Get: unsupported key type %T", key))
	}
}

// Set sets an argument or property on a KDL node, depending on the type of the
// key (integer index for arguments, string for properties).
//
// When setting an argument, if the index is greater than the current number of
// arguments, the Arguments slice will be extended to accommodate the new index,
// filling in any missing indices with KDL nulls. When setting a property, if
// the property does not already exist, it will be added to the PropertyOrder
// slice to maintain the order of properties.
//
// Set panics if node is nil, if a negative integer index is supplied, or if K
// resolves to a type other than ~string or ~int at runtime.
//
// Deprecated: use [Node.SetArg] and [Node.SetProp] instead, which provide more
// explicit APIs for setting arguments and properties, respectively, and avoid
// the need for runtime type checks on the key.
func Set[K keyType](node *Node, key K, value Value) {
	if node == nil {
		panic("kdl.Set: nil node")
	}

	switch key := any(key).(type) {
	case string:
		node.SetProp(key, value)
	case int:
		node.SetArg(key, value)
	default:
		panic(fmt.Sprintf("kdl.Set: unsupported key type %T", key))
	}
}

// GetKV gets the first child with the given name from the KDL document and
// returns its first argument.
//
// If no such child exists, (nil, nil) is returned. If the child does not have
// exactly one argument, an error is returned. GetKV panics if document is nil.
//
// Deprecated: use [Document.GetKV] instead.
func GetKV(document *Document, name string) (*Value, error) {
	if document == nil {
		panic("kdl.GetKV: nil document")
	}

	v, err := document.GetKV(name)
	if err != nil {
		return nil, err
	}
	if !v.IsValid() {
		return nil, nil
	}
	return &v, nil
}

type intoValue interface {
	~string |
		~int | ~int16 | ~int32 | ~int64 |
		~uint | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~bool |
		~*big.Int | ~*big.Float |
		~*Value | any
}

// NewValue wraps a raw value with its corresponding KDL value type. It panics
// if the value cannot be wrapped — that is, if [TryNewValue] would return a
// non-nil error for v. Use [TryNewValue] when v's type is not known to be
// supported.
func NewValue[T intoValue](v T) Value {
	val, err := TryNewValue(v)
	if err != nil {
		panic(err)
	}
	return val
}

// TryNewValue attempts to wrap a raw value with its corresponding KDL value
// type, returning an error if the value cannot be wrapped. It supports any type
// that implements [ValueMarshaler], as well as the following built-in types:
//
//   - string (wrapped as [String])
//   - int, int8, int16, int32, int64 (wrapped as [Int] or [BigInt] kind depending on size)
//   - uint, uint8, uint16, uint32, uint64 (wrapped as [Int] or [BigInt] kind depending on size)
//   - float32, float64 (wrapped as [Float])
//   - bool (wrapped as [Bool])
//   - *big.Int, *big.Float (wrapped as [BigInt] and [BigFloat], respectively)
//   - *Value (if the pointer is nil, it is treated as a KDL [Null]; otherwise,
//     the pointed-to Value is used)
func TryNewValue[T intoValue](v T) (Value, error) {
	switch v := any(v).(type) {
	case ValueMarshaler:
		return v.MarshalKDLValue()
	case string:
		return NewString(v), nil
	case int:
		return NewInt(v), nil
	case int8:
		return NewInt(int(v)), nil
	case int16:
		return NewInt(int(v)), nil
	case int32:
		return NewInt(int(v)), nil
	case int64:
		if v >= math.MinInt && v <= math.MaxInt {
			return NewInt(int(v)), nil
		}
		return NewBigInt(big.NewInt(v)), nil
	case uint:
		if v <= math.MaxInt {
			return NewInt(int(v)), nil
		}
		return NewBigInt(new(big.Int).SetUint64(uint64(v))), nil
	case uint8:
		return NewInt(int(v)), nil
	case uint16:
		return NewInt(int(v)), nil
	case uint32:
		if uint64(v) <= math.MaxInt {
			return NewInt(int(v)), nil
		}
		return NewBigInt(new(big.Int).SetUint64(uint64(v))), nil
	case uint64:
		if v <= math.MaxInt {
			return NewInt(int(v)), nil
		}
		return NewBigInt(new(big.Int).SetUint64(v)), nil
	case float32:
		return NewFloat(float64(v)), nil
	case float64:
		return NewFloat(v), nil
	case bool:
		return NewBool(v), nil
	case *big.Int:
		return NewBigInt(v), nil
	case *big.Float:
		return NewBigFloat(v), nil
	case *Value:
		if v == nil {
			return NewNull(), nil
		}
		return *v, nil
	default:
		return Value{}, fmt.Errorf("kdl.NewValue: unsupported type %T", v)
	}
}
