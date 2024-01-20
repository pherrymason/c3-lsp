package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToVariable(doc *document.Document, variableNode *sitter.Node, identifierNode *sitter.Node, sourceCode []byte, content string) idx.Variable {
	typeNode := identifierNode.PrevSibling()
	typeNodeContent := typeNode.Content(sourceCode)
	variable := idx.NewVariable(content, typeNodeContent, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()), idx.NewRangeFromSitterPositions(variableNode.StartPoint(), variableNode.EndPoint()))

	return variable
}
