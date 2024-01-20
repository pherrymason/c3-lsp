package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToEnum(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Enum {
	// TODO parse attributes
	nodesCount := node.ChildCount()
	nameNode := node.Child(1)

	baseType := ""
	bodyIndex := int(nodesCount - 1)

	enum := idx.NewEnum(nameNode.Content(sourceCode), baseType, []idx.Enumerator{}, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()))

	enumeratorsNode := node.Child(bodyIndex)

	for i := uint32(0); i < enumeratorsNode.ChildCount(); i++ {
		enumeratorNode := enumeratorsNode.Child(int(i))
		if enumeratorNode.Type() == "enumerator" {
			enum.RegisterEnumerator(
				enumeratorNode.Child(0).Content(sourceCode),
				"",
				idx.NewRangeFromSitterPositions(enumeratorNode.StartPoint(), enumeratorNode.EndPoint()),
			)
		}
	}

	return enum
}
