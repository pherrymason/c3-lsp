package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) globalVariableDeclarationNodeToVariable(declarationNode *sitter.Node, moduleName string, docId *string, sourceCode []byte) []*idx.Variable {
	var variables []*idx.Variable
	//var typeNodeContent string
	var vType idx.Type

	//fmt.Println(declarationNode.ChildCount())
	//fmt.Println(declarationNode)
	//fmt.Println(declarationNode.Content(sourceCode))
	//fmt.Println("----")

	for i := uint32(0); i < declarationNode.ChildCount(); i++ {
		n := declarationNode.Child(int(i))
		//fmt.Println(i, ":", n.Type(), ":: ", n.Content(sourceCode), ":: has errors: ", n.HasError())
		switch n.Type() {
		case "type":
			//typeNodeContent = n.Content(sourceCode)
			vType = p.typeNodeToType(n, moduleName, sourceCode)
		case "ident":
			variable := idx.NewVariable(
				n.Content(sourceCode),
				vType,
				//idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
				moduleName,
				docId,
				idx.NewRangeFromTreeSitterPositions(
					n.StartPoint(),
					n.EndPoint(),
				),
				idx.NewRangeFromTreeSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, &variable)
		case ";":
			if n.HasError() && len(variables) > 0 {
				// Last variable is incomplete, remove it
				variables = variables[:len(variables)-1]
			}

		case "multi_declaration":
			sub := n.Child(1)
			variable := idx.NewVariable(
				sub.Content(sourceCode),
				vType,
				//idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
				moduleName,
				docId,
				idx.NewRangeFromTreeSitterPositions(
					sub.StartPoint(),
					sub.EndPoint(),
				),
				idx.NewRangeFromTreeSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, &variable)
		}

	}

	return variables
}

func (p *Parser) localVariableDeclarationNodeToVariable(declarationNode *sitter.Node, moduleName string, docId *string, sourceCode []byte) []*idx.Variable {
	var variables []*idx.Variable
	//var typeNodeContent string
	var vType idx.Type

	//fmt.Println(declarationNode.ChildCount())
	//fmt.Println(declarationNode)
	//fmt.Println(declarationNode.Content(sourceCode))

	for i := uint32(0); i < declarationNode.ChildCount(); i++ {
		n := declarationNode.Child(int(i))
		switch n.Type() {
		case "type":
			//typeNodeContent = n.Content(sourceCode)
			vType = p.typeNodeToType(n, moduleName, sourceCode)
			break

		case "local_decl_after_type":
			identifier := n.ChildByFieldName("name")

			variable := idx.NewVariable(
				identifier.Content(sourceCode),
				vType,
				//idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
				moduleName,
				docId,
				idx.NewRangeFromTreeSitterPositions(
					identifier.StartPoint(),
					identifier.EndPoint(),
				),
				idx.NewRangeFromTreeSitterPositions(
					declarationNode.StartPoint(),
					declarationNode.EndPoint()),
			)
			variables = append(variables, &variable)
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
func (p *Parser) nodeToConstant(node *sitter.Node, moduleName string, docId *string, sourceCode []byte) idx.Variable {
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
		idx.NewTypeFromString(typeNodeContent, moduleName), // <-- moduleName is potentially wrong
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(
			idNode.StartPoint(),
			idNode.EndPoint(),
		),
		idx.NewRangeFromTreeSitterPositions(
			node.StartPoint(),
			node.EndPoint()),
	)

	return constant
}
