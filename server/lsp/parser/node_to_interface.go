package parser

import (
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
interface_declaration: $ => seq(

	  'interface',
	  field('name', $.type_ident),
	  field('body', $.interface_body),
	),
*/
func (p *Parser) nodeToInterface(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Interface {
	// TODO parse attributes
	methods := []idx.Function{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "interface_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				m := n.Child(i)
				if m.Type() == "func_declaration" {
					methods = append(methods, p.nodeToFunction(m, moduleName, docId, sourceCode))
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	_interface := idx.NewInterface(
		nameNode.Content(sourceCode),
		moduleName,
		docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(node.StartPoint(), node.EndPoint()),
	)

	_interface.AddMethods(methods)

	return _interface
}
