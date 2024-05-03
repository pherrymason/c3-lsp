package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
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
func (p *Parser) nodeToDef(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Def {
	definition := ""
	var identifierNode *sitter.Node

	for i := uint32(0); i < (node.ChildCount() - 1); i++ {
		n := node.Child(int(i))
		switch n.Type() {
		case "define_ident":
			identifierNode = n

		case "type_ident":
			identifierNode = n

		case "typedef_type":
			definition = n.Content(sourceCode)
		}
	}

	return idx.NewDef(
		identifierNode.Content(sourceCode),
		definition,
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
	)
}
