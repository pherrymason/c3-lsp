package ast

import (
	sitter "github.com/smacker/go-tree-sitter"
)

func convert_ct_type_ident(node *sitter.Node, source []byte) Expression {
	return Literal{Value: node.Content(source)}
}

/*
"$alignof",

	"$extnameof",
	"$nameof",
	"$offsetof",
	"$qnameof"
*/
func convert_compile_time_call(node *sitter.Node, source []byte) Expression {
	// seq($._ct_call, '(', $.flat_path, ')'),
	endNode := node.NextSibling()
	for {
		n := endNode.NextSibling()

		if n == nil {
			break
		}
		endNode = n
	}

	flatPath := node.NextNamedSibling()
	endNode = flatPath.NextSibling()

	funcCall := FunctionCall{
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), endNode.EndPoint()).
			Build(),
		Identifier: NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			Build(),
		Arguments: []Arg{convert_flat_path(flatPath, source)},
	}

	return funcCall
}

func convert_compile_time_arg(node *sitter.Node, source []byte) Expression {
	endNode := node
	var insideParenths *sitter.Node
	for {
		n := endNode.NextSibling()
		endNode = n
		if n.Type() != "(" && n.Type() != ")" {
			insideParenths = n
		}
		if n.Type() == ")" {
			break
		}
	}

	expr := convert_expression(insideParenths, source)

	funcCall := FunctionCall{
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), endNode.EndPoint()).
			Build(),
		Identifier: NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			Build(),
		Arguments: []Arg{expr},
	}

	return funcCall
}

/*
*

	seq($._ct_analyse, '(', $.comma_decl_or_expr, ')'),
	_ct_analyse: $ => choice(
		'$eval',
		'$defined',
		'$sizeof',
		'$stringify',
		'$is_const',
	)

	-- NOW --
	'$eval',
	'$is_const',
	'$sizeof',
	'$stringify',
	$._ct_arg:
		$vaconst',
		'$vaarg',
		'$varef',
		'$vaexpr',
*/
func convert_compile_time_analyse(node *sitter.Node, source []byte) Expression {
	decl_or_expr_node := node.NextNamedSibling()
	//fmt.Printf("cca: ")
	//debugNode(node, source)
	//fmt.Printf("\nnext: ")
	//debugNode(decl_or_expr_node, source)

	//expressions := convert_token_separated(decl_or_expr_node, ",", source, convert_decl_or_expr)

	funcCall := FunctionCall{
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), decl_or_expr_node.NextSibling().EndPoint()).
			Build(),
		Identifier: NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			Build(),
		Arguments: []Arg{
			convert_expression(decl_or_expr_node, source),
		},
	}

	return funcCall
}
