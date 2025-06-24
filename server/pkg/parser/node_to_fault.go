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
func (p *Parser) nodeToFault(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) []idx.Fault {
	// TODO parse attributes
	module := currentModule.GetModuleString()
	var faults []idx.Fault

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "const_ident":
			faults = append(faults,
				idx.NewFault(
					n.Content(sourceCode),
					module,
					*docId,
					idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
					idx.NewRangeFromTreeSitterPositions(n.StartPoint(), n.EndPoint()),
				),
			)
		}
	}
	/*
		fault := idx.NewFault(
			"",
			baseType,
			constants,
			module,
			*docId,
			idRange,
			idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
		)*/

	return faults
}
