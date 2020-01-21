package xml2json

// Node is a data element on a tree
type Node struct {
	Children map[string]Nodes
	Data     string
}

// Nodes is a list of nodes
type Nodes []*Node

// AddChild appends a node to the list of children
func (n *Node) AddChild(s string, c *Node) {
	// Lazy lazy
	if n.Children == nil {
		n.Children = map[string]Nodes{}
	}

	n.Children[s] = append(n.Children[s], c)
}

// IsComplex returns whether it is a complex type (has children)
func (n *Node) IsComplex() bool {
	return len(n.Children) > 0
}
