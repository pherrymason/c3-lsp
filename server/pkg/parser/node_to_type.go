package parser

import (
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) typeNodeToType(node *sitter.Node, currentModule *symbols.Module, sourceCode []byte) symbols.Type {

	isOptional := false
	baseTypeLanguage := false
	baseType := ""
	modulePath := currentModule.GetModuleString()
	generic_arguments := []symbols.Type{}

	parsedType := symbols.Type{}

	tailChild := node.Child(int(node.ChildCount()) - 1)
	isOptional := !tailChild.IsNamed() && tailChild.Content(sourceCode) == "!"

	//fmt.Println(node.Type(), node.Content(sourceCode), node.ChildCount())
	isCollection := false
	collectionSize := option.None[int]()
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
							gType := p.typeNodeToType(gn, currentModule, sourceCode)
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
			} else if suffix[0] == '[' {
				isCollection = true
				if n.ChildCount() > 2 && n.Child(1).Type() == "integer_literal" {
					sizeStr := n.Child(1).Content(sourceCode)
					i, err := strconv.Atoi(sizeStr)
					if err == nil {
						collectionSize = option.Some(i)
					}
				}
			}
		case "!":
			isOptional = true
		}
	}

	// Is baseType a module generic argument? Flag it.
	isGenericArgument := false
	for genericId, _ := range currentModule.GenericParameters {
		if genericId == baseType {
			isGenericArgument = true
		}
	}

	//var parsedType symbols.Type
	if len(generic_arguments) == 0 {
		if isOptional {
			parsedType = symbols.NewOptionalType(baseTypeLanguage, baseType, pointerCount, isGenericArgument, isCollection, collectionSize, modulePath)
		} else {
			parsedType = symbols.NewType(baseTypeLanguage, baseType, pointerCount, isGenericArgument, isCollection, collectionSize, modulePath)
		}
	} else {
		// TODO Can a type with generic be itself a generic argument?
		parsedType = symbols.NewTypeWithGeneric(baseTypeLanguage, isOptional, baseType, pointerCount, generic_arguments, modulePath)
	}

	return parsedType
}
