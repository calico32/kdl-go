package kdl

import (
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var (
	// ErrStrict is the base error type for strict mode errors. It can be
	// used with [errors.Is] to check for strict mode errors.
	ErrStrict = fmt.Errorf("strict mode error")
)

// Decode reads a KDL document from r and unmarshals it into v. If v implements
// the [DocumentUnmarshaler] interface, that is used to unmarshal the document.
// Otherwise, v must be a pointer to a struct, interface, or map, and the
// document's nodes are unmarshaled into v's fields, properties, or map
// entries.
//
// # Structs
//
// For struct targets, nodes are mapped to struct fields by name or tag. If a
// struct field's type F implements the [Unmarshaler] interface, that is used
// to unmarshal the node. Otherwise, F must be a pointer to:
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
//   - arguments: indicates that the field should receive any arguments not
//     mapped to other fields. Valid only on slice, array, or map types. Can only
//     be used once per struct.
//   - argument: indicates that the field should receive a single argument by
//     position. Multiple fields can be marked with this flag to receive multiple
//     arguments. Valid only on non-slice, non-map types.
//   - properties: indicates that the field should receive any properties not
//     mapped to other fields. Valid only on map or struct types. Can only be used
//     once per struct.
//   - children: indicates that the field should receive any child nodes not
//     mapped to other fields. Valid only on slice, array, or map types. Can only
//     be used once per struct.
//
// For example:
//
//	type Config struct {
//	    Hosts []Host `kdl:"host,multiple"`
//	}
//	type Host struct {
//	    User     string `kdl:"user"`
//	    Hostname string `kdl:"hostname"`
//	    Port     int    `kdl:"port"`
//	    Properties map[string]string `kdl:",properties"`
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
// int, string to bool, etc. Use [DecodeStrict] to disable such conversions.
//
// An error is returned if the KDL value cannot be converted to the target type
// because of a type mismatch or overflow.
//
// # Unused Data
//
// Decode ignores any nodes, properties, or arguments that cannot be mapped to
// the target value. To enable strict mode, which returns an error if any data
// cannot be mapped, use [DecodeStrict].
func Decode(r io.Reader, v any) error {
	doc, err := Parse(r)
	if err != nil {
		return err
	}
	return UnmarshalDocument(doc, v)
}

// DecodeStrict is like [Decode] but enables strict mode, which returns an error
// if any nodes, properties, or arguments cannot be mapped to the target value.
// It also disables any canonical conversions when unmarshaling values.
func DecodeStrict(r io.Reader, v any) error {
	doc, err := Parse(r)
	if err != nil {
		return err
	}
	return UnmarshalDocumentStrict(doc, v)
}

// DecodeNamed is like [Decode] but allows specifying the name of the input
// source, which is used in error messages and locations.
func DecodeNamed(name string, r io.Reader, v any) error {
	doc, err := ParseNamed(name, r)
	if err != nil {
		return err
	}
	return UnmarshalDocument(doc, v)
}

// DecodeNamedStrict is like [DecodeStrict] but allows specifying the name of
// the input source, which is used in error messages and locations.
func DecodeNamedStrict(name string, r io.Reader, v any) error {
	doc, err := ParseNamed(name, r)
	if err != nil {
		return err
	}
	return UnmarshalDocumentStrict(doc, v)
}

// Unmarshal unmarshals n into v. See [Decode] for details.
func Unmarshal(n *Node, v any) error {
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalKDL(n)
	}
	d := &decoder{strict: false}
	target, err := unwrapPointerOrInterface(v)
	if err != nil {
		return err
	}

	return d.unmarshalNode(n, "", 0, target)
}

// UnmarshalStrict unmarshals n into v in strict mode, which returns an error if
// any nodes, properties, or arguments cannot be mapped to the target value. It
// also disables any canonical conversions when unmarshaling values. See
// [DecodeStrict] for details.
func UnmarshalStrict(n *Node, v any) error {
	if u, ok := v.(Unmarshaler); ok {
		return u.UnmarshalKDL(n)
	}
	d := &decoder{strict: true}
	target, err := unwrapPointerOrInterface(v)
	if err != nil {
		return err
	}

	return d.unmarshalNode(n, "", 0, target)
}

// UnmarshalDocument unmarshals the given KDL [Document] into v. See [Decode]
// for details.
func UnmarshalDocument(doc *Document, v any) error {
	return unmarshalDocument(doc, v, false)
}

// UnmarshalDocumentStrict unmarshals the given KDL [Document] into v in strict
// mode, which returns an error if any nodes, properties, or arguments cannot
// be mapped to the target value. It also disables any canonical conversions
// when unmarshaling values. See [DecodeStrict] for details.
func UnmarshalDocumentStrict(doc *Document, v any) error {
	return unmarshalDocument(doc, v, true)
}

// a decoder is a KDL decoder.
type decoder struct {
	strict bool // required
}

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
		return fmt.Errorf("argument must be a pointer to struct, interface, or map (unmarshaling document, got %s)", target.Kind())
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
				return reflect.Value{}, fmt.Errorf("interface must contain a pointer to struct, interface, or map")
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
		if err := d.unmarshalNode(node, "", 0, value); err != nil {
			return err
		}
		if target.MapIndex(key).IsValid() {
			return errors.Errorf("duplicate node %q unmarshaling into map", node.name)
		}
		target.SetMapIndex(key, value)
	}
	return nil
}

// unmarshalNodesIntoStructFields unmarshals the given KDL nodes into a Go
// struct. It attempts to map each node to a struct field by name or tag. If
// strict mode is enabled, an error is returned if any node cannot be mapped to
// a struct field. A nil pointer to the target struct will be allocated.
func (d *decoder) unmarshalNodesIntoStructFields(nodes []*Node, targetStruct reflect.Value) error {
	if targetStruct.Kind() == reflect.Pointer {
		if targetStruct.IsNil() {
			targetStruct.Set(reflect.New(targetStruct.Type().Elem()))
		}
		targetStruct = targetStruct.Elem()
	}

	structType := targetStruct.Type()
	structTags := make([]structTag, structType.NumField())
	noneFound := true

	unusedFields := make(map[int]struct{})

	for i := range structType.NumField() {
		field := structType.Field(i)
		tagStr := field.Tag.Get("kdl")
		if tagStr == "" {
			tagStr = field.Name
		}

		tag, err := parseStructTag(tagStr)
		if err != nil {
			return err
		}
		structTags[i] = tag
		unusedFields[i] = struct{}{}
	}

	for _, node := range nodes {
		index, err := d.unmarshalNodeIntoStructField(node, structTags, targetStruct)
		if err != nil {
			return err
		}

		if index != -1 {
			noneFound = false
			delete(unusedFields, index)
		} else if d.strict {
			return errors.Wrapf(ErrStrict, "no matching field found for node %q", node.name)
		}
	}

	// TODO: decide if this condition is desirable, also outside strict mode?
	if noneFound && len(nodes) > 0 && structType.NumField() > 0 {
		return errors.Errorf("%s: no matching fields found for any nodes unmarshaling into struct %s", nodes[0].loc, structType)
	}

	if len(unusedFields) > 0 && d.strict {
		var sb strings.Builder
		for i := range unusedFields {
			if sb.Len() > 0 {
				sb.WriteString(", ")
			}
			tag := structTags[i]
			sb.WriteString(fmt.Sprintf("%q", tag.name))
		}
		return errors.Wrapf(ErrStrict, "missing values for struct fields: %s", sb.String())
	}

	return nil
}

// unmarshalNodeIntoStructField attempts to unmarshal a KDL node into a field of
// the given struct by matching the node name to struct field names or tags. If
// a matching field is found, the node is unmarshaled into that field and the
// index of the field is returned. If no matching field is found, -1 is
// returned.
func (d *decoder) unmarshalNodeIntoStructField(node *Node, tags []structTag, targetStruct reflect.Value) (index int, err error) {
	nodeName := node.name
	if targetStruct.Kind() == reflect.Pointer {
		if targetStruct.IsNil() {
			targetStruct.Set(reflect.New(targetStruct.Type().Elem()))
		}
		targetStruct = targetStruct.Elem()
	}

	structType := targetStruct.Type()
	for i := range structType.NumField() {
		tag := tags[i]
		if tag.name == "" || tag.name == "-" {
			continue
		}

		if tag.name == nodeName {
			if err := d.unmarshalNode(node, tag.format, tag.flags, targetStruct.Field(i)); err != nil {
				return i, err
			}
			return i, nil
		}
	}

	return -1, nil
}

// unmarshalNodeIntoStruct unmarshals a KDL node into a Go struct. It:
//   - unmarshals properties by name into matching fields;
//   - unmarshals children by name into matching fields;
//   - and finds `arguments`, `properties`, `children`, or `argument` fields and
//     unmarshals accordingly.
//
// A nil pointer to the target struct will be allocated.
//
// A strict mode error is returned if:
//   - there are more arguments than `argument` fields and no `arguments` field is
//     present to consume the excess arguments;
//   - there are properties that do not match any struct field and no `properties`
//     field is present to consume them;
//   - or there are children that do not match any struct field and no `children` field
//     is present to consume them.
//
// If all of the following are true, an error is returned indicating that nothing was
// unmarshaled:
//   - the node has at least one argument, property, or child;
//   - no properties or children matched any struct field or were unmarshaled into;
//   - and there are no `argument`, `arguments`, `properties`, or `children`
//     fields to consume the remaining data.
func (d *decoder) unmarshalNodeIntoStruct(node *Node, targetStruct reflect.Value) error {
	if targetStruct.Kind() == reflect.Pointer {
		if targetStruct.IsNil() {
			targetStruct.Set(reflect.New(targetStruct.Type().Elem()))
		}
		targetStruct = targetStruct.Elem()
	}

	argumentsField := -1
	propertiesField := -1
	childrenField := -1
	argumentFields := make(map[int]int)

	structType := targetStruct.Type()
	structTags := make([]structTag, structType.NumField())

	for i := range structType.NumField() {
		field := structType.Field(i)
		tagStr := field.Tag.Get("kdl")
		if tagStr == "" {
			tagStr = field.Name
		}

		tag, err := parseStructTag(tagStr)
		if err != nil {
			return err
		}

		if tag.flags&arguments != 0 {
			if argumentsField != -1 {
				return errors.New("multiple arguments fields")
			}
			argumentsField = i
		}

		if tag.flags&properties != 0 {
			if propertiesField != -1 {
				return errors.New("multiple properties fields")
			}
			propertiesField = i
		}

		if tag.flags&children != 0 {
			if childrenField != -1 {
				return errors.New("multiple children fields")
			}
			childrenField = i
		}

		if tag.flags&argument != 0 {
			argumentFields[len(argumentFields)] = i
		}

		structTags[i] = tag
	}

	usedProperties := make(map[string]struct{})
	usedChildren := make(map[int]struct{})

	for argumentNum, i := range argumentFields {
		field := targetStruct.Field(i)
		format := structTags[i].format
		if argumentNum >= len(node.args) {
			return errors.Errorf("%s: expected at least %d arguments (unmarshaling node %q into struct %s)", node.loc, argumentNum+1, node.name, structType)
		}
		if err := d.unmarshalValue(node.args[argumentNum], format, field); err != nil {
			return err
		}
	}

	if len(node.args) > len(argumentFields) && argumentsField == -1 && d.strict {
		return errors.Wrapf(ErrStrict, "too many arguments (%d provided, %d expected)", len(node.args), len(argumentFields))
	}

	if argumentsField != -1 {
		field := targetStruct.Field(argumentsField)
		format := structTags[argumentsField].format
		unusedArguments := node.args[len(argumentFields):]
		if err := d.unmarshalValues(unusedArguments, format, field); err != nil {
			return err
		}
	}

	for propName, propValue := range node.props {
		found := false
		for i := 0; i < structType.NumField(); i++ {
			tag := structTags[i]
			if tag.name == propName {
				found = true
				err := d.unmarshalValue(propValue, tag.format, targetStruct.Field(i))
				if err != nil {
					return err
				}
				usedProperties[propName] = struct{}{}
			}
		}
		if !found && propertiesField == -1 && d.strict {
			return errors.Wrapf(ErrStrict, "no matching field found for property %q", propName)
		}
	}

	if propertiesField != -1 {
		field := targetStruct.Field(propertiesField)
		format := structTags[propertiesField].format
		unusedProperties := make(map[string]Value)
		for propName, propValue := range node.props {
			if _, ok := usedProperties[propName]; !ok {
				unusedProperties[propName] = propValue
			}
		}
		if err := d.unmarshalPropertiesField(node.loc, unusedProperties, format, field); err != nil {
			return err
		}
	}

	for i, node := range node.children.Nodes {
		if index, err := d.unmarshalNodeIntoStructField(node, structTags, targetStruct); err != nil {
			return err
		} else if index != -1 {
			usedChildren[i] = struct{}{}
		} else if childrenField == -1 && d.strict {
			return errors.Wrapf(ErrStrict, "no matching field found for node %q", node.name)
		}
	}

	if childrenField != -1 {
		field := targetStruct.Field(childrenField)
		unusedChildren := make([]*Node, 0, len(node.children.Nodes)-len(usedChildren))
		for i, child := range node.children.Nodes {
			if _, ok := usedChildren[i]; !ok {
				unusedChildren = append(unusedChildren, child)
			}
		}
		return d.unmarshalChildrenField(unusedChildren, field)
	}

	// if we didn't unmarshal anything, it's an error
	if (len(node.children.Nodes) > 0 ||
		len(node.props) > 0 ||
		len(node.args) > 0) &&
		len(usedProperties) == 0 &&
		len(usedChildren) == 0 &&
		len(argumentFields) == 0 && argumentsField == -1 && propertiesField == -1 && childrenField == -1 {
		return fmt.Errorf("don't know what to do with node %q unmarshaling into struct %s", node.name, structType)
	}

	return nil
}

// unmarshalValues unmarshals a slice of KDL values into a Go slice, array, or
// map. Behaves as expected for the target type; map keys must be of string,
// integer, or unsigned integer types.
func (d *decoder) unmarshalValues(arguments []Value, format string, target reflect.Value) error {
	if target.Kind() == reflect.Pointer {
		elem := target.Type().Elem()
		if elem.Kind() != reflect.Slice && elem.Kind() != reflect.Array && elem.Kind() != reflect.Map {
			return errors.New("arguments field must point to slice, array, or map")
		}
		if target.IsNil() {
			target.Set(reflect.New(elem))
		}
	}

	// type must be slice, array, or map (or pointer to one of those)
	switch target.Kind() {
	case reflect.Array:
		if len(arguments) != target.Len() {
			return errors.Errorf("expected exactly %d arguments", target.Len())
		}
		for i, arg := range arguments {
			if err := d.unmarshalValue(arg, format, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice:
		if target.IsNil() {
			target.Set(reflect.MakeSlice(target.Type(), len(arguments), len(arguments)))
		}
		for i, arg := range arguments {
			if err := d.unmarshalValue(arg, format, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}

		hasStringKeys := target.Type().Key().ConvertibleTo(reflect.TypeFor[string]())
		hasIntKeys := isInt(target.Type().Key())
		hasUintKeys := isUint(target.Type().Key())
		if !hasStringKeys && !hasIntKeys && !hasUintKeys {
			return errors.New("map key type must be string, integer, or unsigned integer")
		}
		for i, arg := range arguments {
			var key reflect.Value
			switch {
			case hasIntKeys:
				if target.Type().Key().OverflowInt(int64(i)) {
					return errors.Errorf("argument index %d overflows map key type %s", i, target.Type().Key())
				}
				key = reflect.ValueOf(i).Convert(target.Type().Key())
			case hasUintKeys:
				if target.Type().Key().OverflowUint(uint64(i)) {
					return errors.Errorf("argument index %d overflows map key type %s", i, target.Type().Key())
				}
				key = reflect.ValueOf(uint(i)).Convert(target.Type().Key())
			case hasStringKeys:
				key = reflect.ValueOf(fmt.Sprint(i)).Convert(target.Type().Key())
			}
			value := reflect.New(target.Type().Elem()).Elem()
			if err := d.unmarshalValue(arg, format, value); err != nil {
				return err
			}
			target.SetMapIndex(key, value)
		}
		return nil

	default:
		return errors.New("arguments field must be slice, array, or map")
	}
}

// unmarshalPropertiesField unmarshals a map of KDL properties into a Go map
// or struct. Behaves as expected for maps, and for structs, matches property
// names to struct field names or tags, returning a strict mode error if any
// property cannot be matched.
func (d *decoder) unmarshalPropertiesField(location Location, properties map[string]Value, format string, target reflect.Value) error {
	if target.Kind() == reflect.Pointer {
		elem := target.Type().Elem()
		if elem.Kind() != reflect.Map && elem.Kind() != reflect.Struct {
			return errors.New("properties field must point to map or struct")
		}
		if target.IsNil() {
			target.Set(reflect.New(elem))
		}
	}

	switch target.Kind() {
	case reflect.Map:
		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}
		hasStringKeys := target.Type().Key().ConvertibleTo(reflect.TypeFor[string]())
		if !hasStringKeys {
			return errors.New("map key type must be string when unmarshaling properties")
		}
		for propName, propValue := range properties {
			key := reflect.ValueOf(propName).Convert(target.Type().Key())
			value := reflect.New(target.Type().Elem()).Elem()
			if err := d.unmarshalValue(propValue, format, value); err != nil {
				return err
			}
			if target.MapIndex(key).IsValid() {
				return errors.Errorf("%s: duplicate property %q unmarshaling into map", location, propName)
			}
			target.SetMapIndex(key, value)
		}
		return nil

	case reflect.Struct:
		for propName, propValue := range properties {
			fieldFound := false
			ty := target.Type()
			for i := 0; i < ty.NumField(); i++ {
				field := ty.Field(i)
				tagStr := field.Tag.Get("kdl")
				tag, err := parseStructTag(tagStr)
				if err != nil {
					return err
				}
				if tag.name == propName {
					fieldFound = true
					if err := d.unmarshalValue(propValue, tag.format, target.Field(i)); err != nil {
						return err
					}
					break
				}
			}
			if !fieldFound && d.strict {
				return errors.Wrapf(ErrStrict, "no matching field found for property %q", propName)
			}
		}
		return nil

	default:
		return errors.New("properties field must be map or struct")
	}
}

// unmarshalChildrenField unmarshals a slice of KDL nodes into a Go map
// or struct. Behaves as expected for maps using node names as keys. For structs,
// matches node names to struct field names or tags, returning a strict mode
// error if any node cannot be matched.
func (d *decoder) unmarshalChildrenField(children []*Node, target reflect.Value) error {
	if target.Kind() == reflect.Pointer {
		elem := target.Type().Elem()
		if elem.Kind() != reflect.Map && elem.Kind() != reflect.Struct {
			return errors.New("properties field must point to map or struct")
		}
		if target.IsNil() {
			target.Set(reflect.New(elem))
		}
	}

	switch target.Kind() {
	case reflect.Map:
		if target.IsNil() {
			target.Set(reflect.MakeMap(target.Type()))
		}
		hasStringKeys := target.Type().Key().ConvertibleTo(reflect.TypeFor[string]())
		if !hasStringKeys {
			return errors.New("map key type must be string when unmarshaling properties")
		}
		for _, child := range children {
			key := reflect.ValueOf(child.name).Convert(target.Type().Key())
			value := reflect.New(target.Type().Elem()).Elem()
			if err := d.unmarshalNode(child, "", 0, value); err != nil {
				return err
			}
			if target.MapIndex(key).IsValid() {
				return errors.Errorf("%s: duplicate child %q unmarshaling into map", child.loc, child.name)
			}
			target.SetMapIndex(key, value)
		}
		return nil

	case reflect.Struct:
		// ignore?
		panic("unimplemented: unmarshaling children into struct")

	default:
		return errors.New("properties field must be map or struct")
	}
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
func (d *decoder) unmarshalNode(node *Node, format string, flags tagFlags, target reflect.Value) error {
	// TODO: is this or the one further down after the pointer unwrap necessary?
	if target.Type().AssignableTo(reflect.TypeFor[Unmarshaler]()) {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		return target.Interface().(Unmarshaler).UnmarshalKDL(node)
	}

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
			return fmt.Errorf("expected exactly one argument (unmarshaling node %q into %s)", node.name, target.Type())
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
		return d.unmarshalTime(node.args[0], format, target)
	}

	switch target.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64,
		reflect.Bool:
		if len(node.args) != 1 {
			return fmt.Errorf("expected exactly one argument (unmarshaling node %q into %s)", node.name, target.Kind())
		}
		return d.unmarshalValue(node.args[0], format, target)

	case reflect.Struct:
		return d.unmarshalNodeIntoStruct(node, target)

	case reflect.Slice:
		if flags&multiple != 0 {
			if target.IsNil() {
				target.Set(reflect.MakeSlice(target.Type(), 0, 2))
			}

			elem := reflect.New(target.Type().Elem())
			if err := d.unmarshalNode(node, format, flags, elem.Elem()); err != nil {
				return err
			}
			target.Set(reflect.Append(target, elem.Elem()))
			return nil
		} else {
			if target.IsNil() {
				target.Set(reflect.MakeSlice(target.Type(), len(node.args), len(node.args)))
			}
			for i, arg := range node.args {
				if err := d.unmarshalValue(arg, format, target.Index(i)); err != nil {
					return err
				}
			}
		}
		return nil

	case reflect.Array:
		if flags&multiple != 0 {
			panic("unimplemented: multiple flag on array type")
		}
		if len(node.args) != target.Len() {
			return errors.Errorf("expected exactly %d arguments", target.Len())
		}
		for i, arg := range node.args {
			if err := d.unmarshalValue(arg, format, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Interface:
		if len(node.props) == 0 && len(node.children.Nodes) == 0 {
			if len(node.args) != 1 {
				return fmt.Errorf("expected exactly one argument (unmarshaling node %q into interface)", node.name)
			}
			return d.unmarshalValueIntoInterface(node.args[0], target)
		}

		if target.NumMethod() != 0 {
			return fmt.Errorf("cannot unmarshal node %q into non-empty interface", node.name)
		}

		m := reflect.ValueOf(map[string]any{})
		target.Set(m)
		target = m
		fallthrough
	case reflect.Map:
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
			err := d.unmarshalValues(node.args, format, target)
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
				if err := d.unmarshalValue(node.props[prop], format, value); err != nil {
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
				if err := d.unmarshalNode(child, "", 0, value); err != nil {
					return err
				}
				target.SetMapIndex(key, value)
			}
		}
		return nil
	default:
		return errors.Errorf("cannot unmarshal node %q into %s", node.name, target.Type())
	}
}

// unmarshalValue unmarshals a KDL value into a single Go value. See [Unmarshal]
// for details on supported value types. It handles special cases like
// [time.Time], [time.Duration], and [ValueUnmarshaler], using an optional
// format string for guidance.
func (d *decoder) unmarshalValue(value Value, format string, target reflect.Value) error {
	if target.Type().NumMethod() > 0 && target.CanInterface() {
		if u, ok := target.Interface().(ValueUnmarshaler); ok {
			return u.UnmarshalKDL(value)
		}
	}

	switch target.Type() {
	case timeType:
		return d.unmarshalTime(value, format, target)
	case durationType:
		return d.unmarshalDuration(value, format, target)
	}

	switch target.Kind() {
	case reflect.String:
		return d.unmarshalString(value, target)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return d.unmarshalInt(value, target)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return d.unmarshalUint(value, target)
	case reflect.Float32, reflect.Float64:
		return d.unmarshalFloat(value, target)
	case reflect.Bool:
		return d.unmarshalBool(value, target)
	case reflect.Pointer:
		elem := target.Type().Elem()
		switch elem {
		case bigIntType:
			return d.unmarshalBigInt(value, target)
		case bigFloatType:
			return d.unmarshalBigFloat(value, target)
		default:
			if target.IsNil() {
				target.Set(reflect.New(elem))
			}
			return d.unmarshalValue(value, format, target.Elem())
		}
	case reflect.Interface:
		return d.unmarshalValueIntoInterface(value, target)
	}

	return nil
}

// tagFlags represents flags parsed from a struct field's KDL tag.
type tagFlags uint8

// KDL struct tag flags.
const (
	omitempty tagFlags = 1 << iota
	multiple
	arguments
	properties
	argument
	children
)

// structTag represents a parsed KDL struct tag.
type structTag struct {
	name   string
	flags  tagFlags
	format string
}

// parseStructTag parses a KDL struct tag string into a structTag. str should be
// the content inside of the tag, not including the `kdl:" or "` part. It
// returns an error if the tag is empty, contains unknown flags, or contains
// mutually exclusive flags (argument and any of arguments, properties,
// children).
func parseStructTag(str string) (t structTag, err error) {
	parts := strings.Split(str, ",")
	if len(parts) == 0 {
		err = errors.New("empty tag")
		return
	}

	t.name = parts[0]
	for _, part := range parts[1:] {
		switch part {
		case "omitempty":
			t.flags |= omitempty
		case "multiple":
			t.flags |= multiple
		case "arguments":
			t.flags |= arguments
		case "properties":
			t.flags |= properties
		case "argument":
			t.flags |= argument
		case "children":
			t.flags |= children
		default:
			if strings.HasPrefix(part, "format:") {
				t.format = part[7:]
			} else {
				err = errors.Errorf("unknown tag flag %q", part)
				return
			}
		}
	}

	// mutually exclusive flags: argument and any of properties, argument, children
	mutuallyExclusive := 0
	if t.flags&arguments != 0 {
		mutuallyExclusive++
	}
	if t.flags&properties != 0 {
		mutuallyExclusive++
	}
	if t.flags&children != 0 {
		mutuallyExclusive++
	}
	if t.flags&argument != 0 && mutuallyExclusive > 1 {
		err = errors.New("more than one of argument and (arguments, properties, and children) specified in same tag")
	}

	return
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
