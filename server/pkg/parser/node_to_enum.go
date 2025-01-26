package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
enum_arg: $ => seq('=', $._expr),
enum_constant: $ => seq(

	field('name', $.const_ident),
	optional($.attributes),
	field('args', optional($.enum_arg)),

),
enum_param_declaration: $ => seq(

	field('type', $.type),
	field('name', $.ident),

),
enum_param_list: $ => seq('(', commaSep($.enum_param_declaration), ')'),
enum_spec: $ => prec.right(seq(

	  ':',
	  field('type', optional($.type)),
	  optional($.enum_param_list),
	)),
	enum_body: $ => seq(
	  '{',
	  commaSepTrailing($.enum_constant),
	  '}'
	),

enum_declaration: $ => seq(

	  'enum',
	  field('name', $.type_ident),
	  optional($.interface_impl),
	  optional($.enum_spec),
	  optional($.attributes),
	  field('body', $.enum_body),
	),
*/
func (p *Parser) nodeToEnum(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Enum {
	// TODO parse attributes

	baseType := ""
	var enumerators []*idx.Enumerator
	var associatedParameters []idx.Variable

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			typeNode := n.ChildByFieldName("type")
			paramListIndex := 1
			if typeNode != nil {
				// Custom enum backing type is optional
				baseType = typeNode.Content(sourceCode)
				paramListIndex = 2
			}

			paramList := n.Child(paramListIndex)
			// Check if has enum_param_list
			if paramList != nil {
				// Try to get enum_param_list
				for p := 0; p < int(paramList.ChildCount()); p++ {
					paramNode := paramList.Child(p)
					if paramNode.Type() == "enum_param_declaration" {
						paramTypeNode := paramNode.ChildByFieldName("type")
						paramNameNode := paramNode.ChildByFieldName("name")
						if paramTypeNode == nil || paramNameNode == nil {
							continue
						}

						//fmt.Println(paramNode.Type(), paramNode.Content(sourceCode))
						associatedParameters = append(
							associatedParameters,
							idx.NewVariable(
								paramNameNode.Content(sourceCode),
								idx.NewTypeFromString(paramTypeNode.Content(sourceCode), currentModule.GetModuleString()),
								currentModule.GetModuleString(),
								*docId,
								idx.NewRangeFromTreeSitterPositions(paramNameNode.StartPoint(), paramNameNode.EndPoint()),
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
					if name == nil {
						// Invalid node
						continue
					}
					enumerator := idx.NewEnumerator(
						name.Content(sourceCode),
						"",
						associatedParameters,
						currentModule.GetModuleString(),
						idx.NewRangeFromTreeSitterPositions(name.StartPoint(), name.EndPoint()),
						*docId,
					)
					enumerators = append(enumerators, enumerator)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	name := ""
	idRange := idx.NewRange(0, 0, 0, 0)
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
		idRange = idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint())
	}

	enum := idx.NewEnum(
		name,
		baseType,
		[]*idx.Enumerator{},
		currentModule.GetModuleString(),
		*docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	enum.AddEnumerators(enumerators)

	return enum
}
