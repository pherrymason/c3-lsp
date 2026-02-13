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

	idRange := idx.NewRange(0, 0, 0, 0)
	for i := 0; i < int(node.ChildCount()); i++ {
		constantNode := node.Child(i)

		if constantNode.Type() == "const_ident" {
			constants = append(constants,
				idx.NewFaultConstant(
					constantNode.Content(sourceCode),
					"",
					module,
					*docId,
					idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
					idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint()),
				),
			)
		}
	}

	fault := idx.NewFault(
		"",
		baseType,
		constants,
		module,
		*docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node), node.EndPoint()),
	)

	return fault
}
