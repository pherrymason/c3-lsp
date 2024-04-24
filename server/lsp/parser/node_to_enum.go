package parser

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
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
func (p *Parser) nodeToEnum(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Enum {
	// TODO parse attributes

	baseType := ""
	var enumerators []idx.Enumerator

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "enum_spec":
			baseType = n.Child(1).Content(sourceCode)

		case "enum_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				enumeratorNode := n.Child(i)

				if enumeratorNode.Type() == "enum_constant" {
					name := enumeratorNode.ChildByFieldName("name")
					enumerators = append(enumerators,
						idx.NewEnumerator(name.Content(sourceCode),
							"",
							"",
							idx.NewRangeFromSitterPositions(name.StartPoint(), name.EndPoint()),
							doc.URI,
						),
					)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	enum := idx.NewEnum(
		nameNode.Content(sourceCode),
		baseType,
		[]idx.Enumerator{},
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	enum.AddEnumerators(enumerators)

	return enum
}
