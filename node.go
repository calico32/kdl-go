package kdl

import (
	"maps"
	"slices"
)

// A Node represents a KDL node.
type Node struct {
	Name string
	// TypeAnnotation is the KDL type annotation for the node, if any.
	TypeAnnotation *string
	Arguments      []Value
	Properties     map[string]Value
	PropertyOrder  []string
	Children       Document

	// Parent points to the parent node of this node in the document tree, or
	// nil for top-level nodes.
	Parent *Node

	// Hints controls the behavior of the emitter when serializing the node.
	Hints emitterHints
}

type emitterHints struct {
	// EmitEmptyChildren controls whether to emit an empty children block when
	// the node has no children.
	EmitEmptyChildren bool
}

// NewNode creates a new KDL node with the given name.
func NewNode(name string) *Node {
	return &Node{
		Name:          name,
		Arguments:     []Value{},
		Properties:    map[string]Value{},
		PropertyOrder: []string{},
		Children:      Document{Nodes: []*Node{}},
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
	n.Arguments = append(n.Arguments, value)
	return n
}

// AddProperty adds a property to the KDL node with the given key and value and
// returns the node.
func (n *Node) AddProperty(key string, value Value) *Node {
	if !slices.Contains(n.PropertyOrder, key) {
		n.PropertyOrder = append(n.PropertyOrder, key)
	}
	n.Properties[key] = value
	return n
}

// AddChild adds a child node to the KDL node and returns the parent node.
func (n *Node) AddChild(child *Node) *Node {
	n.Children.AddNode(child)
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
	n.Children.AddNodes(children...)
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
	for _, child := range n.Children.Nodes {
		if child.Name == name {
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
	kvs := make([]KV, 0, len(n.Children.Nodes))
	for _, child := range n.Children.Nodes {
		if len(child.Arguments) == 1 {
			kvs = append(kvs, KV{Key: child.Name, Value: child.Arguments[0]})
		}
	}
	return kvs
}

// GetChildren gets all children with the given name from the KDL node and returns them.
//
// If no such children exist, it returns an empty slice.
func (n *Node) GetChildren(name string) []*Node {
	children := make([]*Node, 0, len(n.Children.Nodes))
	for _, child := range n.Children.Nodes {
		if child.Name == name {
			children = append(children, child)
		}
	}
	return children
}

// Clone creates a deep copy of the KDL node and returns it. The cloned node
// will not maintain a reference to the original node's parent.
func (n *Node) Clone() *Node {
	clone := &Node{
		Name:           n.Name,
		TypeAnnotation: n.TypeAnnotation,
		Arguments:      make([]Value, len(n.Arguments)),
		PropertyOrder:  make([]string, len(n.PropertyOrder)),
		Properties:     make(map[string]Value, len(n.Properties)),
		Children:       Document{make([]*Node, 0, len(n.Children.Nodes))},
		Hints:          n.Hints,
	}

	copy(clone.Arguments, n.Arguments)
	copy(clone.PropertyOrder, n.PropertyOrder)
	maps.Copy(clone.Properties, n.Properties)
	for _, child := range n.Children.Nodes {
		clone.Children.Nodes = append(clone.Children.Nodes, child.Clone())
	}

	return clone
}
