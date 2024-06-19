package parser

import (
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
define_declaration: $ => seq(

	  'def',
	  choice(
	    $.define_ident,				// TODO
	    $.define_attribute,			// TODO
	    seq(
	      $.type_ident,
	      optional($.attributes),	// TODO
	      '=',
	      $.typedef_type,
	    ),
	  ),
	  optional($.attributes),		// TODO
	  ';'
	),
*/
func (p *Parser) nodeToDef(node *sitter.Node, moduleName string, docId *string, sourceCode []byte) idx.Def {
	//fmt.Println(node)
	defBuilder := idx.NewDefBuilder("", moduleName, docId).
		WithDocumentRange(
			uint(node.StartPoint().Row),
			uint(node.StartPoint().Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident", "define_ident":
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierRange(
					uint(n.StartPoint().Row),
					uint(n.StartPoint().Column),
					uint(n.EndPoint().Row),
					uint(n.EndPoint().Column),
				)

		case "typedef_type":
			var _type idx.Type
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = p.typeNodeToType(n.Child(0), moduleName, sourceCode)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	return *defBuilder.Build()
}
