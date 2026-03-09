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
func (p *Parser) nodeToFaultDef(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.FaultDef {
	attributes := parseNodeAttributes(node, sourceCode)

	baseType := ""
	module := currentModule.GetModuleString()
	var constants []*idx.FaultConstant

	idRange := idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node), startPointSkippingDocComment(node))
	for i := 0; i < int(node.ChildCount()); i++ {
		constantNode := node.Child(i)

		if constantNode.Type() == "const_ident" {
			constantRange := idx.NewRangeFromTreeSitterPositions(constantNode.StartPoint(), constantNode.EndPoint())
			if len(constants) == 0 {
				idRange = constantRange
			}
			constants = append(constants,
				idx.NewFaultConstant(
					constantNode.Content(sourceCode),
					"",
					module,
					*docId,
					constantRange,
					constantRange,
				),
			)
		}
	}

	faultDef := idx.NewFaultDef(
		"",
		baseType,
		constants,
		module,
		*docId,
		idRange,
		idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node), node.EndPoint()),
	)
	faultDef.SetAttributes(attributes)

	return faultDef
}
