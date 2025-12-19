package kdl

import (
	"math/big"
	"reflect"

	"time"

	"github.com/pkg/errors"
)

// a decoder is a KDL decoder.
type decoder struct {
	strict bool // required
}

// unmarshalDocument unmarshals a KDL document into the given Go value v.
func unmarshalDocument(doc *Document, v any, strict bool) error {
	if u, ok := v.(DocumentUnmarshaler); ok {
		return u.UnmarshalKDL(doc)
	}

	target, err := unwrapPointerOrInterface(v)
	if err != nil {
		return err
	}

	d := &decoder{strict: strict}
	switch target.Kind() {
	case reflect.Map:
		return d.unmarshalNodesIntoMap(doc.Nodes, target)
	case reflect.Struct:
		return d.unmarshalNodesIntoStructFields(doc.Nodes, target)
	default:
		return errors.Errorf("argument must be a pointer to struct, interface, or map (unmarshaling document, got %s)", target.Kind())
	}
}

// unwrapPointerOrInterface acts as an entrypoint for the decoder, converting an
// any into a reflect.Value representing the target to unmarshal into.
//   - If v is a reflect.Value, it is returned directly with no changes. The caller
//     is responsible for ensuring it is a pointer to struct, interface, or map.
//   - If v is pointer to any, a reflect.Value of map[string]any is returned.
//   - If v is pointer to a non-empty interface and its concrete value is a pointer,
//     a reflect.Value of the pointer's element is returned. For all other concrete
//     types, an error is returned.
//   - If v is otherwise a pointer to a different type, a reflect.Value of the
//     pointer's element is returned. It is the caller’s responsibility to ensure
//     it can be unmarshaled into.
//   - If v is not a pointer, an error is returned.
func unwrapPointerOrInterface(v any) (reflect.Value, error) {
	if v, ok := v.(reflect.Value); ok {
		return v, nil
	}

	target := reflect.ValueOf(v)

	if target.Kind() != reflect.Pointer {
		return reflect.Value{}, errors.New("argument must be a pointer to struct, interface, or map")
	}

	target = target.Elem()

	if target.Kind() == reflect.Interface {
		if target.NumMethod() == 0 {
			m := reflect.ValueOf(map[string]any{})
			target.Set(m)
			target = m
		} else {
			target = target.Elem()
			if target.Kind() != reflect.Pointer {
				return reflect.Value{}, errors.Errorf("interface must contain a pointer to struct, interface, or map")
			}
			target = target.Elem()
		}
	}
	return target, nil
}

// unmarshalNodesIntoMap unmarshals the given KDL nodes into a Go map. It treats
// the nodes' names as map keys and unmarshals each node into a value of the map
// value type.
//   - The target map must have string keys.
//   - Duplicate node names result in an error.
//
// This method cannot return a strict mode error because nodes are always added
// to the map.
func (d *decoder) unmarshalNodesIntoMap(nodes []*Node, target reflect.Value) error {
	if target.IsNil() {
		target.Set(reflect.MakeMap(target.Type()))
	}
	hasStringKeys := target.Type().Key().ConvertibleTo(reflect.TypeFor[string]())
	if !hasStringKeys {
		return errors.New("map key type must be string when unmarshaling properties")
	}
	for _, node := range nodes {
		key := reflect.ValueOf(node.name).Convert(target.Type().Key())
		value := reflect.New(target.Type().Elem()).Elem()
		if err := d.unmarshalNode(node, structTag{}, value); err != nil {
			return err
		}
		if target.MapIndex(key).IsValid() {
			return errors.Errorf("duplicate node %q unmarshaling into map", node.name)
		}
		target.SetMapIndex(key, value)
	}
	return nil
}

var (
	bigIntType   = reflect.TypeFor[big.Int]()
	bigFloatType = reflect.TypeFor[big.Float]()
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
)

// unmarshalNode unmarshals a KDL node into a single value. See [Unmarshal] for
// details on supported target types. It handles special cases like time.Time and
// time.Duration, using an optional format string and flags for guidance. It
// delegates [decoder.unmarshalNodeIntoStruct] for struct targets.
func (d *decoder) unmarshalNode(node *Node, tag structTag, target reflect.Value) error {
	for target.Kind() == reflect.Pointer {
		elemType := target.Type().Elem()
		if target.IsNil() {
			target.Set(reflect.New(elemType))
		}
		target = target.Elem()
	}

	if reflect.PointerTo(target.Type()).AssignableTo(reflect.TypeFor[Unmarshaler]()) {
		u := reflect.New(target.Type())
		err := u.Interface().(Unmarshaler).UnmarshalKDL(node)
		if err != nil {
			return err
		}
		target.Set(u.Elem())
		return nil
	}

	if reflect.PointerTo(target.Type()).AssignableTo(reflect.TypeFor[ValueUnmarshaler]()) {
		if len(node.args) != 1 {
			return errors.Errorf("expected exactly one argument (unmarshaling node %q into %s)", node.name, target.Type())
		}
		u := reflect.New(target.Type())
		err := u.Interface().(ValueUnmarshaler).UnmarshalKDL(node.args[0])
		if err != nil {
			return err
		}
		target.Set(u.Elem())
		return nil
	}

	// special handling for time.Time and time.Duration
	if target.Type() == timeType || target.Type() == durationType {
		if len(node.args) != 1 {
			return errors.Errorf("expected exactly one argument (unmarshaling node %q into %s)", node.name, target.Type())
		}
		return d.unmarshalTime(node.args[0], tag, target)
	}

	switch target.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Bool:
		if len(node.args) != 1 {
			return errors.Errorf("expected exactly one argument (unmarshaling node %q into %s)", node.name, target.Kind())
		}
		return d.unmarshalValue(node.args[0], tag, target)

	case reflect.Struct:
		return d.unmarshalNodeIntoStruct(node, target)

	case reflect.Slice:
		if tag.flags&multiple != 0 {
			if target.IsNil() {
				target.Set(reflect.MakeSlice(target.Type(), 0, 2))
			}

			elem := reflect.New(target.Type().Elem())
			if err := d.unmarshalNode(node, tag, elem.Elem()); err != nil {
				return err
			}
			target.Set(reflect.Append(target, elem.Elem()))
			return nil
		} else {
			if target.IsNil() {
				target.Set(reflect.MakeSlice(target.Type(), len(node.args), len(node.args)))
			}
			for i, arg := range node.args {
				if err := d.unmarshalValue(arg, tag, target.Index(i)); err != nil {
					return err
				}
			}
		}
		return nil

	case reflect.Array:
		if tag.flags&multiple != 0 {
			panic("unimplemented: multiple flag on array type")
		}
		if len(node.args) != target.Len() {
			return errors.Errorf("expected exactly %d arguments", target.Len())
		}
		for i, arg := range node.args {
			if err := d.unmarshalValue(arg, tag, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Interface:
		if len(node.props) == 0 && len(node.children.Nodes) == 0 {
			if len(node.args) != 1 {
				return errors.Errorf("expected exactly one argument (unmarshaling node %q into interface)", node.name)
			}
			return d.unmarshalValueIntoInterface(node.args[0], target)
		}

		if target.NumMethod() != 0 {
			return errors.Errorf("cannot unmarshal node %q into non-empty interface", node.name)
		}

		m := reflect.ValueOf(map[string]any{})
		target.Set(m)
		target = m
		fallthrough
	case reflect.Map:
		return d.unmarshalNodeIntoMap(node, tag, target)
	default:
		return errors.Errorf("cannot unmarshal node %q into %s", node.name, target.Type())
	}
}

// unmarshalNodeIntoMap unmarshals a KDL node into a Go map. It treats the node's
// arguments, properties, and children as map entries.
//   - The target map must have string/integer/unsigned integer keys.
//   - Arguments are added first, followed by properties, then children.
//   - For properties and children, the map key is the property/child name. For
//     arguments, the map key is the argument index,
//   - The map value type must be a supported value type (see [Decode]).
func (d *decoder) unmarshalNodeIntoMap(node *Node, tag structTag, target reflect.Value) error {
	if target.IsNil() {
		target.Set(reflect.MakeMap(target.Type()))
	}

	hasStringKeys := target.Type().Key().ConvertibleTo(reflect.TypeFor[string]())
	hasIntKeys := isInt(target.Type().Key())
	hasUintKeys := isUint(target.Type().Key())
	if !hasStringKeys && !hasIntKeys && !hasUintKeys {
		return errors.New("map key type must be string, integer, or unsigned integer")
	}

	hasArguments := len(node.args) > 0
	hasProperties := len(node.propOrder) > 0
	hasChildren := len(node.children.Nodes) > 0

	if hasArguments {
		err := d.unmarshalValues(node.args, tag, target)
		if err != nil {
			return err
		}
	}

	if hasProperties {
		if !hasStringKeys {
			return errors.New("map key type must be string when unmarshaling properties")
		}
		for _, prop := range node.propOrder {
			key := reflect.ValueOf(prop).Convert(target.Type().Key())
			value := reflect.New(target.Type().Elem()).Elem()
			if err := d.unmarshalValue(node.props[prop], tag, value); err != nil {
				return err
			}
			target.SetMapIndex(key, value)
		}
	}

	if hasChildren {
		if !hasStringKeys {
			return errors.New("map key type must be string when unmarshaling children")
		}

		for _, child := range node.children.Nodes {
			key := reflect.ValueOf(child.name).Convert(target.Type().Key())
			value := reflect.New(target.Type().Elem()).Elem()
			if err := d.unmarshalNode(child, tag, value); err != nil {
				return err
			}
			target.SetMapIndex(key, value)
		}
	}
	return nil
}

// isInt returns true if t is any integer type.
func isInt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

// isUint returns true if t is any unsigned integer type.
func isUint(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}
