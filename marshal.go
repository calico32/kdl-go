package kdl

// A ValueMarshaller can marshal itself to a KDL Value.
type ValueMarshaller interface {
	MarshalKDLValue() (Value, error)
}

// A Marshaller can marshal itself to a KDL Node.
type Marshaller interface {
	MarshalKDL() (*Node, error)
}

// A DocumentMarshaller can marshal itself to a KDL Document.
type DocumentMarshaller interface {
	MarshalKDLDocument() (*Document, error)
}

// A ValueUnmarshaller can unmarshal itself from a KDL Value.
type ValueUnmarshaller interface {
	UnmarshalKDLValue(value Value) error
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

// MarshalAll marshals the given [Marshaller]s and returns a slice of the
// resulting KDL nodes.
func MarshalAll[T Marshaller](items []T) ([]*Node, error) {
	out := make([]*Node, 0, len(items))
	for _, item := range items {
		node, err := item.MarshalKDL()
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}
