package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (p *Parser) nodeToFunction(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Function {
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	var argumentIds []string
	arguments := []idx.Variable{}
	parameters := node.Child(2)
	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			argNode := parameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			argument := p.nodeToArgument(doc, argNode, sourceCode)
			arguments = append(
				arguments,
				argument,
			)
			argumentIds = append(argumentIds, argument.GetName())
		}
	}
	/*
			for i := uint32(0); i < parameters.ChildCount(); i++ {
				argNode := parameters.Child(int(i))
				switch argNode.Type() {
				case "parameter":
					arguments = append(arguments, p.nodeToArgument(doc, argNode, sourceCode))
				}
			}


		for _, arg := range arguments {
			argumentIds = append(argumentIds, arg.GetName())
		}*/
	/*
		var symbol idx.Function
		if typeIdentifier != "" {
			symbol = idx.NewTypeFunction(typeIdentifier, nameNode.Content(sourceCode), node.ChildByFieldName("return_type").Content(sourceCode), argumentIds, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()), protocol.CompletionItemKindFunction)
		} else {
			symbol = idx.NewFunction(nameNode.Content(sourceCode), node.ChildByFieldName("return_type").Content(sourceCode), argumentIds, doc.ModuleName, doc.URI, idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()), idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()), protocol.CompletionItemKindFunction)
		}

		variables := p.FindVariableDeclarations(doc, node)
		variables = append(arguments, variables...)

		symbol.AddVariables(variables)
	*/

	var symbol idx.Function
	symbol = idx.NewFunction(
		nameNode.Content(sourceCode),
		funcHeader.ChildByFieldName("return_type").Content(sourceCode), argumentIds,
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(nameNode.StartPoint(),
			nameNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(),
			node.EndPoint()),
		protocol.CompletionItemKindFunction,
	)

	variables := p.FindVariableDeclarations(doc, node)
	variables = append(arguments, variables...)

	symbol.AddVariables(variables)

	return symbol
}

// nodeToArgument Very similar to nodeToVariable, but arguments have optional identifiers (for example when using `self` for struct methods)
/*
	_parameter: $ => choice(
      seq($.type, $.ident, optional($.attributes)),			// 3
      seq($.type, '...', $.ident, optional($.attributes)),	// 3/4
      seq($.type, '...', $.ct_ident),						// 3
      seq($.type, $.ct_ident),								// 2
      seq($.type, '...', optional($.attributes)),			// 2/3
      seq($.type, $.hash_ident, optional($.attributes)),	// 2/3
      seq($.type, '&', $.ident, optional($.attributes)),	// 3/4
      seq($.type, optional($.attributes)),					// 1/2
      seq('&', $.ident, optional($.attributes)),			// 2/3
      seq($.hash_ident, optional($.attributes)),			// 1/2
      '...',												// 1
      seq($.ident, optional($.attributes)),					// 1/2
      seq($.ident, '...', optional($.attributes)),			// 2/3
      $.ct_ident,											// 1
      seq($.ct_ident, '...'),								// 2
    ),
*/
func (p *Parser) nodeToArgument(doc *document.Document, argNode *sitter.Node, sourceCode []byte) idx.Variable {
	var identifier string = ""
	var idRange idx.Range
	var argType string = ""

	for i := uint32(0); i < argNode.ChildCount(); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "type":
			argType = n.Content(sourceCode)
			break
		case "ident":
			identifier = n.Content(sourceCode)
			break
		}
	}

	/*
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
		}*/

	variable := idx.NewVariable(identifier, argType, doc.ModuleName, doc.URI, idRange, idx.NewRangeFromSitterPositions(argNode.StartPoint(), argNode.EndPoint()))

	return variable
}
