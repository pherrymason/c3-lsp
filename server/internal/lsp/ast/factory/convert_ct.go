package factory

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

// Reference: https://c3-lang.org/generic-programming/reflection/#compile-time-functions

func (c *ASTConverter) convert_ct_type_ident(node *sitter.Node, source []byte) ast.Expression {
	return &ast.BasicLit{
		NodeAttributes: ast.NewAttrNodeFromSitterNode(c.getNextID(), node),
		Kind:           ast.STRING,
		Value:          node.Content(source)}
}

// convert_compile_time_call_expr
// Converts C3 builtin calls into an ast.CallExpr
func (c *ASTConverter) convert_compile_time_call_expr(node *sitter.Node, source []byte) ast.Expression {
	var endNode *sitter.Node
	n := node
	Lparen := 0
	Rparen := 0
	arguments := []ast.Expression{}
	for {
		n = n.NextSibling()
		if n == nil {
			break
		}

		debugNode(n, source, "ct")
		if n.Type() == "(" {
			Lparen = int(n.StartPoint().Column)
		} else if n.Type() == ")" {
			Rparen = int(n.StartPoint().Column)
		} else if n.Type() == "flat_path" {
			expr := c.convert_flat_path(n, source)
			arguments = append(arguments, expr)
		} else if n.Type() != ";" && n.Type() != "," {
			expr := c.convert_expression(n, source)
			arguments = append(arguments, expr)
		}
		endNode = n
	}

	callExpr := &ast.CallExpr{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterStartEnd(node.StartPoint(), endNode.EndPoint()).
			Build(),
		CompileTime: true,
		Identifier: ast.NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			Build(),
		Lparen:    uint(Lparen),
		Arguments: arguments,
		Rparen:    uint(Rparen),
	}

	return callExpr
}

func (c *ASTConverter) convert_compile_time_arg(node *sitter.Node, source []byte) ast.Expression {
	endNode := node
	var insideParenths *sitter.Node
	excluded := map[string]bool{"[": true, "]": true, "(": true, ")": true}
	for {
		n := endNode.NextSibling()
		endNode = n
		if !excluded[n.Type()] {
			insideParenths = n
			break
		}
	}

	expr := c.convert_expression(insideParenths, source)

	funcCall := &ast.SubscriptExpression{
		NodeAttributes: ast.NewNodeAttributesBuilder().
			WithSitterStartEnd(node.StartPoint(), endNode.EndPoint()).
			Build(),
		Argument: ast.NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			IsCompileTime(true).
			Build(),
		Index: expr,
	}

	return funcCall
}
