package parser

import sitter "github.com/smacker/go-tree-sitter"

func startPointSkippingDocComment(node *sitter.Node) sitter.Point {
	if node == nil {
		return sitter.Point{}
	}

	if node.ChildCount() == 0 {
		return node.StartPoint()
	}

	first := node.Child(0)
	if first != nil && first.Type() == "doc_comment" {
		for i := 1; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil {
				return child.StartPoint()
			}
		}
	}

	return node.StartPoint()
}
