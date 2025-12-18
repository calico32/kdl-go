package kdl

import (
	"fmt"
	"math/big"
	"slices"
)

type keyType interface{ ~string | ~int }

// Get gets an argument or property from a KDL node, depending on the type of
// the key (integer index for arguments, string for properties).
//
// If the key is missing, the resulting value and error will both be nil.
func Get[K keyType](node *Node, key K) (*Value, error) {
	if node == nil {
		panic("kdl.Get(): nil node")
	}

	switch key := any(key).(type) {
	case string:
		if val, ok := node.props[key]; ok {
			return &val, nil
		}
		return nil, nil
	case int:
		if key < len(node.args) {
			return &node.args[key], nil
		}
		return nil, nil
	default:
		panic(fmt.Sprintf("Get(): unsupported key type %T", key))
	}
}

// Set sets an argument or property on a KDL node, depending on the type of the
// key (integer index for arguments, string for properties).
//
// When setting an argument, if the index is greater than the current
// number of arguments, the Arguments slice will be extended to accommodate the
// new index, filling in any missing indices with KDL nulls.
// When setting a property, if the property does not already exist, it will be
// added to the PropertyOrder slice to maintain the order of properties.
func Set[K keyType](node *Node, key K, value Value) {
	if node == nil {
		panic("kdl.Set(): nil node")
	}

	switch key := any(key).(type) {
	case string:
		if !slices.Contains(node.propOrder, key) {
			node.propOrder = append(node.propOrder, key)
		}
		node.props[key] = value
	case int:
		if key < 0 {
			panic(fmt.Sprintf("Set(): negative argument index %d", key))
		}
		if key >= len(node.args) {
			// Extend the Arguments slice to accommodate the new index
			for i := len(node.args); i <= key; i++ {
				node.args = append(node.args, NewNull())
			}
		}
		node.args[key] = value
	default:
		panic(fmt.Sprintf("Set(): unsupported key type %T", key))
	}
}

// GetKV gets the first child with the given name from the KDL document and returns
// its first argument.
//
// If no such child exists, (nil, nil) is returned. If the child does not have
// exactly one argument, an error is returned.
func GetKV(document *Document, name string) (*Value, error) {
	if document == nil {
		panic("kdl.GetKV(): nil document")
	}

	for _, child := range document.Nodes {
		if child.name == name {
			if len(child.args) != 1 {
				return nil, fmt.Errorf("child %s does not have exactly one argument", name)
			}
			return &child.args[0], nil
		}
	}

	return nil, nil
}

type intoValue interface {
	~string |
		~int | ~int16 | ~int32 | ~int64 |
		~uint | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~bool |
		~*big.Int | ~*big.Float |
		~*Value
}

// NewValue wraps a raw value with its corresponding KDL value type. It panics
// if the value is not a valid KDL value.
func NewValue[T intoValue](v T) Value {
	switch v := any(v).(type) {
	case string:
		return NewString(v)
	case int:
		return NewInt(v)
	case int16:
		return NewInt(int(v))
	case int32:
		return NewInt(int(v))
	case int64:
		return NewInt(int(v))
	case uint:
		return NewInt(int(v))
	case uint16:
		return NewInt(int(v))
	case uint32:
		return NewInt(int(v))
	case uint64:
		return NewBigInt(new(big.Int).SetUint64(v))
	case float32:
		return NewFloat(float64(v))
	case float64:
		return NewFloat(v)
	case bool:
		return NewBool(v)
	case *big.Int:
		return NewBigInt(v)
	case *big.Float:
		return NewBigFloat(v)
	case *Value:
		if v == nil {
			return NewNull()
		}
		return *v
	default:
		panic(fmt.Sprintf("kdl.NewValue(): unsupported type %T", v))
	}
}
