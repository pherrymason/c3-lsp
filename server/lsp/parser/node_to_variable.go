package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) globalVariableDeclarationNodeToVariable(declarationNode *sitter.Node, moduleName string, docId string, sourceCode []byte) []idx.Variable {
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
				moduleName,
				docId,
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
				moduleName,
				docId,
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

func (p *Parser) localVariableDeclarationNodeToVariable(declarationNode *sitter.Node, moduleName string, docId string, sourceCode []byte) []idx.Variable {
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
				moduleName,
				docId,
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
func (p *Parser) nodeToConstant(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Variable {
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
		moduleName,
		docId,
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
