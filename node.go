package kdl

import (
	"maps"
	"slices"
)

// A nodeEntryKind tags an entry in a Node's args/props insertion order.
type nodeEntryKind uint8

const (
	nodeEntryArg nodeEntryKind = iota
	nodeEntryProp
)

// A Node represents a KDL node.
type Node struct {
	name      string
	typ       string
	typeValid bool
	args      []Value
	props     map[string]Value // required
	propOrder []string
	// entries records the source/insertion order of args and props. Each entry
	// is either nodeEntryArg or nodeEntryProp; the i-th nodeEntryArg refers to
	// args[i] and the i-th nodeEntryProp refers to propOrder[i].
	entries  []nodeEntryKind
	children Document

	// hints controls the behavior of the emitter when serializing the node.
	hints emitterHints

	// blankLineBefore is set by the parser when one or more blank lines precede
	// this node in the source.
	blankLineBefore bool

	// childrenInline tracks how the children block was formatted in the source.
	// nil = programmatic (no source), &true = inline in source, &false = multiline in source.
	childrenInline *bool

	// leadingComments holds comments that appear on lines before this node.
	leadingComments []Comment

	// trailingComment holds a single-line comment that appears on the same line
	// as this node (after all arguments, properties, and children).
	trailingComment *Comment

	// inlineSlashdashes holds /- comments on individual args, props, or children
	// blocks within this node's body, in source order.
	inlineSlashdashes []InlineSlashdash

	// loc is the location of the start of the node name in the source file, if available.
	loc Location

	// nameEndLoc is the location of the end (exclusive) of the node name token.
	nameEndLoc Location

	// endLoc is the location of the end (exclusive) of the node in the source file,
	// covering the last token of the node body (last arg/prop value, or closing } of
	// children block, or last inline slashdash target).
	endLoc Location

	// typeAnnotStart is the location of the type annotation content (ident
	// inside the parens), if available.
	typeAnnotStart Location
	// typeAnnotEnd is the end location (exclusive) of the type annotation
	// content, if available.
	typeAnnotEnd Location

	// propKeyStart stores the source location of the key token for each property, if available.
	propKeyStart map[string]Location
	// propKeyEnd stores the exclusive end location of the key token for each
	// property, if available.
	propKeyEnd map[string]Location
}

// Name returns the name of the KDL node.
func (n *Node) Name() string { return n.name }

// TypeAnnotation returns the type annotation of the KDL node, if any.
func (n *Node) TypeAnnotation() (string, bool) { return n.typ, n.typeValid }

// Arguments returns the arguments of the KDL node.
func (n *Node) Arguments() []Value { return n.args }

// Properties returns the properties of the KDL node.
func (n *Node) Properties() map[string]Value { return n.props }

// PropertyOrder returns the order of properties in the KDL node.
func (n *Node) PropertyOrder() []string { return n.propOrder }

// entriesConsistent reports whether n.entries matches the current args/props.
func (n *Node) entriesConsistent() bool {
	if len(n.entries) != len(n.args)+len(n.propOrder) {
		return false
	}
	var a, p int
	for _, k := range n.entries {
		switch k {
		case nodeEntryArg:
			a++
		case nodeEntryProp:
			p++
		}
	}
	return a == len(n.args) && p == len(n.propOrder)
}

// removeNthEntry removes the n-th (0-based) occurrence of kind k from entries.
// It returns true if an entry was removed, or false if no such entry exists.
func (n *Node) removeNthEntry(k nodeEntryKind, idx int) bool {
	count := 0
	for i, e := range n.entries {
		if e != k {
			continue
		}
		if count == idx {
			n.entries = append(n.entries[:i], n.entries[i+1:]...)
			return true
		}
		count++
	}
	return false
}

// Children returns the children of the KDL node.
func (n *Node) Children() *Document { return &n.children }

// Location returns the location of the start of the node name in the source
// file, if available. Returns a zero Location when location tracking is off.
func (n *Node) Location() Location { return n.loc }

// NameEndLocation returns the location of the end (exclusive) of the node name
// token in the source file. Returns a zero Location when location tracking is off.
func (n *Node) NameEndLocation() Location { return n.nameEndLoc }

// EndLocation returns the location of the end (exclusive) of the node in the
// source file. Returns a zero Location when location tracking is off or the
// node was programmatically created.
func (n *Node) EndLocation() Location { return n.endLoc }

// TypeAnnotationRange returns the source range of the type annotation content
// (the identifier inside the parentheses, not the parens themselves). ok is
// false when no type annotation is present or location tracking is off.
func (n *Node) TypeAnnotationRange() (start, end Location, ok bool) {
	if !n.typeValid || n.typeAnnotStart.Line == 0 {
		return
	}
	return n.typeAnnotStart, n.typeAnnotEnd, true
}

// PropertyKeyLocation returns the source range of the key token for the named
// property. Returns zero Locations and ok=false when the property does not
// exist or location tracking is off.
func (n *Node) PropertyKeyLocation(key string) (start, end Location, ok bool) {
	if n.propKeyStart == nil {
		return
	}
	start, ok = n.propKeyStart[key]
	if !ok {
		return
	}
	end = n.propKeyEnd[key]
	return start, end, true
}

// setPropertyKeyLocation stores the source range of a property key.
func (n *Node) setPropertyKeyLocation(key string, start, end Location) {
	if n.propKeyStart == nil {
		n.propKeyStart = make(map[string]Location)
		n.propKeyEnd = make(map[string]Location)
	}
	n.propKeyStart[key] = start
	n.propKeyEnd[key] = end
}

// Hints returns the emitter hints for the KDL node.
func (n *Node) Hints() *emitterHints { return &n.hints }

// HasBlankLineBefore reports whether a blank line preceded this node in the
// parsed source.
func (n *Node) HasBlankLineBefore() bool { return n.blankLineBefore }

// LeadingComments returns comments that appear on lines before this node in the
// parsed source. Single-line and multi-line comments are preserved exactly;
// slashdash comments carry the commented-out node for re-formatting.
func (n *Node) LeadingComments() []Comment { return n.leadingComments }

// TrailingComment returns the single-line comment on the same line as this node,
// if any. ok is false when no trailing comment is present.
func (n *Node) TrailingComment() (c Comment, ok bool) {
	if n.trailingComment == nil {
		return Comment{}, false
	}
	return *n.trailingComment, true
}

// InlineSlashdashes returns the ordered list of /- comments on args, props, and
// children blocks within this node's body.
func (n *Node) InlineSlashdashes() []InlineSlashdash { return n.inlineSlashdashes }

// ChildrenInline reports how the children block was formatted in the parsed
// source. ok is false when the node was programmatically created (no source).
func (n *Node) ChildrenInline() (inline, ok bool) {
	if n.childrenInline == nil {
		return false, false
	}
	return *n.childrenInline, true
}

// NewNode creates a new KDL node with the given name.
func NewNode(name string) *Node {
	return &Node{
		name:  name,
		props: map[string]Value{},
	}
}

// NewKV creates a new KDL node with the given name and a single argument
// representing the given value.
func NewKV[T intoValue](name string, value T) *Node {
	return NewNode(name).AddArgument(NewValue(value))
}

// NewKValue creates a new KDL node with the given name and the given value as
// an argument.
func NewKValue(name string, value Value) *Node {
	return NewNode(name).AddArgument(value)
}

// AddArgument adds a value as an argument to the KDL node and returns the node.
func (n *Node) AddArgument(value Value) *Node {
	n.args = append(n.args, value)
	n.entries = append(n.entries, nodeEntryArg)
	return n
}

// RemoveArgument removes the argument at the given index from the KDL node and
// returns the node. If the index is out of bounds, RemoveArgument does nothing.
func (n *Node) RemoveArgument(index int) *Node {
	if index < 0 || index >= len(n.args) {
		return n
	}
	n.args = append(n.args[:index], n.args[index+1:]...)
	n.removeNthEntry(nodeEntryArg, index)
	return n
}

// AddProperty adds a property to the KDL node with the given key and value and
// returns the node.
func (n *Node) AddProperty(key string, value Value) *Node {
	if !slices.Contains(n.propOrder, key) {
		n.propOrder = append(n.propOrder, key)
		n.entries = append(n.entries, nodeEntryProp)
	}
	n.props[key] = value
	return n
}

// RemoveProperty removes the property with the given key from the KDL node and
// returns the node. If the property does not exist, RemoveProperty does
// nothing.
func (n *Node) RemoveProperty(key string) *Node {
	if _, ok := n.props[key]; ok {
		idx := slices.Index(n.propOrder, key)
		delete(n.props, key)
		n.propOrder = slices.DeleteFunc(n.propOrder, func(s string) bool { return s == key })
		if idx >= 0 {
			n.removeNthEntry(nodeEntryProp, idx)
		}
	}
	return n
}

// AddChild adds a child node to the KDL node and returns the parent node.
func (n *Node) AddChild(child *Node) *Node {
	n.children.AddNode(child)
	return n
}

// AddKV adds a key-value pair as a child node with the given name and value and
// returns the parent node.
func (n *Node) AddKV(name string, value Value) *Node {
	n.AddChild(NewKValue(name, value))
	return n
}

// NewChild creates a new child node with the given name, adds it to the parent
// node, and returns the new child node.
func (n *Node) NewChild(name string) *Node {
	child := NewNode(name)
	n.AddChild(child)
	return child
}

// NewKV creates a new child node with the given name and value, adds it to the
// parent node, and returns the parent node.
func (n *Node) NewKV(name string, value Value) *Node {
	child := NewKValue(name, value)
	n.AddChild(child)
	return n
}

// AddChildren adds multiple child nodes to the KDL node and returns the parent
// node.
func (n *Node) AddChildren(children ...*Node) *Node {
	n.children.AddNodes(children...)
	return n
}

// AddChildrenFunc calls fn, adds all the returned nodes as children, and
// returns the parent node.
func (n *Node) AddChildrenFunc(fn func(n *Document)) *Node {
	d := NewDocument()
	fn(d)
	n.AddChildren(d.Nodes...)
	return n
}

// GetChild gets the first child with the given name from the KDL node and returns it.
//
// If no such child exists, it returns nil.
func (n *Node) GetChild(name string) *Node {
	for _, child := range n.children.Nodes {
		if child.name == name {
			return child
		}
	}

	return nil
}

type KV struct {
	Key   string
	Value Value
}

// GetKVs gets all child nodes that have a single argument and returns them as a slice of key-value pairs.
//
// If no such children exist, it returns an empty slice.
func (n *Node) GetKVs() []KV {
	kvs := make([]KV, 0, len(n.children.Nodes))
	for _, child := range n.children.Nodes {
		if len(child.args) == 1 {
			kvs = append(kvs, KV{Key: child.name, Value: child.args[0]})
		}
	}
	return kvs
}

// GetChildren gets all children with the given name from the KDL node and returns them.
//
// If no such children exist, it returns an empty slice.
func (n *Node) GetChildren(name string) []*Node {
	children := make([]*Node, 0, len(n.children.Nodes))
	for _, child := range n.children.Nodes {
		if child.name == name {
			children = append(children, child)
		}
	}
	return children
}

// Clone creates a deep copy of the KDL node and returns it.
func (n *Node) Clone() *Node {
	clone := &Node{
		name:            n.name,
		typ:             n.typ,
		typeValid:       n.typeValid,
		args:            make([]Value, len(n.args)),
		propOrder:       make([]string, len(n.propOrder)),
		entries:         make([]nodeEntryKind, len(n.entries)),
		props:           make(map[string]Value, len(n.props)),
		children:        Document{Nodes: make([]*Node, 0, len(n.children.Nodes))},
		hints:           n.hints,
		blankLineBefore: n.blankLineBefore,
		loc:             n.loc,
		nameEndLoc:      n.nameEndLoc,
		endLoc:          n.endLoc,
		typeAnnotStart:  n.typeAnnotStart,
		typeAnnotEnd:    n.typeAnnotEnd,
	}

	if n.childrenInline != nil {
		v := *n.childrenInline
		clone.childrenInline = &v
	}
	if len(n.leadingComments) > 0 {
		clone.leadingComments = make([]Comment, len(n.leadingComments))
		copy(clone.leadingComments, n.leadingComments)
	}
	if n.trailingComment != nil {
		c := *n.trailingComment
		clone.trailingComment = &c
	}
	if len(n.inlineSlashdashes) > 0 {
		clone.inlineSlashdashes = make([]InlineSlashdash, len(n.inlineSlashdashes))
		for i, sd := range n.inlineSlashdashes {
			c := sd
			if sd.childrenInline != nil {
				v := *sd.childrenInline
				c.childrenInline = &v
			}
			// Deep-copy children nodes
			if len(sd.children.Nodes) > 0 {
				c.children.Nodes = make([]*Node, len(sd.children.Nodes))
				for j, ch := range sd.children.Nodes {
					c.children.Nodes[j] = ch.Clone()
				}
			}
			clone.inlineSlashdashes[i] = c
		}
	}
	copy(clone.args, n.args)
	copy(clone.propOrder, n.propOrder)
	copy(clone.entries, n.entries)
	maps.Copy(clone.props, n.props)
	if n.propKeyStart != nil {
		clone.propKeyStart = maps.Clone(n.propKeyStart)
		clone.propKeyEnd = maps.Clone(n.propKeyEnd)
	}
	for _, child := range n.children.Nodes {
		clone.children.Nodes = append(clone.children.Nodes, child.Clone())
	}
	if len(n.children.TrailingComments) > 0 {
		clone.children.TrailingComments = make([]Comment, len(n.children.TrailingComments))
		copy(clone.children.TrailingComments, n.children.TrailingComments)
	}

	return clone
}
