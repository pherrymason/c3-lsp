package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
interface_declaration: $ => seq(

	  'interface',
	  field('name', $.type_ident),
	  field('body', $.interface_body),
	),
*/
func (p *Parser) nodeToInterface(doc *document.Document, node *sitter.Node, sourceCode []byte) idx.Interface {
	// TODO parse attributes
	methods := []idx.Function{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "interface_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				m := n.Child(i)
				if m.Type() == "func_declaration" {
					methods = append(methods, p.nodeToFunction(doc, m, sourceCode))
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	_interface := idx.NewInterface(
		nameNode.Content(sourceCode),
		doc.ModuleName,
		doc.URI,
		idx.NewRangeFromSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	_interface.AddMethods(methods)

	return _interface
}
