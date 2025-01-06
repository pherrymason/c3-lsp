package factory

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func (c *ASTConverter) convert_ct_type_ident(node *sitter.Node, source []byte) ast.Expression {
	return &ast.BasicLit{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Kind:           ast.STRING,
		Value:          node.Content(source)}
}

/*
"$alignof",

	"$extnameof",
	"$nameof",
	"$offsetof",
	"$qnameof"
*/
func (c *ASTConverter) convert_compile_time_call(node *sitter.Node, source []byte) ast.Expression {
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

	funcCall := &ast.FunctionCall{
		NodeAttributes: ast.NewNodeAttributesBuilder(c.getNextID()).
			WithSitterStartEnd(node.StartPoint(), endNode.EndPoint()).
			Build(),
		Identifier: ast.NewIdentifierBuilder(c.getNextID()).
			WithName(node.Content(source)).
			WithSitterPos(node).
			BuildPtr(),
		Arguments: []ast.Expression{c.convert_flat_path(flatPath, source)},
	}

	return funcCall
}

func (c *ASTConverter) convert_compile_time_arg(node *sitter.Node, source []byte) ast.Expression {
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

	expr := c.convert_expression(insideParenths, source)

	funcCall := &ast.FunctionCall{
		NodeAttributes: ast.NewNodeAttributesBuilder(c.getNextID()).
			WithSitterStartEnd(node.StartPoint(), endNode.EndPoint()).
			Build(),
		Identifier: ast.NewIdentifierBuilder(c.getNextID()).
			WithName(node.Content(source)).
			WithSitterPos(node).
			BuildPtr(),
		Arguments: []ast.Expression{expr.(ast.Expression)},
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
func (c *ASTConverter) convert_compile_time_analyse(node *sitter.Node, source []byte) ast.Expression {
	decl_or_expr_node := node.NextNamedSibling()
	//fmt.Printf("cca: ")
	//debugNode(node, source)
	//fmt.Printf("\nnext: ")
	//debugNode(decl_or_expr_node, source)

	//expressions := convert_token_separated(decl_or_expr_node, ",", source, convert_decl_or_expr)

	funcCall := &ast.FunctionCall{
		NodeAttributes: ast.NewNodeAttributesBuilder(c.getNextID()).
			WithSitterStartEnd(node.StartPoint(), decl_or_expr_node.NextSibling().EndPoint()).
			Build(),
		Identifier: ast.NewIdentifierBuilder(c.getNextID()).
			WithName(node.Content(source)).
			WithSitterPos(node).
			BuildPtr(),
		Arguments: []ast.Expression{
			c.convert_expression(decl_or_expr_node, source).(ast.Expression),
		},
	}

	return funcCall
}

func cast_expressions_to_args(expressions []ast.Expression) []ast.Expression {
	var args []ast.Expression

	for _, expr := range expressions {
		if arg, ok := expr.(ast.Expression); ok {
			args = append(args, arg)
		} else {
			// Si algún elemento no puede convertirse, retornamos un error
			panic(fmt.Sprintf("no se pudo convertir %v a Arg", expr))
		}
	}

	return args
}

func (c *ASTConverter) convert_compile_time_call_unk(node *sitter.Node, source []byte) ast.Expression {
	next := node.NextNamedSibling()
	_args := c.convert_token_separated(next, ",", source, cv_expr_fn(c.convert_expression))
	var args []ast.Expression
	for _, a := range _args {
		args = append(args, a.(ast.Expression))
	}

	return &ast.FunctionCall{
		NodeAttributes: ast.NewNodeAttributesBuilder(c.getNextID()).WithSitterStartEnd(node.StartPoint(), next.EndPoint()).Build(),
		Identifier: ast.NewIdentifierBuilder(c.getNextID()).
			WithName(node.Content(source)).
			WithSitterPos(node).
			BuildPtr(),
		Arguments: cast_expressions_to_args(args),
	}
}

func (c *ASTConverter) convert_feature(node *sitter.Node, source []byte) ast.Expression {
	next := node.NextNamedSibling()
	return &ast.FunctionCall{
		NodeAttributes: ast.NewNodeAttributesBuilder(c.getNextID()).WithSitterStartEnd(node.StartPoint(), next.EndPoint()).Build(),
		Identifier: ast.NewIdentifierBuilder(c.getNextID()).
			WithName(node.Content(source)).
			WithSitterPos(node).
			BuildPtr(),
		Arguments: []ast.Expression{c.convert_base_expression(next, source)},
	}
}
