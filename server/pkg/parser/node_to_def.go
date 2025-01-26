package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
define_ident: $ => seq(

	choice(
	  seq($.ident, '=', $.define_path_ident),
	  seq($.const_ident, '=', $.path_const_ident),
	  seq($.at_ident, '=', $.define_path_at_ident),
	),
	optional($.generic_arguments),

),

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
func (p *Parser) nodeToDef(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Def {
	//fmt.Println(node)
	defBuilder := idx.NewDefBuilder("", currentModule.GetModuleString(), *docId).
		WithDocumentRange(
			uint(node.StartPoint().Row),
			uint(node.StartPoint().Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident":
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierRange(
					uint(n.StartPoint().Row),
					uint(n.StartPoint().Column),
					uint(n.EndPoint().Row),
					uint(n.EndPoint().Column),
				)

		case "define_ident":
			nameNode := n.Child(0)
			resolvesTo := n.Child(2).Content(sourceCode)

			if n.ChildCount() >= 4 {
				// Also include applied generic arguments
				resolvesTo += n.Child(3).Content(sourceCode)
			}

			defBuilder.WithName(nameNode.Content(sourceCode)).
				WithIdentifierRange(
					uint(nameNode.StartPoint().Row),
					uint(nameNode.StartPoint().Column),
					uint(nameNode.EndPoint().Row),
					uint(nameNode.EndPoint().Column),
				).
				WithResolvesTo(resolvesTo)

		case "typedef_type":
			var _type idx.Type
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = p.typeNodeToType(n.Child(0), currentModule, sourceCode)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	return *defBuilder.Build()
}
