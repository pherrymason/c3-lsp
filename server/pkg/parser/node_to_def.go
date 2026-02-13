package parser

import (
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
alias_declaration: $ => seq(

	'alias',
	choice(
	  // Variable/function/macro/method/module
	  seq(
	    field('name', $._func_macro_ident),
	    optional($.attributes),
	    choice(
	      seq('=', 'module', $.path_ident),
	      $._assign_right_expr,
	    )
	  ),
	  // Constant
	  seq(
	    field('name', $.const_ident),
	    optional($.attributes),
	    $._assign_right_expr,
	  ),
	  // Type/function
	  seq(
	    field('name', $.type_ident),
	    optional($.attributes),
	    '=',
	    choice($._type_expr, $.func_signature)
	  ),
	),
	';'

),
*/
func (p *Parser) nodeToDef(node *sitter.Node, currentModule *idx.Module, docId *string, sourceCode []byte) idx.Def {
	//fmt.Println(node)
	// TODO: attributes
	start := startPointSkippingDocComment(node)
	defBuilder := idx.NewDefBuilder("", currentModule.GetModuleString(), *docId).
		WithDocumentRange(
			uint(start.Row),
			uint(start.Column),
			uint(node.EndPoint().Row),
			uint(node.EndPoint().Column),
		)
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		defBuilder.WithName(nameNode.Content(sourceCode)).
			WithIdentifierRange(
				uint(nameNode.StartPoint().Row),
				uint(nameNode.StartPoint().Column),
				uint(nameNode.EndPoint().Row),
				uint(nameNode.EndPoint().Column),
			)
	}
	var bodyNode = &sitter.Node{}
	for i := 0; i < int(node.ChildCount()-1); i++ {
		if node.Child(i).Type() == "=" {
			bodyNode = node.Child(i + 1)
			break
		}
	}
	if bodyNode.Type() == "type" {
		// Might contain module path
		type_ := p.typeNodeToType(bodyNode, currentModule, sourceCode)
		defBuilder.WithResolvesToType(type_)
	} else {
		defBuilder.WithResolvesTo(bodyNode.Content(sourceCode))
	}

	return *defBuilder.Build()
}
