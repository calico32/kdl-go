package kdl

import (
	"maps"
	"slices"
)

// A Node represents a KDL node.
type Node struct {
	name      string
	ty        string
	typeValid bool
	args      []Value
	props     map[string]Value // required
	propOrder []string
	children  Document

	// hints controls the behavior of the emitter when serializing the node.
	hints emitterHints

	// loc is the location of the node in the source file, if available.
	loc Location
}

// Name returns the name of the KDL node.
func (n *Node) Name() string { return n.name }

// TypeAnnotation returns the type annotation of the KDL node, if any.
func (n *Node) TypeAnnotation() (string, bool) { return n.ty, n.typeValid }

// Arguments returns the arguments of the KDL node.
func (n *Node) Arguments() []Value { return n.args }

// Properties returns the properties of the KDL node.
func (n *Node) Properties() map[string]Value { return n.props }

// PropertyOrder returns the order of properties in the KDL node.
func (n *Node) PropertyOrder() []string { return n.propOrder }

// Children returns the children of the KDL node.
func (n *Node) Children() *Document { return &n.children }

// Location returns the location of the KDL node in the source file, if available.
func (n *Node) Location() Location { return n.loc }

// Hints returns the emitter hints for the KDL node.
func (n *Node) Hints() *emitterHints { return &n.hints }

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
	return n
}

// RemoveArgument removes the argument at the given index from the KDL node and
// returns the node. If the index is out of bounds, RemoveArgument does nothing.
func (n *Node) RemoveArgument(index int) *Node {
	if index < 0 || index >= len(n.args) {
		return n
	}
	n.args = append(n.args[:index], n.args[index+1:]...)
	return n
}

// AddProperty adds a property to the KDL node with the given key and value and
// returns the node.
func (n *Node) AddProperty(key string, value Value) *Node {
	if !slices.Contains(n.propOrder, key) {
		n.propOrder = append(n.propOrder, key)
	}
	n.props[key] = value
	return n
}

// RemoveProperty removes the property with the given key from the KDL node and
// returns the node. If the property does not exist, RemoveProperty does
// nothing.
func (n *Node) RemoveProperty(key string) *Node {
	if _, ok := n.props[key]; ok {
		delete(n.props, key)
		n.propOrder = slices.DeleteFunc(n.propOrder, func(s string) bool { return s == key })
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
		name:      n.name,
		ty:        n.ty,
		args:      make([]Value, len(n.args)),
		propOrder: make([]string, len(n.propOrder)),
		props:     make(map[string]Value, len(n.props)),
		children:  Document{make([]*Node, 0, len(n.children.Nodes))},
		hints:     n.hints,
		loc:       n.loc,
	}

	copy(clone.args, n.args)
	copy(clone.propOrder, n.propOrder)
	maps.Copy(clone.props, n.props)
	for _, child := range n.children.Nodes {
		clone.children.Nodes = append(clone.children.Nodes, child.Clone())
	}

	return clone
}
