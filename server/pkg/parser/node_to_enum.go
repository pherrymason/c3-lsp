package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
		enum_declaration: $ => seq(
			'enum',
			field('name', $.type_ident),
			optional($.interface_impl),
			optional($.enum_spec),
			optional($.attributes),
			field('body', $.enum_body),
		),

		enum_spec: $ => seq(
	      ':',
	      $.type,
	      optional($.enum_param_list),
	    ),

		enum_body: $ => seq(
			'{',
			commaSepTrailing1($.enum_constant),
			'}'
		),
		enum_constant: $ => seq(
			field('name', $.const_ident),
			field('args', optional($.enum_arg)),
			optional($.attributes),
		),
*/
func (p *Parser) nodeToEnum(node *sitter.Node, moduleName string, docId *string, sourceCode []byte) idx.Enum {
	// TODO parse attributes

	baseType := ""
	var enumerators []*idx.Enumerator
	var associatedParameters []idx.Variable

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			baseType = n.Child(1).Content(sourceCode)
			// Check if has enum_param_list
			if n.ChildCount() >= 3 {
				// Try to get enum_param_list
				param_list := n.Child(2)
				for p := 0; p < int(param_list.ChildCount()); p++ {
					paramNode := param_list.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						//fmt.Println(paramNode.Type(), paramNode.Content(sourceCode))
						associatedParameters = append(
							associatedParameters,
							idx.NewVariable(
								paramNode.Child(1).Content(sourceCode),
								idx.NewTypeFromString(paramNode.Child(0).Content(sourceCode), moduleName),
								moduleName,
								docId,
								idx.NewRangeFromTreeSitterPositions(paramNode.Child(0).StartPoint(), paramNode.Child(0).EndPoint()),
								idx.NewRangeFromTreeSitterPositions(paramNode.StartPoint(), paramNode.EndPoint()),
							),
						)
					}
				}
			}

		case "enum_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				enumeratorNode := n.Child(i)

				if enumeratorNode.Type() == "enum_constant" {
					name := enumeratorNode.ChildByFieldName("name")
					enumerator := idx.NewEnumerator(
						name.Content(sourceCode),
						"",
						associatedParameters,
						moduleName,
						idx.NewRangeFromTreeSitterPositions(name.StartPoint(), name.EndPoint()),
						docId,
					)
					enumerators = append(enumerators, enumerator)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	enum := idx.NewEnum(
		nameNode.Content(sourceCode),
		baseType,
		[]*idx.Enumerator{},
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	enum.AddEnumerators(enumerators)

	return enum
}
