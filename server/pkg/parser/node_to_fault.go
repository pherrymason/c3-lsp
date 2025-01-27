package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
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
	commaSepTrailing($.const_ident),
	'}'

),
*/
func (p *Parser) nodeToFault(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Fault {
	// TODO parse attributes

	baseType := "" // TODO Parse type!
	module := currentModule.GetModuleString()
	var constants []*idx.FaultConstant

	nameNode := node.ChildByFieldName("name")
	name := ""
	idRange := idx.NewRange(0, 0, 0, 0)
	if nameNode != nil {
		name = nameNode.Content(sourceCode)
		idRange = idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint())
	}

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
							name,
							module,
							*docId,
							idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
							idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
						),
					)
				}
			}
		}
	}

	fault := idx.NewFault(
		name,
		baseType,
		constants,
		module,
		*docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	return fault
}
