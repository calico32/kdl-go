package kdl

import "testing"

func TestNewDocument(t *testing.T) {
	doc := NewDocument(
		NewNode("node1"),
		NewNode("node2"),
	)
	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected first node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
	if doc.Nodes[1].name != "node2" {
		t.Errorf("expected second node to be 'node2', got '%s'", doc.Nodes[1].name)
	}
}

func TestDocumentAddNode(t *testing.T) {
	doc := NewDocument()
	doc.AddNode(NewNode("node1"))
	if len(doc.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
}

func TestDocumentAddNodes(t *testing.T) {
	doc := NewDocument()
	doc.AddNodes(
		NewNode("node1"),
		NewNode("node2"),
	)
	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected first node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
	if doc.Nodes[1].name != "node2" {
		t.Errorf("expected second node to be 'node2', got '%s'", doc.Nodes[1].name)
	}
}

func TestDocumentRemoveNode(t *testing.T) {
	doc := NewDocument(
		NewNode("node1"),
		NewNode("node2"),
		NewNode("node3"),
		NewNode("node2"),
	)
	doc.RemoveNode("node2")
	if len(doc.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected first node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
	if doc.Nodes[1].name != "node3" {
		t.Errorf("expected second node to be 'node3', got '%s'", doc.Nodes[1].name)
	}
	if doc.Nodes[2].name != "node2" {
		t.Errorf("expected third node to be 'node2', got '%s'", doc.Nodes[2].name)
	}
}

func TestDocumentRemoveNodes(t *testing.T) {
	doc := NewDocument(
		NewNode("node1"),
		NewNode("node2"),
		NewNode("node3"),
		NewNode("node2"),
	)
	doc.RemoveNodes("node2")
	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected first node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
	if doc.Nodes[1].name != "node3" {
		t.Errorf("expected second node to be 'node3', got '%s'", doc.Nodes[1].name)
	}
}

func TestDocumentRemoveNodeAt(t *testing.T) {
	doc := NewDocument(
		NewNode("node1"),
		NewNode("node2"),
		NewNode("node3"),
	)
	doc.RemoveNodeAt(1)
	if len(doc.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(doc.Nodes))
	}
	if doc.Nodes[0].name != "node1" {
		t.Errorf("expected first node to be 'node1', got '%s'", doc.Nodes[0].name)
	}
	if doc.Nodes[1].name != "node3" {
		t.Errorf("expected second node to be 'node3', got '%s'", doc.Nodes[1].name)
	}
}

func TestDocumentGetNode(t *testing.T) {
	n2 := NewNode("node2")
	doc := NewDocument(
		NewNode("node1"),
		n2,
	)
	if got := doc.GetNode("node2"); got != n2 {
		t.Errorf("GetNode() = %v, want %v", got, n2)
	}
	if got := doc.GetNode("node3"); got != nil {
		t.Errorf("GetNode() = %v, want nil", got)
	}
}

func TestDocumentGetNodes(t *testing.T) {
	n2a := NewNode("node2")
	n2b := NewNode("node2")
	doc := NewDocument(
		NewNode("node1"),
		n2a,
		n2b,
	)
	nodes := doc.GetNodes("node2")
	if len(nodes) != 2 {
		t.Fatalf("GetNodes() len = %d, want 2", len(nodes))
	}
	if nodes[0] != n2a {
		t.Errorf("GetNodes()[0] = %v, want %v", nodes[0], n2a)
	}
	if nodes[1] != n2b {
		t.Errorf("GetNodes()[1] = %v, want %v", nodes[1], n2b)
	}
}

func TestDocumentGetKV(t *testing.T) {
	// node exists and has a value = nil error, correct value
	doc := NewDocument(
		NewNode("node1", NewString("value1")),
		NewNode("node2", NewString("value2")),
	)
	v, err := doc.GetKV("node1")
	if err != nil {
		t.Fatalf("GetKV(node1) error: %v", err)
	}
	if !v.Equal(NewString("value1")) {
		t.Errorf("GetKV(node1) = %s, want 'value1'", v)
	}

	// node does not exist = nil error, zero value
	v, err = doc.GetKV("node3")
	if err != nil {
		t.Fatalf("GetKV(node3) error: %v", err)
	}
	if v.IsValid() {
		t.Errorf("GetKV(node3) = %s, want invalid value", v)
	}

	// node exists but has no arguments = error
	doc = NewDocument(
		NewNode("node1"),
	)
	v, err = doc.GetKV("node1")
	if err == nil {
		t.Fatalf("GetKV(node1) expected error, got nil")
	}
	if v.IsValid() {
		t.Errorf("GetKV(node1) = %s, want invalid value", v)
	}

	// node exists but has too many arguments = error
	doc = NewDocument(
		NewNode("node1", NewString("value1"), NewString("extra")),
	)
	v, err = doc.GetKV("node1")
	if err == nil {
		t.Fatalf("GetKV(node1) expected error, got nil")
	}
	if v.IsValid() {
		t.Errorf("GetKV(node1) = %s, want invalid value", v)
	}
}
