package parser

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
fault_declaration: $ => seq(

	'fault',
	field('name', $.type_ident),
	optional($.interface_impl),
	optional($.attributes),
	field('body', $.fault_body),

),

fault_body: $ => seq(

	'{',
	commaSepTrailing1($.const_ident),
	'}'

),
*/
func (p *Parser) nodeToFault(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Fault {
	// TODO parse attributes

	baseType := ""
	var constants []idx.FaultConstant

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "fault_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				constantNode := n.Child(i)

				if constantNode.Type() == "const_ident" {
					constants = append(constants,
						idx.NewFaultConstant(
							constantNode.Content(sourceCode),
							idx.NewRangeFromSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
						),
					)
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	enum := idx.NewFault(
		nameNode.Content(sourceCode),
		baseType,
		constants,
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	return enum
}
