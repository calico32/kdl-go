package kdl

import (
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// tagFlags represents flags parsed from a struct field's KDL tag.
type tagFlags uint16

const (
	omitempty  tagFlags = 1 << iota // omit empty values when marshaling
	omitzero                        // omit zero values when marshaling
	strict                          // enable strict unmarshaling for this field
	multiple                        // multiple nodes with the same name can be unmarshaled into a slice
	argument                        // consumes a single argument
	arguments                       // consumes all remaining arguments
	child                           // consumes a named child only (do not match properties)
	children                        // consumes all remaining children
	property                        // consumes a named property only (do not match children)
	properties                      // consumes all remaining properties
)

func (tag tagFlags) String() string {
	var parts []string
	if tag&omitempty != 0 {
		parts = append(parts, "omitempty")
	}
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
	return strings.Join(parts, ",")
}

func lookupFlag(name string) tagFlags {
	switch name {
	case "omitempty":
		return omitempty
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
				err = errors.Errorf("duplicate tag flag %q", part)
				return
			}
			t.flags |= flag
			continue
		}
		if strings.HasPrefix(part, "format:") {
			t.format = part[7:]
			continue
		}

		err = errors.Errorf("unknown tag flag %q", part)
		return
	}

	allowedCombinations := map[tagFlags][]tagFlags{
		omitempty:  {argument, arguments, property, properties, child, children, multiple},
		omitzero:   {argument, arguments, property, properties, child, children, multiple},
		strict:     {omitzero, omitempty, argument, arguments, property, properties, child, children, multiple},
		argument:   {omitempty, omitzero, strict},
		arguments:  {omitempty, omitzero, strict},
		property:   {omitempty, omitzero, strict},
		properties: {omitempty, omitzero, strict, children},
		child:      {omitempty, omitzero, strict, multiple},
		children:   {omitempty, omitzero, strict, properties},
		multiple:   {omitempty, omitzero, strict, child},
	}

	for thisFlag, allowed := range allowedCombinations {
		if t.flags&thisFlag != 0 {
			for otherFlag := range allowedCombinations {
				hasOtherFlag := otherFlag != thisFlag && t.flags&otherFlag != 0
				isAllowed := slices.Contains(allowed, otherFlag)
				if hasOtherFlag && !isAllowed {
					err = errors.Errorf("%q flag cannot be combined with %q flag", thisFlag, otherFlag)
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
