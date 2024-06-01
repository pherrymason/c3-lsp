package parser

import (
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

/*
		func_definition: $ => seq(
	      'fn',
	      $.func_header,
	      $.fn_parameter_list,
	      optional($.attributes),
	      field('body', $.macro_func_body),
	    ),
		func_header: $ => seq(
			field('return_type', $._type_or_optional_type),
			optional(seq(field('method_type', $.type), '.')),
			field('name', $._func_macro_name),
		),
*/
func (p *Parser) nodeToFunction(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Function {
	var typeIdentifier string
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	if funcHeader.ChildByFieldName("method_type") != nil {
		typeIdentifier = funcHeader.ChildByFieldName("method_type").Content(sourceCode)
	}

	var argumentIds []string
	arguments := []*idx.Variable{}
	parameters := node.Child(2)
	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			argNode := parameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			argument := p.nodeToArgument(argNode, typeIdentifier, moduleName, docId, sourceCode)
			arguments = append(
				arguments,
				argument,
			)
			argumentIds = append(argumentIds, argument.GetName())
		}
	}

	var symbol idx.Function
	if typeIdentifier != "" {
		symbol = idx.NewTypeFunction(
			typeIdentifier,
			nameNode.Content(sourceCode),
			funcHeader.ChildByFieldName("return_type").Content(sourceCode), argumentIds,
			moduleName,
			docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
			protocol.CompletionItemKindFunction,
		)
	} else {
		symbol = idx.NewFunction(
			nameNode.Content(sourceCode),
			funcHeader.ChildByFieldName("return_type").Content(sourceCode), argumentIds,
			moduleName,
			docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
		)
	}

	var variables []*idx.Variable
	if node.ChildByFieldName("body") != nil {
		variables = p.FindVariableDeclarations(node, moduleName, docId, sourceCode)
	}

	for _, variable := range arguments {
		variables = append(variables, variable)
	}
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
func (p *Parser) nodeToArgument(argNode *sitter.Node, methodIdentifier string, moduleName string, docId string, sourceCode []byte) *idx.Variable {
	var identifier string = ""
	var idRange idx.Range
	var argType string = ""

	for i := uint32(0); i < argNode.ChildCount(); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "type":
			argType = n.Content(sourceCode)
		case "ident":
			identifier = n.Content(sourceCode)
			idRange = idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint())
			if identifier == "self" && methodIdentifier != "" {
				argType = methodIdentifier
			}
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

	variable := idx.NewVariable(
		identifier,
		idx.NewTypeFromString(argType, moduleName),
		moduleName,
		docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(argNode.StartPoint(),
			argNode.EndPoint()),
	)

	return &variable
}

/*
		func_definition: $ => seq(
	      'fn',
	      $.func_header,
	      $.fn_parameter_list,
	      optional($.attributes),
	      field('body', $.macro_func_body),
	    ),
		func_header: $ => seq(
			field('return_type', $._type_or_optional_type),
			optional(seq(field('method_type', $.type), '.')),
			field('name', $._func_macro_name),
		),


		macro_declaration: $ => seq(
	      'macro',
	      choice($.func_header, $.macro_header),
	      $.macro_parameter_list,
	      optional($.attributes),
	      field('body', $.macro_func_body),
	    ),
		macro_header: $ => seq(
		  optional(seq(field('method_type', $.type), '.')),
		  field('name', $._func_macro_name),
		),
*/
func (p *Parser) nodeToMacro(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Function {
	var typeIdentifier string
	var nameNode *sitter.Node
	funcHeader := node.Child(1)

	nameNode = funcHeader.ChildByFieldName("name")
	/*
		if funcHeader.Type() == "func_header" && funcHeader.ChildByFieldName("method_type") != nil {
			typeIdentifier = funcHeader.ChildByFieldName("method_type").Content(sourceCode)
		}*/

	var argumentIds []string
	arguments := []*idx.Variable{}
	parameters := node.Child(2)
	if parameters.ChildCount() > 2 {
		for i := uint32(0); i < parameters.ChildCount(); i++ {
			argNode := parameters.Child(int(i))
			if argNode.Type() != "parameter" {
				continue
			}

			argument := p.nodeToArgument(argNode, typeIdentifier, moduleName, docId, sourceCode)
			arguments = append(
				arguments,
				argument,
			)
			argumentIds = append(argumentIds, argument.GetName())
		}
	}

	symbol := idx.NewMacro(
		nameNode.Content(sourceCode),
		argumentIds,
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
			nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
			node.EndPoint()),
	)

	if node.ChildByFieldName("body") != nil {
		variables := p.FindVariableDeclarations(node, moduleName, docId, sourceCode)
		variables = append(arguments, variables...)
		symbol.AddVariables(variables)
	}

	return symbol
}
