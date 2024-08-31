package parser

import (
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
	module: $ => seq(
	'module',
	field('path', $.path_ident),
	optional(alias($.generic_module_parameters, $.generic_parameters)),
	optional($.attributes),
	';'

	attributes:
		@private

),
*/
func (p *Parser) nodeToModule(doc *document.Document, node *sitter.Node, sourceCode []byte) (*symbols.Module, string, map[string]*symbols.GenericParameter) {

	moduleName := node.ChildByFieldName("path").Content(sourceCode)

	generic_parameters := make(map[string]*symbols.GenericParameter)
	attributes := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println("Node type:", n.Type(), ":: ", n.Content(sourceCode))
		switch n.Type() {
		case "generic_parameters", "generic_module_parameters":
			for g := 0; g < int(n.ChildCount()); g++ {
				gn := n.Child(g)
				//fmt.Println("G Node type:", gn.Type(), ":: ", gn.Content(sourceCode))
				if gn.Type() == "type_ident" {
					genericName := gn.Content(sourceCode)
					param := symbols.NewGenericParameter(
						genericName,
						moduleName,
						doc.URI,
						symbols.NewRangeFromTreeSitterPositions(gn.StartPoint(), gn.EndPoint()),
						symbols.NewRangeFromTreeSitterPositions(gn.StartPoint(), gn.EndPoint()),
					)
					generic_parameters[genericName] = param
				}
			}
		case "attributes":
			for a := 0; a < int(n.ChildCount()); a++ {
				gn := n.Child(a)
				//fmt.Println("Attr Node type:", gn.Type(), ":: ", gn.Content(sourceCode))
				attributes = append(attributes, gn.Content(sourceCode))
			}
		}
	}

	name := node.ChildByFieldName("path")
	module := symbols.NewModule(
		moduleName,
		doc.URI,
		symbols.NewRangeFromTreeSitterPositions(name.StartPoint(), name.EndPoint()),
		symbols.NewRangeFromTreeSitterPositions(name.StartPoint(), name.EndPoint()),
	)
	module.SetAttributes(attributes)
	module.SetGenericParameters(generic_parameters)

	return module, moduleName, generic_parameters
}

/*
		import_declaration: $ => seq(
	      'import',
	      field('path', commaSep1($.path_ident)),
	      optional($.attributes),
	      ';'
	    ),
*/
func (p *Parser) nodeToImport(doc *document.Document, node *sitter.Node, sourceCode []byte) []string {
	imports := []string{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

		switch n.Type() {
		case "path_ident":
			temp_mod := ""
			for m := 0; m < int(n.ChildCount()); m++ {
				sn := n.Child(m)
				if sn.Type() == "ident" || sn.Type() == "module_resolution" {
					temp_mod += sn.Content(sourceCode)
				}
			}
			imports = append(imports, temp_mod)
		}
	}

	return imports
}
