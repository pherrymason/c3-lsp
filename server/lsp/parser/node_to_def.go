package parser

import (
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
func (p *Parser) nodeToDef(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Def {
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
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(identifierNode.StartPoint(), identifierNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)
}
