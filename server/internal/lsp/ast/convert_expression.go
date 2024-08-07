package ast

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

/*
d

	$.assignment_expr,
	$.ternary_expr,
	$.lambda_expr,
	$.elvis_orelse_expr,
	$.suffix_expr,
	$.binary_expr,
	$.unary_expr,
	$.cast_expr,
	$.rethrow_expr,
	$.trailing_generic_expr,
	$.update_expr,
	$.call_expr,
	$.subscript_expr,
	$.initializer_list,
	$._base_expr,
*/
func convert_expression(node *sitter.Node, source []byte) Expression {
	debugNode(node, source)
	fmt.Printf("================\n")

	base_expr := convert_base_expression(node, source)
	if base_expr != nil {
		return base_expr
	}

	return nil
}

func convert_base_expression(node *sitter.Node, source []byte) Expression {
	var expression Expression
	fmt.Printf("Converting expression %s\n", node.Type())
	nodeType := node.Type()
	if is_literal(node) {
		expression = convert_literal(node, source)
	} else {
		switch nodeType {
		case "ident", "ct_ident", "hash_ident":
			expression = NewIdentifierBuilder().WithName(node.Content(source)).WithSitterPos(node).Build()

		case "module_ident_expr":
			expression = NewIdentifierBuilder().
				WithName(node.ChildByFieldName("ident").Content(source)).
				WithPath(node.Child(0).Child(0).Content(source)).
				WithSitterPos(node).Build()

		case "bytes_expr":
			expression = convert_literal(node.Child(0), source)

		case "builtin":
			// TODO Improve, generate BuiltinLiteral
			expression = Literal{Value: node.Content(source)}

		case "unary_expr":
			expression = UnaryExpression{
				Operator:   node.Child(0).Content(source),
				Expression: convert_base_expression(node.ChildByFieldName("argument"), source),
			}

		case "initializer_list":
			initList := InitializerList{
				ASTNodeBase: NewBaseNodeBuilder().
					WithSitterPos(node).Build(),
			}
			for i := 0; i < int(node.ChildCount()); i++ {
				n := node.Child(i)
				if n.Type() == "arg" {
					initList.Args = append(initList.Args, convert_arg(n, source))
				}
			}
			expression = initList

		case "field_expr":
		case "type_access_expr":
		case "paren_expr":
		case "expr_block":

		case "$vacount": // literally $vacount

			// Sequences
			/*
				seq($.type, $.initializer_list),
				seq($._ct_call, '(', $.flat_path, ')'),
				seq($._ct_arg, '(', $._expr, ')'),
				seq($._ct_analyse, '(', $.comma_decl_or_expr, ')'),
				seq('$feature', '(', $.const_ident, ')'),
				seq('$and', '(', $.comma_decl_or_expr, ')'),
				seq('$or', '(', $.comma_decl_or_expr, ')'),
				seq('$assignable', '(', $._expr, ',', $.type, ')'),
				seq('$embed', '(', commaSep($._constant_expr), ')'),

				seq($.lambda_declaration, $.compound_stmt),
			*/
		}

	}

	return expression
}

func convert_literal(node *sitter.Node, sourceCode []byte) Expression {
	var literal Expression
	fmt.Printf("Converting literal %s\n", node.Type())
	switch node.Type() {
	case "string_literal", "char_literal", "raw_string_literal", "bytes_literal":
		literal = Literal{Value: node.Content(sourceCode)}
	case "integer_literal", "real_literal":
		literal = Literal{Value: node.Content(sourceCode)}
	case "false":
		literal = BoolLiteral{Value: false}
	case "true":
		literal = BoolLiteral{Value: true}
	case "null":
		literal = Literal{Value: "null"}
	default:
		panic(fmt.Sprintf("Literal type not supported: %s\n", node.Type()))
	}

	return literal
}

func convert_arg(node *sitter.Node, source []byte) Arg {
	debugNode(node, source)
	childCount := int(node.ChildCount())
	switch node.Child(0).Type() {
	case "param_path":
		param_path := node.Child(0)
		var arg Arg
		param_path_element := param_path.Child(0)
		debugNode(param_path_element.Child(0), source)

		argType := 0
		for p := 0; p < int(param_path_element.ChildCount()); p++ {
			pnode := param_path_element.Child(p)
			if pnode.IsNamed() {
				if pnode.Type() == "ident" {
					argType = 1
				} else {
					argType = 0
				}
			}
		}

		if argType == 1 {
			arg = ArgFieldSet{
				FieldName: param_path_element.Child(1).Content(source),
			}
		} else {
			arg = ArgParamPathSet{
				Path: node.Child(0).Content(source),
			}
		}

		for j := 1; j < childCount; j++ {
			fmt.Print("\t       ")
			n := node.Child(j)
			debugNode(n, source)
			var expr Expression
			if n.Type() == "type" {
				expr = convert_type(n, source)
			} else if n.Type() != "=" {
				expr = convert_expression(n, source)
			}

			switch v := arg.(type) {
			case ArgParamPathSet:
				v.Expr = expr
				arg = v
			case ArgFieldSet:
				v.Expr = expr
				arg = v
			}
		}
		return arg

	case "type":
		return Expression(convert_type(node.Child(0), source))
	case "$vasplat":
		return Literal{Value: node.Content(source)}
	case "...":
		return Expression(convert_expression(node.Child(1), source))
	}

	return nil
	/*
		for i := 0; i < childCount; i++ {
			fmt.Print("- ")
			debugNode(node.Child(i), source)
			nodeType := node.Type()

			// param_path + = expr|type
			if nodeType == "param_path" {
				arg = ArgParamPathSet{
					path: node.Child(i).Content(source),
				}

				for j := i + 1; j < childCount; i++ {
					fmt.Print("\t       ")
					n := node.Child(j)
					debugNode(n, source)
					if n.Type() != "=" {

					}
				}
			}
		}

		return arg*/
}

func debugNode(node *sitter.Node, source []byte) {
	fmt.Printf("%s: %s\n", node.Type(), node.Content(source))
}
