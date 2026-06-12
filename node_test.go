package kdl

import "testing"

func TestNewNode(t *testing.T) {
	node := NewNode("node1")
	if node.Name() != "node1" {
		t.Errorf("expected node name to be 'node1', got '%s'", node.name)
	}
}

func TestNodeAddChild(t *testing.T) {
	node := NewNode("parent")
	child := NewNode("child")
	node.AddChild(child)
	if len(node.Children().Nodes) != 1 {
		t.Errorf("expected 1 child, got %d", len(node.Children().Nodes))
	}
	if node.Children().Nodes[0] != child {
		t.Errorf("expected child to be the same as the one added")
	}
}

func TestNodeAddChildren(t *testing.T) {
	node := NewNode("parent")
	child1 := NewNode("child1")
	child2 := NewNode("child2")
	node.AddChildren(child1, child2)
	if len(node.Children().Nodes) != 2 {
		t.Errorf("expected 2 children, got %d", len(node.Children().Nodes))
	}
	if node.Children().Nodes[0] != child1 {
		t.Errorf("expected first child to be the same as the first one added")
	}
	if node.Children().Nodes[1] != child2 {
		t.Errorf("expected second child to be the same as the second one added")
	}
}

func TestNodeRemoveChild(t *testing.T) {
	node := NewNode("parent")
	child1 := NewNode("child1")
	child2 := NewNode("child2")
	child3 := NewNode("child3")
	node.AddChildren(child1, child2, child3)
	node.RemoveChild("child2")
	if len(node.Children().Nodes) != 2 {
		t.Errorf("expected 2 children, got %d", len(node.Children().Nodes))
	}
	if node.Children().Nodes[0] != child1 {
		t.Errorf("expected first child to be the same as the first one added")
	}
	if node.Children().Nodes[1] != child3 {
		t.Errorf("expected second child to be the same as the third one added")
	}
}

func TestNodeRemoveChildren(t *testing.T) {
	node := NewNode("parent")
	child1 := NewNode("child1")
	child2a := NewNode("child2")
	child3 := NewNode("child3")
	child2b := NewNode("child2")
	node.AddChildren(child1, child2a, child3, child2b)
	node.RemoveChildren("child2")
	if len(node.Children().Nodes) != 2 {
		t.Errorf("expected 2 children, got %d", len(node.Children().Nodes))
	}
	if node.Children().Nodes[0] != child1 {
		t.Errorf("expected first child to be the same as the first one added")
	}
	if node.Children().Nodes[1] != child3 {
		t.Errorf("expected second child to be the same as the third one added")
	}
}

func TestNodeRemoveChildAt(t *testing.T) {
	node := NewNode("parent")
	child1 := NewNode("child1")
	child2 := NewNode("child2")
	child3 := NewNode("child3")
	node.AddChildren(child1, child2, child3)
	node.RemoveChildAt(1)
	if len(node.Children().Nodes) != 2 {
		t.Errorf("expected 2 children, got %d", len(node.Children().Nodes))
	}
	if node.Children().Nodes[0] != child1 {
		t.Errorf("expected first child to be the same as the first one added")
	}
	if node.Children().Nodes[1] != child3 {
		t.Errorf("expected second child to be the same as the third one added")
	}
}
