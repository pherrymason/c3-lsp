package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
distinct_declaration: $ => seq(

	  'distinct',
	  field('name', $.type_ident),
	  optional($.interface_impl),  // TODO
	  optional($.attributes),      // TODO
	  '=',
	  optional('inline'),
	  $.type,
	  ';'
	),
*/
func (p *Parser) nodeToDistinct(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Distinct {
	start := startPointSkippingDocComment(node)

	distinctBuilder := idx.NewDistinctBuilder("", currentModule.GetModuleString(), *docId).
		WithDocumentRange(
			uint(start.Row),
			uint(start.Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		distinctBuilder.
			WithName(nameNode.Content(sourceCode)).
			WithIdentifierRange(
				uint(nameNode.StartPoint().Row),
				uint(nameNode.StartPoint().Column),
				uint(nameNode.EndPoint().Row),
				uint(nameNode.EndPoint().Column),
			)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "inline":
			distinctBuilder.WithInline(true)
		case "type":
			// Might contain module path
			_type := p.typeNodeToType(n, currentModule, sourceCode)
			distinctBuilder.WithBaseType(_type)
		}
	}

	return *distinctBuilder.Build()
}
