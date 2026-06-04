package kdl

import "fmt"

// A Document is a collection of nodes.
type Document struct {
	Nodes []*Node
	// TrailingComments holds comments that appear after the last node in the
	// document (or children block).
	TrailingComments []Comment
}

// NewDocument creates a new KDL document with the given nodes.
func NewDocument(nodes ...*Node) *Document {
	return &Document{Nodes: nodes}
}

// AddNode adds a node to the document and returns the document.
func (d *Document) AddNode(node *Node) *Document {
	d.Nodes = append(d.Nodes, node)
	return d
}

// AddNodes adds multiple nodes to the document and returns the document.
func (d *Document) AddNodes(nodes ...*Node) *Document {
	d.Nodes = append(d.Nodes, nodes...)
	return d
}

// GetNode gets the first node with the given name from the KDL document and
// returns it.
//
// If no such node exists, it returns nil.
func (d *Document) GetNode(name string) *Node {
	for _, child := range d.Nodes {
		if child.name == name {
			return child
		}
	}

	return nil
}

// GetNodes gets all nodes with the given name from the KDL document and returns
// them.
//
// If no such nodes exist, it returns an empty slice.
func (d *Document) GetNodes(name string) []*Node {
	children := make([]*Node, 0, len(d.Nodes))
	for _, child := range d.Nodes {
		if child.name == name {
			children = append(children, child)
		}
	}
	return children
}

// GetKV gets the first node with the given name from the KDL document and
// returns its first argument.
//
// If no such node exists, GetKV returns a zero Value and a nil error. If the
// node does not have exactly one argument, a non-nil error is returned.
func (d *Document) GetKV(name string) (Value, error) {
	for _, child := range d.Nodes {
		if child.name == name {
			if len(child.args) != 1 {
				return Value{}, fmt.Errorf("node %s does not have exactly one argument", name)
			}
			return child.Arg(0), nil
		}
	}

	return Value{}, nil
}
