package kdl

// Walk calls fn for each node in doc in depth-first pre-order. If fn returns
// false for a node, that node's children are not visited.
func Walk(doc *Document, fn func(node *Node, depth int) bool) {
	if doc == nil {
		return
	}
	walkDepth(doc.Nodes, 0, fn)
}

func walkDepth(nodes []*Node, depth int, fn func(*Node, int) bool) {
	for _, n := range nodes {
		if fn(n, depth) {
			if ch := n.Children(); ch != nil && len(ch.Nodes) > 0 {
				walkDepth(ch.Nodes, depth+1, fn)
			}
		}
	}
}
