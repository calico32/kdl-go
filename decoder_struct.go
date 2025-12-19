package kdl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

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
	strictFields := make(map[int]struct{})

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

		if tag.flags&strict != 0 {
			strictFields[i] = struct{}{}
		}
	}

	for _, node := range nodes {
		index, err := d.unmarshalNodeIntoStructField(node, structTags, targetStruct)
		if err != nil {
			return err
		}

		if index != -1 {
			noneFound = false
			delete(unusedFields, index)
			delete(strictFields, index)
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

	if len(strictFields) > 0 {
		var sb strings.Builder
		for i := range strictFields {
			if sb.Len() > 0 {
				sb.WriteString(", ")
			}
			tag := structTags[i]
			sb.WriteString(fmt.Sprintf("%q", tag.name))
		}
		return errors.Wrapf(ErrStrict, "missing values for strict struct fields: %s", sb.String())
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

		if tag.name == nodeName && tag.flags&property == 0 {
			if err := d.unmarshalNode(node, tag, targetStruct.Field(i)); err != nil {
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
//   - there are children that do not match any struct field and no `children` field
//     is present to consume them;
//   - or any field marked `strict` is not unmarshaled into.
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
	strictFields := make(map[int]struct{})

	structType := targetStruct.Type()
	structTags := make([]structTag, structType.NumField())

	for i := range structType.NumField() {
		field := structType.Field(i)
		tagStr, ok := field.Tag.Lookup("kdl")
		if !ok {
			tagStr = field.Name
		}

		tag, err := parseStructTag(tagStr)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("parsing struct tag for field %s.%s: %s", structType, field.Name, err))
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

		if tag.flags&strict != 0 {
			strictFields[i] = struct{}{}
		}

		structTags[i] = tag
	}

	usedProperties := make(map[string]struct{})
	usedChildren := make(map[int]struct{})

	for argumentNum, i := range argumentFields {
		field := targetStruct.Field(i)
		if argumentNum >= len(node.args) && (d.strict || structTags[i].flags&strict != 0) {
			return errors.Errorf("%s: expected at least %d arguments (unmarshaling node %q into struct %s)", node.loc, argumentNum+1, node.name, structType)
		}
		if err := d.unmarshalValue(node.args[argumentNum], structTags[i], field); err != nil {
			return err
		}
		delete(strictFields, i)
	}

	if len(node.args) > len(argumentFields) && argumentsField == -1 && d.strict {
		return errors.Wrapf(ErrStrict, "too many arguments (%d provided, %d expected)", len(node.args), len(argumentFields))
	}

	if argumentsField != -1 {
		field := targetStruct.Field(argumentsField)
		unusedArguments := node.args[len(argumentFields):]
		if err := d.unmarshalValues(unusedArguments, structTags[argumentsField], field); err != nil {
			return err
		}
		delete(strictFields, argumentsField)
	}

	for propName, propValue := range node.props {
		found := false
		for i := 0; i < structType.NumField(); i++ {
			tag := structTags[i]
			if tag.name == propName && tag.flags&child == 0 {
				found = true
				err := d.unmarshalValue(propValue, tag, targetStruct.Field(i))
				if err != nil {
					return err
				}
				usedProperties[propName] = struct{}{}
				delete(strictFields, i)
			}
		}
		if !found && propertiesField == -1 && d.strict {
			return errors.Wrapf(ErrStrict, "no matching field found for property %q", propName)
		}
	}

	if propertiesField != -1 {
		field := targetStruct.Field(propertiesField)
		unusedProperties := make(map[string]Value)
		for propName, propValue := range node.props {
			if _, ok := usedProperties[propName]; !ok {
				unusedProperties[propName] = propValue
			}
		}
		if err := d.unmarshalPropertiesField(node.loc, unusedProperties, structTags[propertiesField], field); err != nil {
			return err
		}
		delete(strictFields, propertiesField)
	}

	for i, node := range node.children.Nodes {
		if index, err := d.unmarshalNodeIntoStructField(node, structTags, targetStruct); err != nil {
			return err
		} else if index != -1 {
			usedChildren[i] = struct{}{}
			delete(strictFields, index)
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
		if err := d.unmarshalChildrenField(unusedChildren, structTags[childrenField], field); err != nil {
			return err
		}
		delete(strictFields, childrenField)
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

	// report any missing strict fields
	if len(strictFields) > 0 {
		var sb strings.Builder
		for i := range strictFields {
			if sb.Len() > 0 {
				sb.WriteString(", ")
			}
			tag := structTags[i]
			sb.WriteString(fmt.Sprintf("%q", tag.name))
		}
		return errors.Wrapf(ErrStrict, "missing values for strict struct fields: %s", sb.String())
	}

	return nil
}

// unmarshalValues unmarshals a slice of KDL values into a Go slice, array, or
// map. Behaves as expected for the target type; map keys must be of string,
// integer, or unsigned integer types.
func (d *decoder) unmarshalValues(arguments []Value, tag structTag, target reflect.Value) error {
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
			if err := d.unmarshalValue(arg, tag, target.Index(i)); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice:
		if target.IsNil() {
			target.Set(reflect.MakeSlice(target.Type(), len(arguments), len(arguments)))
		}
		for i, arg := range arguments {
			if err := d.unmarshalValue(arg, tag, target.Index(i)); err != nil {
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
			if err := d.unmarshalValue(arg, tag, value); err != nil {
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
func (d *decoder) unmarshalPropertiesField(location Location, properties map[string]Value, tag structTag, target reflect.Value) error {
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
			if err := d.unmarshalValue(propValue, tag, value); err != nil {
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
					if err := d.unmarshalValue(propValue, tag, target.Field(i)); err != nil {
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
func (d *decoder) unmarshalChildrenField(children []*Node, tag structTag, target reflect.Value) error {
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
			if err := d.unmarshalNode(child, tag, value); err != nil {
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
		return errors.Errorf("properties field must be map or struct (got %s)", target.Type())
	}
}
