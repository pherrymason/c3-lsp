package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
interface_declaration: $ => seq(

	  'interface',
	  field('name', $.type_ident),
	  field('body', $.interface_body),
	),
*/
func (p *Parser) nodeToInterface(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Interface {
	// TODO parse attributes
	methods := []*idx.Function{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "interface_body":
			for i := 0; i < int(n.ChildCount()); i++ {
				m := n.Child(i)
				if m.Type() == "func_declaration" {
					fun, err := p.nodeToFunction(m, currentModule, docId, sourceCode)
					if err == nil {
						methods = append(methods, &fun)
					}
					continue
				}

				if m.Type() == "interface_func_declaration" {
					for j := 0; j < int(m.ChildCount()); j++ {
						inner := m.Child(j)
						if inner.Type() != "func_declaration" {
							continue
						}
						fun, err := p.nodeToFunction(inner, currentModule, docId, sourceCode)
						if err == nil {
							methods = append(methods, &fun)
						}
					}
				}
			}
		}
	}

	nameNode := node.ChildByFieldName("name")
	_interface := idx.NewInterface(
		nameNode.Content(sourceCode),
		currentModule.GetModuleString(),
		*docId,
		idx.NewRangeFromTreeSitterPositions(nameNode.StartPoint(), nameNode.EndPoint()),
		idx.NewRangeFromTreeSitterPositions(startPointSkippingDocComment(node), node.EndPoint()),
	)

	_interface.AddMethods(methods)

	return _interface
}
