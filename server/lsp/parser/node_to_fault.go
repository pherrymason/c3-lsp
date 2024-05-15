package parser

import (
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
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
func (p *Parser) nodeToFault(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Fault {
	// TODO parse attributes

	baseType := "" // TODO Parse type!
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
							idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
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
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	return enum
}
