package parser

import (
	"strconv"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

func (p *Parser) typeNodeToType(node *sitter.Node, currentModule *symbols.Module, sourceCode []byte) symbols.Type {
	//fmt.Println(node, node.Content(sourceCode))
	baseTypeLanguage := false
	baseType := ""
	modulePath := currentModule.GetModuleString()
	generic_arguments := []symbols.Type{}

	parsedType := symbols.Type{}

	tailChild := node.Child(int(node.ChildCount()) - 1)
	isOptional := !tailChild.IsNamed() && tailChild.Content(sourceCode) == "?"

	//fmt.Println(node.Type(), node.Content(sourceCode), node.ChildCount())
	isCollection := false
	collectionSize := option.None[int]()
	pointerCount := 0

	if node.Type() == "base_type_name" {
		baseTypeLanguage = true
		baseType = node.Content(sourceCode)
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		// fmt.Println(node.Type()+"---"+n.Type(), n.Content(sourceCode))
		switch n.Type() {
		case "base_type_name":
			baseTypeLanguage = true
			baseType = n.Content(sourceCode)
		case "type_ident":
			baseType = n.Content(sourceCode)

		case "generic_type_ident":
			if n.ChildCount() >= 1 {
				parse_path_type_ident(n.Child(0), sourceCode, &modulePath, &baseType)
			}
			if n.ChildCount() >= 2 {
				genericArgList := n.Child(1)
				for g := 0; g < int(genericArgList.ChildCount()); g++ {
					gn := genericArgList.Child(g)
					if gn.Type() == "type" {
						gType := p.typeNodeToType(gn, currentModule, sourceCode)
						generic_arguments = append(generic_arguments, gType)
					}
				}
			}

		case "path_type_ident":
			parse_path_type_ident(n, sourceCode, &modulePath, &baseType)

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

func parse_path_type_ident(n *sitter.Node, sourceCode []byte, modulePath, baseType *string) {
	if n.ChildCount() == 2 {
		*modulePath = strings.Trim(n.Child(0).Content(sourceCode), ":")
		*baseType = n.Child(1).Content(sourceCode)
	} else if n.ChildCount() > 0 {
		*baseType = n.Child(0).Content(sourceCode)
	}
}
