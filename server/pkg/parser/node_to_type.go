package parser

import (
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) typeNodeToType(node *sitter.Node, moduleName string, sourceCode []byte) symbols.Type {

	if node.Type() == "optional_type" {
		return p.extTypeNodeToType(node.Child(0), true, moduleName, sourceCode)
	}

	return p.extTypeNodeToType(node, false, moduleName, sourceCode)
}

func (p *Parser) extTypeNodeToType(node *sitter.Node, isOptional bool, moduleName string, sourceCode []byte) symbols.Type {
	//fmt.Println(node, node.Content(sourceCode))
	baseTypeLanguage := false
	baseType := ""
	modulePath := moduleName
	generic_arguments := []symbols.Type{}

	//fmt.Println(node.Type(), node.Content(sourceCode), node.ChildCount())

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
					//baseType += bn.Content(sourceCode)
					for g := 0; g < int(bn.ChildCount()); g++ {
						gn := bn.Child(g)
						if gn.Type() == "type" {
							gType := p.typeNodeToType(gn, moduleName, sourceCode)
							generic_arguments = append(generic_arguments, gType)
						}
					}

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

	var parsedType symbols.Type
	if len(generic_arguments) == 0 {
		if isOptional {
			parsedType = symbols.NewOptionalType(baseTypeLanguage, baseType, pointerCount, modulePath)
		} else {
			parsedType = symbols.NewType(baseTypeLanguage, baseType, pointerCount, modulePath)
		}
	} else {
		parsedType = symbols.NewTypeWithGeneric(baseTypeLanguage, isOptional, baseType, pointerCount, generic_arguments, modulePath)
	}

	return parsedType
}
