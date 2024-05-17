package symbols

type Node struct {
	symbol   Indexable
	children []*Node
}

func NewNode(symbol Indexable) Node {
	return Node{
		symbol: symbol,
	}
}

func (n *Node) Insert(node *Node) {
	n.children = append(n.children, n)
}

func (n Node) GetSymbol() Indexable {
	return n.symbol
}
