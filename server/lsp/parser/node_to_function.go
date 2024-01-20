package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (p *Parser) nodeToFunction(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Function {

	nameNode := node.ChildByFieldName("name")

	// Extract function arguments
	arguments := []idx.Variable{}
	parameters := node.ChildByFieldName("parameters")
	for i := uint32(0); i < parameters.ChildCount(); i++ {
		argNode := parameters.Child(int(i))
		switch argNode.Type() {
		case "parameter":
			arguments = append(arguments, p.nodeToArgument(doc, argNode, sourceCode))
		}
	}

	var argumentIds []string
	for _, arg := range arguments {
		argumentIds = append(argumentIds, arg.GetName())
	}

	symbol := idx.NewFunction(nameNode.Content(sourceCode), node.ChildByFieldName("return_type").Content(sourceCode), argumentIds, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()), protocol.CompletionItemKindFunction)

	variables := p.FindVariableDeclarations(doc, node)
	variables = append(arguments, variables...)

	// TODO Previous node has the info about which function is belongs to.

	symbol.AddVariables(variables)

	return symbol
}

// nodeToArgument Very similar to nodeToVariable, but arguments have optional identifiers (for example when using `self` for struct methods)
func (p *Parser) nodeToArgument(doc *document.Document, argNode *sitter.Node, sourceCode []byte) idx.Variable {
	var identifier string = ""
	var idRange idx.Range
	var argType string = ""
	if argNode.ChildCount() == 2 {
		if argNode.Child(0).Type() == "identifier" {
			// argument without type
			idNode := argNode.Child(0)
			identifier = idNode.Content(sourceCode)
			idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
		} else {
			// first node is type
			argType = argNode.Child(0).Content(sourceCode)

			idNode := argNode.Child(1)
			identifier = idNode.Content(sourceCode)
			idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
		}
	} else if argNode.ChildCount() == 1 {
		idNode := argNode.Child(0)
		identifier = idNode.Content(sourceCode)
		idRange = idx.NewRangeFromSitterPositions(idNode.StartPoint(), idNode.EndPoint())
	}

	variable := idx.NewVariable(identifier, argType, doc.ModuleName, doc.URI, idRange, idx.NewRangeFromSitterPositions(argNode.StartPoint(), argNode.EndPoint()))

	return variable
}
