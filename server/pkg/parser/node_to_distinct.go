package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
distinct_declaration: $ => seq(

	  'distinct',
	  field('name', $.type_ident),
	  optional($.interface_impl),
	  optional($.attributes),
	  '=',
	  optional('inline'),
	  $.type,
	  ';'
	),
*/
func (p *Parser) nodeToTypeDef(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.TypeDef {
	start := startPointSkippingDocComment(node)

	typeDefBuilder := idx.NewTypeDefBuilder("", currentModule.GetModuleString(), *docId).
		WithDocumentRange(
			uint(start.Row),
			uint(start.Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		typeDefBuilder.
			WithName(nameNode.Content(sourceCode)).
			WithIdentifierRange(
				uint(nameNode.StartPoint().Row),
				uint(nameNode.StartPoint().Column),
				uint(nameNode.EndPoint().Row),
				uint(nameNode.EndPoint().Column),
			)
	}

	attributes := parseNodeAttributes(node, sourceCode)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "inline":
			typeDefBuilder.WithInline(true)
		case "type":
			// Might contain module path
			_type := p.typeNodeToType(n, currentModule, sourceCode)
			typeDefBuilder.WithBaseType(_type)
		}
	}

	td := typeDefBuilder.Build()
	td.SetAttributes(attributes)
	return *td
}
