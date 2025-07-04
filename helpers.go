package kdl

import (
	"fmt"
	"math"
	"math/big"
	"slices"
)

type keyType interface{ ~string | ~int }

// Get gets an argument or property from a KDL node, depending on the type of
// the key (integer index for arguments, string for properties). It applies the
// transformation function fn to the value before returning it. Any errors from transforming the value are returned.
//
// If the key is missing, an sub-error of [ErrNotFound] is returned.
func Get[K keyType, R any](node *Node, key K, fn func(Value) (R, error)) (R, error) {
	var zero R
	if node == nil {
		return zero, fmt.Errorf("node is nil")
	}

	switch key := any(key).(type) {
	case string:
		if val, ok := node.Properties[key]; ok {
			return fn(val)
		} else {
			return zero, fmt.Errorf("property %s: %w", key, ErrNotFound)
		}
	case int:
		if len(node.Arguments) <= key {
			return zero, fmt.Errorf("argument at index %d: %w", key, ErrNotFound)
		}
		return fn(node.Arguments[key])
	default:
		panic(fmt.Sprintf("unsupported key type %T", key))
	}
}

// Set sets an argument or property on a KDL node, depending on the type of the
// key (integer index for arguments, string for properties). It returns an error
// if the key is invalid or if there is an error setting the value.
//
// When setting an argument, if the index is greater than the current
// number of arguments, the Arguments slice will be extended to accommodate the
// new index, filling in any missing indices with KDL nulls.
// When setting a property, if the property does not already exist, it will be
// added to the PropertyOrder slice to maintain the order of properties.
func Set[K keyType](node *Node, key K, value Value) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	switch key := any(key).(type) {
	case string:
		if !slices.Contains(node.PropertyOrder, key) {
			node.PropertyOrder = append(node.PropertyOrder, key)
		}
		node.Properties[key] = value
	case int:
		if key < 0 {
			return fmt.Errorf("invalid argument index %d", key)
		}
		if key >= len(node.Arguments) {
			// Extend the Arguments slice to accommodate the new index
			for i := len(node.Arguments); i <= key; i++ {
				node.Arguments = append(node.Arguments, Null{})
			}
		}
		node.Arguments[key] = value
	default:
		panic(fmt.Sprintf("unsupported key type %T", key))
	}

	return nil
}

// GetKV gets the first child with the given name from the KDL node and applies
// a transformation to its first argument before returning it.
//
// If no such child exists, it returns an error that with [ErrNotFound] in its tree,
// which can be checked with [errors.Is].
func GetKV[R any](node *Node, name string, fn func(Value) (R, error)) (R, error) {
	if node == nil {
		return *new(R), fmt.Errorf("node is nil")
	}

	for _, child := range node.Children {
		if child.Name == name {
			return fn(child.Arguments[0])
		}
	}

	var zero R
	return zero, fmt.Errorf("child %s: %w", name, ErrNotFound)
}

func CastAll[T any](values []Value, fn func(Value) (T, error)) ([]T, error) {
	if values == nil {
		return nil, fmt.Errorf("values is nil")
	}

	out := make([]T, 0, len(values))
	for _, v := range values {
		val, err := fn(v)
		if err != nil {
			return nil, fmt.Errorf("casting value %s: %w", v.String(), err)
		}
		out = append(out, val)
	}

	return out, nil
}

func AsPointer[T any](fn func(Value) (T, error)) func(Value) (*T, error) {
	return func(v Value) (*T, error) {
		if v == nil {
			return nil, fmt.Errorf("value is nil")
		}

		val, err := fn(v)
		if err != nil {
			return nil, err
		}

		return &val, nil
	}
}

// AsString returns the underlying value of a KDL string or an error if the
// value is not a string.
func AsString(v Value) (string, error) {
	switch v := v.(type) {
	case String:
		return v.value, nil
	default:
		return "", fmt.Errorf("value is not a string")
	}
}

// AsInt64 returns the underlying value of a KDL integer as an int64 or an
// error if the value is not an integer or cannot be represented as an int64.
func AsInt64(v Value) (int64, error) {
	switch v := v.(type) {
	case Integer:
		return v.value, nil
	case BigInt:
		if !v.value.IsInt64() {
			return 0, fmt.Errorf("big.Int value %s cannot be represented as int64", v.value.String())
		}
		return v.value.Int64(), nil
	default:
		return 0, fmt.Errorf("value is not an integer")
	}
}

// AsInt returns the underlying value of a KDL integer as an int or an error if
// the value is not an integer or cannot be represented as an int.
func AsInt(v Value) (int, error) {
	i64, err := AsInt64(v)
	if err != nil {
		return 0, err
	}
	if i64 > math.MaxInt || i64 < math.MinInt {
		return 0, fmt.Errorf("int value %d cannot be represented as int", i64)
	}
	return int(i64), nil
}

// AsFloat64 returns the underlying value of a KDL float as a float64 or an error
// if the value is not a float. If the value is a big.Float and cannot be
// represented as a float64, it returns the closest float64 value and an error
// indicating that the value was not exact.
func AsFloat64(v Value) (float64, error) {
	switch v := v.(type) {
	case Float:
		return v.value, nil
	case BigFloat:
		f64, accuracy := v.value.Float64()
		var err error
		if accuracy != big.Exact {
			err = fmt.Errorf("big.Float value %s cannot be represented as float64", v.value.String())
		}
		return f64, err
	default:
		return 0, fmt.Errorf("value is not a float")
	}

}

// AsBigInt returns the underlying value of a KDL integer as a big.Int, or
// an error if the value is not an integer.
func AsBigInt(v Value) (*big.Int, error) {
	switch v := v.(type) {
	case BigInt:
		return v.value, nil
	case Integer:
		return new(big.Int).SetInt64(v.value), nil
	default:
		return nil, fmt.Errorf("value is not a big.Int")
	}
}

// AsBigFloat returns the underlying value of a KDL float as a big.Float, or
// an error if the value is not a float.
func AsBigFloat(v Value) (*big.Float, error) {
	switch v := v.(type) {
	case BigFloat:
		return v.value, nil
	case Float:
		return big.NewFloat(v.value), nil
	default:
		return nil, fmt.Errorf("value is not a big.Float")
	}
}

// AsBool returns the underlying value of a KDL boolean as a bool or an
// error if the value is not a boolean.
func AsBool(v Value) (bool, error) {
	switch v := v.(type) {
	case Boolean:
		return v.value, nil
	default:
		return false, fmt.Errorf("value is not a boolean")
	}
}

// AsNull returns the underlying value of a KDL null or an error if the value is
// not null.
func AsNull(v Value) (Null, error) {
	switch v := v.(type) {
	case Null:
		return v, nil
	default:
		return Null{}, fmt.Errorf("value is not null")
	}
}

type intoValue interface {
	~string |
		~int | ~int16 | ~int32 | ~int64 |
		~uint | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~bool |
		~*big.Int | ~*big.Float |
		~*String | ~*Integer | ~*Float | ~*BigInt | ~*BigFloat | ~*Boolean | ~*Null |
		String | Integer | Float | BigInt | BigFloat | Boolean | Null |
		any
}

// NewValue wraps a raw value with its corresponding KDL value type. It panics
// if the value is not a valid KDL value.
func NewValue[T intoValue](v T) Value {
	switch v := any(v).(type) {
	case string:
		return String{value: v}
	case int:
		return Integer{value: int64(v)}
	case int16:
		return Integer{value: int64(v)}
	case int32:
		return Integer{value: int64(v)}
	case int64:
		return Integer{value: v}
	case uint:
		return Integer{value: int64(v)}
	case uint16:
		return Integer{value: int64(v)}
	case uint32:
		return Integer{value: int64(v)}
	case uint64:
		return Integer{value: int64(v)}
	case float32:
		return Float{value: float64(v)}
	case float64:
		return Float{value: v}
	case bool:
		return Boolean{value: v}
	case *big.Int:
		return BigInt{value: v}
	case *big.Float:
		return BigFloat{value: v}
	case *String:
		return v
	case *Integer:
		return v
	case *Float:
		return v
	case *BigInt:
		return v
	case *BigFloat:
		return v
	case *Boolean:
		return v
	case *Null:
		return v
	case String:
		return v
	case Integer:
		return v
	case Float:
		return v
	case BigInt:
		return v
	case BigFloat:
		return v
	case Boolean:
		return v
	case Null:
		return v
	default:
		panic(fmt.Sprintf("kdl.Wrap: unsupported type %T", v))
	}
}
