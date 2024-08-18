package parser

import (
	"errors"

	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
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
func (p *Parser) nodeToFunction(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) (idx.Function, error) {
	var typeIdentifier string
	funcHeader := node.Child(1)
	nameNode := funcHeader.ChildByFieldName("name")

	if nameNode == nil || funcHeader == nil {
		return idx.Function{}, errors.New("child node not found")
	}

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

			argument := p.nodeToArgument(argNode, typeIdentifier, currentModule, docId, sourceCode)
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
			p.typeNodeToType(funcHeader.ChildByFieldName("return_type"), currentModule, sourceCode),
			//funcHeader.ChildByFieldName("return_type").Content(sourceCode),
			argumentIds,
			currentModule.GetModuleString(),
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
			p.typeNodeToType(funcHeader.ChildByFieldName("return_type"), currentModule, sourceCode),
			argumentIds,
			currentModule.GetModuleString(),
			docId,
			idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
				nameNode.EndPoint()),
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
				node.EndPoint()),
		)
	}

	var variables []*idx.Variable
	if node.ChildByFieldName("body") != nil {
		variables = p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
	}

	variables = append(variables, arguments...)

	symbol.AddVariables(variables)

	return symbol, nil
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
func (p *Parser) nodeToArgument(argNode *sitter.Node, methodIdentifier string, currentModule *idx.Module, docId *string, sourceCode []byte) *idx.Variable {
	var identifier string = ""
	var idRange idx.Range
	var argType idx.Type

	for i := uint32(0); i < argNode.ChildCount(); i++ {
		n := argNode.Child(int(i))

		switch n.Type() {
		case "type":
			argType = p.typeNodeToType(n, currentModule, sourceCode)
		case "ident":
			identifier = n.Content(sourceCode)
			idRange = idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint())
			// When detecting a self, the type is the Struct type
			if identifier == "self" && methodIdentifier != "" {
				argType = idx.NewTypeFromString(methodIdentifier, currentModule.GetModuleString())
			}
		}
	}

	variable := idx.NewVariable(
		identifier,
		argType,
		currentModule.GetModuleString(),
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
func (p *Parser) nodeToMacro(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Function {
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

			argument := p.nodeToArgument(argNode, typeIdentifier, currentModule, docId, sourceCode)
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
		currentModule.GetModuleString(),
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(),
			nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(),
			node.EndPoint()),
	)

	if node.ChildByFieldName("body") != nil {
		variables := p.FindVariableDeclarations(node, currentModule.GetModuleString(), currentModule, docId, sourceCode)
		variables = append(arguments, variables...)
		symbol.AddVariables(variables)
	}

	return symbol
}
