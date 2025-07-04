package kdl

// A Marshaller can marshal itself to a KDL Node.
type Marshaller interface {
	MarshalKDL() (*Node, error)
}

// A DocumentMarshaller can marshal itself to a KDL Document.
type DocumentMarshaller interface {
	MarshalKDLDocument() (*Document, error)
}

// An Unmarshaller can unmarshal itself from a KDL Node.
type Unmarshaller interface {
	UnmarshalKDL(node *Node) error
}

// A DocumentUnmarshaller can unmarshal itself from a KDL Document.
type DocumentUnmarshaller interface {
	UnmarshalKDLDocument(doc *Document) error
}

// MarshalNodes marshals the given [Marshaller]s and adds them to the document's nodes.
func (d *Document) MarshalNodes(nodes ...Marshaller) error {
	if cap(d.Nodes)-len(d.Nodes) < len(nodes) {
		d.Nodes = append(make([]*Node, 0, len(d.Nodes)+len(nodes)), d.Nodes...)
	}

	for _, node := range nodes {
		n, err := node.MarshalKDL()
		if err != nil {
			return err
		}
		d.Nodes = append(d.Nodes, n)
	}
	return nil
}

// MarshalChildren marshals the given [Marshaller]s and adds them to the node's children.
func (n *Node) MarshalChildren(children ...Marshaller) error {
	if cap(n.Children)-len(n.Children) < len(children) {
		n.Children = append(make([]*Node, 0, len(n.Children)+len(children)), n.Children...)
	}

	for _, child := range children {
		c, err := child.MarshalKDL()
		if err != nil {
			return err
		}
		n.Children = append(n.Children, c)
	}
	return nil
}

// unmarshallable contrains type T such that *T implements the [Unmarshaller] interface.
type unmarshallable[T any] interface {
	*T
	Unmarshaller
}

// UnmarshalAll unmarshals the given nodes into a provided [Unmarshaller] type. It returns the first
// error encountered during unmarshalling, or nil if all nodes were successfully unmarshalled.
func UnmarshalAll[T any, U unmarshallable[T]](nodes []*Node) ([]*T, error) {
	out := make([]*T, 0, len(nodes))
	for _, node := range nodes {
		item := new(T)
		if err := U(item).UnmarshalKDL(node); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}
