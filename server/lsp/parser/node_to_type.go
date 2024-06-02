package parser

import (
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) typeNodeToType(node *sitter.Node, moduleName string, sourceCode []byte) symbols.Type {
	//fmt.Println(node, node.Content(sourceCode))
	baseTypeLanguage := false
	baseType := ""
	modulePath := moduleName

	pointerCount := 0
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		//fmt.Println(n.Type(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type":
			for b := 0; b < int(n.ChildCount()); b++ {
				bn := n.Child(b)
				//fmt.Println("---"+bn.Type(), bn.Content(sourceCode))
				switch bn.Type() {
				case "base_type_name":
					baseTypeLanguage = true
					baseType = bn.Content(sourceCode)
				case "type_ident":
					baseType = bn.Content(sourceCode)
				case "generic_arguments":
					baseType += bn.Content(sourceCode)

				case "module_type_ident":
					//fmt.Println(bn)
					modulePath = strings.Trim(bn.Child(0).Content(sourceCode), ":")
					baseType = bn.Child(1).Content(sourceCode)
				}

			}

		case "type_suffix":
			suffix := n.Content(sourceCode)
			if suffix == "*" {
				pointerCount = 1
			}
		}
	}

	return symbols.NewType(baseTypeLanguage, baseType, pointerCount, modulePath)
}
