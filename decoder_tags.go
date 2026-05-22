package kdl

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// tagFlags represents flags parsed from a struct field's KDL tag.
type tagFlags uint16

const (
	omitzero   tagFlags = 1 << iota // omit zero values when marshaling
	strict                          // enable strict unmarshaling for this field
	multiple                        // multiple nodes with the same name can be unmarshaled into a slice
	argument                        // consumes a single argument
	arguments                       // consumes all remaining arguments
	child                           // consumes a named child only (do not match properties)
	children                        // consumes all remaining children
	property                        // consumes a named property only (do not match children)
	properties                      // consumes all remaining properties
	presence                        // if a child with this name is present, set to true (only for bool fields)
)

func (tag tagFlags) String() string {
	var parts []string
	if tag&omitzero != 0 {
		parts = append(parts, "omitzero")
	}
	if tag&strict != 0 {
		parts = append(parts, "strict")
	}
	if tag&multiple != 0 {
		parts = append(parts, "multiple")
	}
	if tag&argument != 0 {
		parts = append(parts, "argument")
	}
	if tag&arguments != 0 {
		parts = append(parts, "arguments")
	}
	if tag&child != 0 {
		parts = append(parts, "child")
	}
	if tag&children != 0 {
		parts = append(parts, "children")
	}
	if tag&property != 0 {
		parts = append(parts, "property")
	}
	if tag&properties != 0 {
		parts = append(parts, "properties")
	}
	if tag&presence != 0 {
		parts = append(parts, "presence")
	}
	return strings.Join(parts, ",")
}

func lookupFlag(name string) tagFlags {
	switch name {
	case "omitzero":
		return omitzero
	case "strict":
		return strict
	case "multiple":
		return multiple
	case "argument", "arg":
		return argument
	case "arguments", "args":
		return arguments
	case "child":
		return child
	case "children":
		return children
	case "property", "prop":
		return property
	case "properties", "props":
		return properties
	case "presence":
		return presence
	default:
		return 0
	}
}

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
		if flag := lookupFlag(part); flag != 0 {
			if t.flags&flag != 0 {
				err = fmt.Errorf("duplicate tag flag %q", part)
				return
			}
			t.flags |= flag
			continue
		}
		if strings.HasPrefix(part, "format:") {
			t.format = part[7:]
			continue
		}

		err = fmt.Errorf("unknown tag flag %q", part)
		return
	}

	allowedCombinations := map[tagFlags][]tagFlags{
		omitzero:   {strict, arguments, property, properties, child, children, multiple, presence},
		strict:     {omitzero, argument, arguments, property, properties, child, children, multiple, presence},
		argument:   {strict},
		arguments:  {omitzero, strict},
		property:   {omitzero, strict},
		properties: {omitzero, strict, children},
		child:      {omitzero, strict, multiple, presence},
		children:   {omitzero, strict, properties},
		multiple:   {omitzero, strict, child},
		presence:   {omitzero, strict, child},
	}

	for thisFlag, allowed := range allowedCombinations {
		if t.flags&thisFlag != 0 {
			for otherFlag := range allowedCombinations {
				hasOtherFlag := otherFlag != thisFlag && t.flags&otherFlag != 0
				isAllowed := slices.Contains(allowed, otherFlag)
				if hasOtherFlag && !isAllowed {
					err = fmt.Errorf("%q flag cannot be combined with %q flag", thisFlag, otherFlag)
					return
				}
			}
		}
	}

	if t.flags&argument != 0 && t.name != "" {
		err = errors.New("argument tag cannot have a name")
		return
	}
	if (t.flags&child != 0 || t.flags&property != 0) && t.name == "" {
		err = errors.New("child/property tag must have a name")
		return
	}

	return
}

type structContext struct {
	tags           []structTag
	argsField      int // required
	propsField     int // required
	childrenField  int // required
	argFields      []int
	unusedFields   map[int]struct{}    // required
	strictFields   map[int]struct{}    // required
	usedChildren   map[int]struct{}    // required
	usedProperties map[string]struct{} // required
}

func newStructContext(typ reflect.Type) (*structContext, error) {
	ctx := &structContext{
		tags:           make([]structTag, typ.NumField()),
		argsField:      -1,
		propsField:     -1,
		childrenField:  -1,
		argFields:      []int{},
		unusedFields:   make(map[int]struct{}),
		strictFields:   make(map[int]struct{}),
		usedChildren:   make(map[int]struct{}),
		usedProperties: make(map[string]struct{}),
	}
	for fieldIndex := 0; fieldIndex < typ.NumField(); fieldIndex++ {
		field := typ.Field(fieldIndex)
		tagStr, hasKdlTag := field.Tag.Lookup("kdl")
		if !hasKdlTag {
			tagStr = field.Name
		}

		if !field.IsExported() {
			if hasKdlTag {
				return nil, fmt.Errorf("unexported field %q has kdl tag", field.Name)
			}

			// otherwise, ignore unexported field
			ctx.tags[fieldIndex] = structTag{}
			continue
		}

		tag, err := parseStructTag(tagStr)
		if err != nil {
			return nil, fmt.Errorf("parsing kdl tag for field %q: %w", field.Name, err)
		}
		ctx.tags[fieldIndex] = tag

		if tag.flags&strict != 0 {
			ctx.strictFields[fieldIndex] = struct{}{}
		}

		ctx.unusedFields[fieldIndex] = struct{}{}

		if tag.flags&argument != 0 {
			ctx.argFields = append(ctx.argFields, fieldIndex)
		}
		if tag.flags&arguments != 0 {
			if ctx.argsField != -1 {
				return nil, fmt.Errorf("multiple arguments fields in struct (field %q and %q in struct %s)", typ.Field(ctx.argsField).Name, field.Name, typ)
			}
			ctx.argsField = fieldIndex
		}
		if tag.flags&properties != 0 {
			if ctx.propsField != -1 {
				return nil, fmt.Errorf("multiple properties fields in struct (field %q and %q in struct %s)", typ.Field(ctx.propsField).Name, field.Name, typ)
			}
			ctx.propsField = fieldIndex
		}
		if tag.flags&children != 0 {
			if ctx.childrenField != -1 {
				return nil, fmt.Errorf("multiple children fields in struct (field %q and %q in struct %s)", typ.Field(ctx.childrenField).Name, field.Name, typ)
			}
			ctx.childrenField = fieldIndex
		}
	}

	return ctx, nil
}

func (ctx *structContext) markFieldUsed(index int) {
	delete(ctx.unusedFields, index)
	delete(ctx.strictFields, index)
}

func (ctx *structContext) markPropertyUsed(name string) {
	ctx.usedProperties[name] = struct{}{}
}

func (ctx *structContext) markChildUsed(index int) {
	ctx.usedChildren[index] = struct{}{}
}

func (ctx *structContext) isFieldStrict(index int) bool {
	_, ok := ctx.strictFields[index]
	return ok
}

func (ctx *structContext) isPropertyUnused(name string) bool {
	_, ok := ctx.usedProperties[name]
	return !ok
}

func (ctx *structContext) isChildUnused(index int) bool {
	_, ok := ctx.usedChildren[index]
	return !ok
}
