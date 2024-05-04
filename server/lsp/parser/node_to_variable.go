package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) nodeToVariable(doc *document.Document, variableNode *sitter.Node, identifierNode *sitter.Node, sourceCode []byte, content string) idx.Variable {
	typeNode := identifierNode.PrevSibling()
	typeNodeContent := typeNode.Content(sourceCode)
	variable := idx.NewVariable(
		content,
		indexables.NewTypeFromString(typeNodeContent),
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()),
		idx.NewRangeFromSitterPositions(variableNode.StartPoint(), variableNode.EndPoint()),
	)

	return variable
}

func (p *Parser) globalVariableDeclarationNodeToVariable(doc *document.Document, declarationNode *sitter.Node, sourceCode []byte) []idx.Variable {
	var variables []idx.Variable
	var typeNodeContent string

	//fmt.Println(declarationNode.ChildCount())
	//fmt.Println(declarationNode)
	//fmt.Println(declarationNode.Content(sourceCode))
	//fmt.Println("----")

	for i := uint32(0); i < declarationNode.ChildCount(); i++ {
		n := declarationNode.Child(int(i))
		//fmt.Println(i, ":", n.Type(), ":: ", n.Content(sourceCode), ":: has errors: ", n.HasError())
		switch n.Type() {
		case "type":
			typeNodeContent = n.Content(sourceCode)
		case "ident":
			variable := idx.NewVariable(
				n.Content(sourceCode),
				indexables.NewTypeFromString(typeNodeContent),
				doc.ModuleName,
				doc.URI,
				idx.NewRangeFromSitterPositions(
					n.StartPoint(),
					n.EndPoint(),
				),
				idx.NewRangeFromSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, variable)
		case ";":
			if n.HasError() && len(variables) > 0 {
				// Last variable is incomplete, remove it
				variables = variables[:len(variables)-1]
			}

		case "multi_declaration":
			sub := n.Child(1)
			variable := idx.NewVariable(
				sub.Content(sourceCode),
				indexables.NewTypeFromString(typeNodeContent),
				doc.ModuleName,
				doc.URI,
				idx.NewRangeFromSitterPositions(
					sub.StartPoint(),
					sub.EndPoint(),
				),
				idx.NewRangeFromSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, variable)
		}

	}

	return variables
}

func (p *Parser) localVariableDeclarationNodeToVariable(doc *document.Document, declarationNode *sitter.Node, sourceCode []byte) []idx.Variable {
	var variables []idx.Variable
	var typeNodeContent string

	//fmt.Println(declarationNode.ChildCount())
	//fmt.Println(declarationNode)
	//fmt.Println(declarationNode.Content(sourceCode))

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
				indexables.NewTypeFromString(typeNodeContent),
				doc.ModuleName,
				doc.URI,
				idx.NewRangeFromSitterPositions(
					identifier.StartPoint(),
					identifier.EndPoint(),
				),
				idx.NewRangeFromSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, variable)
		}
	}

	return variables
}

/*
		const_declaration: $ => seq(
	      'const',
	      field('type', optional($.type)),
	      $.const_ident,
	      optional($.attributes),
	      optional($._assign_right_expr),
	      ';'
	    )
*/
func (p *Parser) nodeToConstant(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Variable {
	var constant idx.Variable
	var typeNodeContent string
	var idNode *sitter.Node

	//fmt.Println(node.ChildCount())
	//fmt.Println(node)
	//fmt.Println(node.Content(sourceCode))

	for i := uint32(0); i < node.ChildCount(); i++ {
		n := node.Child(int(i))
		switch n.Type() {
		case "type":
			typeNodeContent = n.Content(sourceCode)

		case "const_ident":
			idNode = n
		}
	}

	constant = idx.NewConstant(
		idNode.Content(sourceCode),
		indexables.NewTypeFromString(typeNodeContent),
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(
			idNode.StartPoint(),
			idNode.EndPoint(),
		),
		idx.NewRangeFromSitterPositions(
			node.StartPoint(),
			node.EndPoint()),
	)

	return constant
}
