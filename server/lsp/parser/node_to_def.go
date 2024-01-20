package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToDef(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Def {
	identifierNode := node.Child(1)
	definition := ""

	for i := uint32(3); i < (node.ChildCount() - 1); i++ {
		definition += node.Child(int(i)).Content(sourceCode)
	}

	return idx.NewDef(identifierNode.Content(sourceCode), definition, "", doc.URI, idx.NewRangeFromSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()))
}
