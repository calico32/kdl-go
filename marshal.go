package kdl

// A ValueMarshaler can marshal itself to a KDL Value.
type ValueMarshaler interface {
	MarshalKDLValue() (Value, error)
}

// A Marshaler can marshal itself to a KDL Node.
type Marshaler interface {
	MarshalKDL() (*Node, error)
}

// A DocumentMarshaler can marshal itself to a KDL Document.
type DocumentMarshaler interface {
	MarshalKDLDocument() (*Document, error)
}

// A ValueUnmarshaler can unmarshal itself from a KDL Value.
type ValueUnmarshaler interface {
	UnmarshalKDLValue(value Value) error
}

// An Unmarshaler can unmarshal itself from a KDL Node.
type Unmarshaler interface {
	UnmarshalKDL(node *Node) error
}

// A DocumentUnmarshaler can unmarshal itself from a KDL Document.
type DocumentUnmarshaler interface {
	UnmarshalKDLDocument(doc *Document) error
}

// MarshalNodes marshals the given [Marshaler]s and adds them to the document's nodes.
func (d *Document) MarshalNodes(nodes ...Marshaler) error {
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

// unmarshalable contrains type T such that *T implements the [Unmarshaler] interface.
type unmarshalable[T any] interface {
	*T
	Unmarshaler
}

// UnmarshalAll unmarshals the given nodes into a provided [Unmarshaler] type. It returns the first
// error encountered during unmarshaling, or nil if all nodes were successfully unmarshaled.
func UnmarshalAll[T any, U unmarshalable[T]](nodes []*Node) ([]*T, error) {
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

// MarshalAll marshals the given [Marshaler]s and returns a slice of the
// resulting KDL nodes.
func MarshalAll[T Marshaler](items []T) ([]*Node, error) {
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
