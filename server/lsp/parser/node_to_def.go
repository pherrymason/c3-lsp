package parser

import (
	"strings"

	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
define_declaration: $ => seq(

	  'def',
	  choice(
	    $.define_ident,				// TODO
	    $.define_attribute,			// TODO
	    seq(
	      $.type_ident,
	      optional($.attributes),	// TODO
	      '=',
	      $.typedef_type,
	    ),
	  ),
	  optional($.attributes),		// TODO
	  ';'
	),
*/
func (p *Parser) nodeToDef(node *sitter.Node, moduleName string, docId string, sourceCode []byte) idx.Def {
	//definition := ""
	//var identifierNode *sitter.Node
	//fmt.Println(node)
	defBuilder := idx.NewDefBuilder("", moduleName, docId).
		WithDocumentRange(
			uint(node.StartPoint().Row),
			uint(node.StartPoint().Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type_ident", "define_ident":
			//identifierNode = n
			defBuilder.WithName(n.Content(sourceCode)).
				WithIdentifierRange(
					uint(n.StartPoint().Row),
					uint(n.StartPoint().Column),
					uint(n.EndPoint().Row),
					uint(n.EndPoint().Column),
				)

		case "typedef_type":
			var _type idx.Type
			if n.Child(0).Type() == "type" {
				// Might contain module path
				_type = p.typeNodeToType(n.Child(0), moduleName, sourceCode)
				defBuilder.WithResolvesToType(_type)
			} else if n.Child(0).Type() == "func_typedef" {
				defBuilder.WithResolvesTo(n.Content(sourceCode))
			}
		}
	}

	return *defBuilder.Build()
}

func (p *Parser) typeNodeToType(node *sitter.Node, moduleName string, sourceCode []byte) idx.Type {
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

	return idx.NewType(baseTypeLanguage, baseType, pointerCount, modulePath)
}
