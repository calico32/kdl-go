package kdl

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// Encode marshals the given value into a KDL Document and writes its KDL
func Encode(v any, w io.Writer, opts ...EmitterOption) error {
	doc, err := Marshal(v)
	if err != nil {
		return err
	}

	err = Emit(doc, w, opts...)
	if err != nil {
		return err
	}
	return nil
}

type MarshalOption func(*encoder)

func Marshal(v any, opts ...MarshalOption) (*Document, error) {
	var target reflect.Value
	if rv, ok := v.(reflect.Value); ok {
		target = rv
	} else {
		target = reflect.ValueOf(v)
		if !target.IsValid() {
			return nil, errors.Errorf("cannot marshal nil value")
		}
		for target.Kind() == reflect.Pointer {
			target = target.Elem()
			if !target.IsValid() {
				return nil, errors.Errorf("cannot marshal nil pointer")
			}
		}

		if target.Kind() == reflect.Interface {
			target = target.Elem()
			if !target.IsValid() {
				return nil, errors.Errorf("cannot marshal nil interface")
			}
			for target.Kind() == reflect.Pointer {
				target = target.Elem()
				if !target.IsValid() {
					return nil, errors.Errorf("cannot marshal nil pointer inside interface")
				}
			}
		}
	}

	doc := &Document{}
	e := &encoder{
		stack: []*Document{doc},
	}
	for _, opt := range opts {
		opt(e)
	}
	defer un(e.trace("Marshal %s", target.Type()))

	if target.Type().Implements(reflect.TypeFor[DocumentMarshaler]()) {
		dm := target.Interface().(DocumentMarshaler)
		marshaledDoc, err := dm.MarshalKDLDocument()
		if err != nil {
			return nil, err
		}
		return marshaledDoc, nil
	}

	if reflect.PointerTo(target.Type()).Implements(reflect.TypeFor[DocumentMarshaler]()) {
		ptr := reflect.New(target.Type())
		ptr.Elem().Set(target)
		dm := ptr.Interface().(DocumentMarshaler)
		marshaledDoc, err := dm.MarshalKDLDocument()
		if err != nil {
			return nil, err
		}
		return marshaledDoc, nil
	}

	switch target.Kind() {
	case reflect.Struct:
		err := e.encodeStructFieldsAsNodes(target)
		if err != nil {
			return nil, err
		}
	case reflect.Map:
		err := e.encodeMapEntriesAsNodes(target)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("argument must be struct or map (marshaling document, got %s)", target.Kind())
	}

	return doc, nil
}

func WithTrace(w io.Writer) MarshalOption {
	return func(e *encoder) {
		e.traceWriter = w
	}
}

type encoder struct {
	stack       []*Document // required
	traceWriter io.Writer
	indent      int
}

func (e *encoder) tracef(format string, args ...any) {
	if e.traceWriter != nil {
		fmt.Fprintf(e.traceWriter, "%s", strings.Repeat("  ", e.indent))
		fmt.Fprintf(e.traceWriter, format, args...)
	}
}

func (e *encoder) trace(format string, args ...any) func() {
	e.tracef(format+"\n", args...)
	e.indent++
	return func() {
		e.indent--
	}
}

func un(f func()) {
	f()
}

func (e *encoder) currentContext() *Document {
	return e.stack[len(e.stack)-1]
}

func (e *encoder) pushContext(doc *Document) {
	e.stack = append(e.stack, doc)
}

func (e *encoder) popContext() {
	e.stack = e.stack[:len(e.stack)-1]
}

func (e *encoder) encodeStructFieldsAsNodes(target reflect.Value) error {
	defer un(e.trace("encodeStructFieldsAsNodes %s", target.Type()))
	ctx, err := newStructContext(target.Type())
	if err != nil {
		return err
	}

	for i := range target.NumField() {
		tag := ctx.tags[i]
		if tag.name == "" || tag.name == "-" {
			continue
		}
		field := target.Field(i)
		if !field.CanInterface() {
			panic(fmt.Sprintf("kdl.encoder: unexported field %s.%s, should be unreachable",
				target.Type(), target.Type().Field(i).Name))
		}

		if tag.flags&omitzero != 0 && field.IsZero() {
			continue
		}

		err := e.encodeValueAsNode(tag.name, tag, field)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) encodeStructAsNode(name string, target reflect.Value) error {
	defer un(e.trace("encodeStructAsNode %s", target.Type()))
	ctx, err := newStructContext(target.Type())
	if err != nil {
		return err
	}

	node := NewNode(name)

	for fieldIndex := range target.Type().NumField() {
		tag := ctx.tags[fieldIndex]
		if tag.name == "-" {
			continue
		}
		field := target.Field(fieldIndex)
		if !field.CanInterface() {
			panic(fmt.Sprintf("kdl.encoder: unexported field %s.%s, should be unreachable",
				target.Type(), target.Type().Field(fieldIndex).Name))
		}

		if tag.flags&omitzero != 0 && field.IsZero() {
			continue
		}

		if tag.flags&argument != 0 {
			e.tracef("argument: %s %s\n", tag.name, field.Type())
			value, err := TryNewValue(field.Interface())
			if err != nil {
				return err
			}
			node.AddArgument(value)
			continue
		}

		if tag.flags&arguments != 0 {
			for i := 0; i < field.Len(); i++ {
				e.tracef("arguments: %s[%d] %s\n", tag.name, i, field.Index(i).Type())
				if tag.flags&omitzero != 0 && field.Index(i).IsZero() {
					continue
				}
				value, err := TryNewValue(field.Index(i).Interface())
				if err != nil {
					return err
				}
				node.AddArgument(value)
			}
			continue
		}

		if tag.flags&property != 0 {
			e.tracef("property: %s %s\n", tag.name, field.Type())
			value, err := TryNewValue(field.Interface())
			if err != nil {
				return err
			}
			node.AddProperty(tag.name, value)
			continue
		}

		if tag.flags&properties != 0 {
			e.tracef("properties: %s %s\n", tag.name, field.Type())
			switch field.Kind() {
			case reflect.Map:
				for _, key := range field.MapKeys() {
					val := field.MapIndex(key)
					if tag.flags&omitzero != 0 && val.IsZero() {
						continue
					}
					value, err := TryNewValue(val.Interface())
					if err != nil {
						return err
					}
					node.AddProperty(fmt.Sprint(key.Interface()), value)
				}
			case reflect.Struct:
				err := e.encodeStructIntoProperties(node, field)
				if err != nil {
					return err
				}
			default:
				return errors.Errorf("unsupported properties field kind %s", field.Kind())
			}
			continue
		}

		if tag.flags&children != 0 {
			e.tracef("children: %s %s\n", tag.name, field.Type())
			switch field.Kind() {
			case reflect.Map:
				e.pushContext(node.Children())
				err := e.encodeMapEntriesAsNodes(field)
				if err != nil {
					return err
				}
				e.popContext()
				continue
			case reflect.Struct:
				e.pushContext(node.Children())
				err := e.encodeStructFieldsAsNodes(field)
				if err != nil {
					return err
				}
				e.popContext()
				continue
			default:
				return errors.Errorf("unsupported children field kind %s", field.Kind())
			}
		}

		// no special flags, treat as child node
		e.pushContext(node.Children())
		err := e.encodeValueAsNode(tag.name, tag, field)
		if err != nil {
			return err
		}
		e.popContext()
	}

	e.currentContext().AddNode(node)
	return nil
}

func (e *encoder) encodeMapAsNode(name string, target reflect.Value) error {
	defer un(e.trace("encodeMapAsNode %s", target.Type()))
	node := NewNode(name)
	e.pushContext(node.Children())
	err := e.encodeMapEntriesAsNodes(target)
	if err != nil {
		return err
	}
	e.popContext()
	e.currentContext().AddNode(node)
	return nil
}

type mapKey struct {
	string
	reflect.Value
}

func (e *encoder) encodeMapEntriesAsNodes(target reflect.Value) error {
	defer un(e.trace("encodeMapEntriesAsNodes %s", target.Type()))
	keys := make([]mapKey, 0, target.Len())
	for _, key := range target.MapKeys() {
		keys = append(keys, mapKey{
			fmt.Sprint(key.Interface()),
			key,
		})
	}
	slices.SortStableFunc(keys, func(a, b mapKey) int {
		return strings.Compare(a.string, b.string)
	})

	for _, key := range keys {
		val := target.MapIndex(key.Value)
		err := e.encodeValueAsNode(key.string, structTag{}, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodeValueAsNode(name string, tag structTag, target reflect.Value) error {
	defer un(e.trace("encodeValueAsNode %s %s", name, target.Type()))

	if encoded, err := e.tryEncodeCustomMarshalerAsNode(name, target); encoded || err != nil {
		return err
	}

	for target.Kind() == reflect.Pointer {
		if encoded, err := e.tryEncodeCustomMarshalerAsNode(name, target); encoded || err != nil {
			return err
		}
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		value, err := TryNewValue(target.Interface())
		if err != nil {
			return err
		}
		childNode := NewKValue(name, value)
		e.currentContext().AddNode(childNode)

	case reflect.Struct:
		err := e.encodeStructAsNode(name, target)
		if err != nil {
			return err
		}

	case reflect.Slice, reflect.Array:
		if tag.flags&multiple != 0 {
			// each element becomes its own node
			for i := 0; i < target.Len(); i++ {
				err := e.encodeValueAsNode(name, tag, target.Index(i))
				if err != nil {
					return err
				}
			}
			return nil
		}
		child := NewNode(name)
		for i := 0; i < target.Len(); i++ {
			value, err := TryNewValue(target.Index(i).Interface())
			if err != nil {
				return err
			}
			child.AddArgument(value)
		}
		e.currentContext().AddNode(child)

	case reflect.Map:
		err := e.encodeMapAsNode(name, target)
		if err != nil {
			return err
		}

	default:
		return errors.Errorf("unsupported value kind %s", target.Kind())
	}
	return nil
}

func (e *encoder) encodeStructIntoProperties(node *Node, target reflect.Value) error {
	defer un(e.trace("encodeStructIntoProperties %s", target.Type()))
	ctx, err := newStructContext(target.Type())
	if err != nil {
		return err
	}

	for i := range target.NumField() {
		tag := ctx.tags[i]
		if tag.name == "" || tag.name == "-" {
			continue
		}
		field := target.Field(i)
		if !field.CanInterface() {
			panic(fmt.Sprintf("kdl.encoder: unexported field %s.%s, should be unreachable",
				target.Type(), target.Type().Field(i).Name))
		}

		if tag.flags&omitzero != 0 && field.IsZero() {
			continue
		}

		value, err := TryNewValue(field.Interface())
		if err != nil {
			return err
		}
		node.AddProperty(tag.name, value)
	}

	return nil
}

func (e *encoder) tryEncodeCustomMarshalerAsNode(name string, target reflect.Value) (bool, error) {
	defer un(e.trace("tryEncodeCustomMarshalerAsNode %s %s", name, target.Type()))
	getPointer := func() reflect.Value {
		if target.CanAddr() {
			return target.Addr()
		}

		// make addressable copy
		copy := reflect.New(target.Type()).Elem()
		copy.Set(target)
		return copy.Addr()
	}

	if target.Type().Implements(reflect.TypeFor[Marshaler]()) {
		n, err := target.Interface().(Marshaler).MarshalKDL()
		if err != nil {
			return false, err
		}
		if n.name == "" {
			n.name = name
		}
		e.currentContext().AddNode(n)
		return true, nil
	}

	if reflect.PointerTo(target.Type()).Implements(reflect.TypeFor[Marshaler]()) {
		ptr := getPointer()
		n, err := ptr.Interface().(Marshaler).MarshalKDL()
		if err != nil {
			return false, err
		}
		if n.name == "" {
			n.name = name
		}
		e.currentContext().AddNode(n)
		return true, nil
	}

	if target.Type().Implements(reflect.TypeFor[DocumentMarshaler]()) {
		doc, err := target.Interface().(DocumentMarshaler).MarshalKDLDocument()
		if err != nil {
			return false, err
		}
		e.currentContext().AddNode(NewNode(name).AddChildren(doc.Nodes...))
		return true, nil
	}

	if reflect.PointerTo(target.Type()).Implements(reflect.TypeFor[DocumentMarshaler]()) {
		ptr := getPointer()
		doc, err := ptr.Interface().(DocumentMarshaler).MarshalKDLDocument()
		if err != nil {
			return false, err
		}
		e.currentContext().AddNode(NewNode(name).AddChildren(doc.Nodes...))
		return true, nil
	}

	if target.Type().Implements(reflect.TypeFor[ValueMarshaler]()) {
		v, err := target.Interface().(ValueMarshaler).MarshalKDLValue()
		if err != nil {
			return false, err
		}
		childNode := NewKValue(name, v)
		e.currentContext().AddNode(childNode)
		return true, nil
	}

	if reflect.PointerTo(target.Type()).Implements(reflect.TypeFor[ValueMarshaler]()) {
		ptr := getPointer()
		v, err := ptr.Interface().(ValueMarshaler).MarshalKDLValue()
		if err != nil {
			return false, err
		}
		childNode := NewKValue(name, v)
		e.currentContext().AddNode(childNode)
		return true, nil
	}

	return false, nil
}
