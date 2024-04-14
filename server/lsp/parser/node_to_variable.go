package parser

import (
	"fmt"

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

func (p *Parser) globalVariableDeclarationNodeToVariable(doc *document.Document, declarationNode *sitter.Node, sourceCode []byte) idx.Variable {
	typeNode := declarationNode.ChildByFieldName("type")
	typeNodeContent := typeNode.Content(sourceCode)

	identifierNode := declarationNode.Child(1)
	fmt.Println(identifierNode.Content(sourceCode))

	variable := idx.NewVariable(
		identifierNode.Content(sourceCode),
		typeNodeContent,
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(
			identifierNode.StartPoint(),
			identifierNode.EndPoint(),
		),
		idx.NewRangeFromSitterPositions(
			declarationNode.StartPoint(),
			declarationNode.EndPoint()),
	)

	return variable
}

func (p *Parser) localVariableDeclarationNodeToVariable(doc *document.Document, declarationNode *sitter.Node, sourceCode []byte) []idx.Variable {
	//typeNode := declarationNode.ChildByFieldName("type")
	var variables []idx.Variable
	var typeNodeContent string

	fmt.Println(declarationNode.ChildCount())
	fmt.Println(declarationNode)
	fmt.Println(declarationNode.Content(sourceCode))

	for i := uint32(0); i < declarationNode.ChildCount(); i++ {
		n := declarationNode.Child(int(i))
		switch n.Type() {
		case "type":
			typeNodeContent = n.Content(sourceCode)
			break

		case "local_decl_after_type":
			identifier := n.ChildByFieldName("name")

			variable := idx.NewVariable(
				identifier.Content(sourceCode),
				typeNodeContent,
				doc.ModuleName,
				doc.URI,
				idx.NewRangeFromSitterPositions(
					identifier.StartPoint(),
					identifier.EndPoint(),
				),
				idx.NewRangeFromSitterPositions(
					n.StartPoint(),
					n.EndPoint()),
			)
			variables = append(variables, variable)
		}

		fmt.Println(
			declarationNode.Child(int(i)).Type(),
			declarationNode.Child(int(i)).Content(sourceCode))
	}

	return variables
}
