package parser

import sitter "github.com/smacker/go-tree-sitter"

func parseNodeAttributes(node *sitter.Node, sourceCode []byte) []string {
	attributes := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "attributes" {
			continue
		}

		for j := 0; j < int(child.ChildCount()); j++ {
			attributeNode := child.Child(j)
			attributes = append(attributes, attributeNode.Content(sourceCode))
		}
	}

	return attributes
}

func firstChildOfType(node *sitter.Node, wantedType string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == wantedType {
			return child
		}
	}

	return nil
}
