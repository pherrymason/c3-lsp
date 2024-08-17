package ast

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
)

/*
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
	//fmt.Print("convert_expression:\n")
	//debugNode(node, source)
	return anyOf([]NodeRule{
		NodeOfType("assignment_expr"),
		NodeOfType("ternary_expr"),
		NodeSequenceOf([]NodeRule{
			NodeOfType("lambda_declaration"),
			NodeOfType("implies_body"),
		}, "lambda_expr"),
		NodeOfType("elvis_orelse_expr"),
		NodeOfType("optional_expr"),
		NodeOfType("binary_expr"),
		NodeOfType("unary_expr"),
		NodeOfType("cast_expr"),
		NodeOfType("rethrow_expr"),
		NodeOfType("trailing_generic_expr"),
		NodeOfType("update_expr"),
		NodeOfType("call_expr"),
		NodeOfType("subscript_expr"),
		NodeOfType("initializer_list"),
		NodeTryConversionFunc("_base_expr"),
	}, node, source)
	/*
	   switch node.Type() {
	   case "assignment_expr":

	   	return convert_assignment_expr(node, source)

	   case "ternary_expr":

	   		return convert_ternary_expr(node, source)
	   	}

	   base_expr := convert_base_expression(node, source)

	   	if base_expr != nil {
	   		return base_expr
	   	}

	   return nil
	*/
}

func convert_assignment_expr(node *sitter.Node, source []byte) Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")
	var left Expression
	var right Expression
	operator := ""
	if leftNode.Type() == "ct_type_ident" {
		left = convert_ct_type_ident(leftNode, source)
		right = convert_type(rightNode, source)
		operator = "="
	} else {
		left = convert_expression(leftNode, source)
		right = convert_expression(rightNode, source)
		operator = node.ChildByFieldName("operator").Content(source)
	}

	return AssignmentStatement{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Left:        left,
		Right:       right,
		Operator:    operator,
	}
}

func convert_binary_expr(node *sitter.Node, source []byte) Expression {
	left := convert_expression(node.ChildByFieldName("left"), source)
	operator := node.ChildByFieldName("operator").Content(source)
	right := convert_expression(node.ChildByFieldName("right"), source)

	return BinaryExpr{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Left:        left,
		Operator:    operator,
		Right:       right,
	}
}

func convert_ternary_expr(node *sitter.Node, source []byte) Expression {
	expected := []NodeRule{
		NodeOfType("binary_expr"),
		NodeOfType("unary_expr"),
		NodeOfType("cast_expr"),
		NodeOfType("rethrow_expr"),
		NodeOfType("trailing_generic_expr"),
		NodeOfType("update_expr"),
		NodeOfType("call_expr"),
		NodeOfType("subscript_expr"),
		NodeOfType("initializer_list"),
		NodeOfType("_base_expr"),
	}
	condition := anyOf(expected, node.ChildByFieldName("condition"), source)

	return TernaryExpression{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Condition:   condition,
		Consequence: convert_expression(node.ChildByFieldName("consequence"), source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_elvis_orelse_expr(node *sitter.Node, source []byte) Expression {
	conditionNode := node.ChildByFieldName("condition")

	return TernaryExpression{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Condition:   convert_expression(conditionNode, source),
		Consequence: convert_ident(conditionNode, source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_optional_expr(node *sitter.Node, source []byte) Expression {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(source)
	if operatorNode.NextSibling().Type() == "!" {
		operator += "!"
	}

	argumentNode := node.ChildByFieldName("argument")
	return OptionalExpression{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		Operator:    operator,
		Argument:    convert_expression(argumentNode, source),
	}
}

func convert_base_expression(node *sitter.Node, source []byte) Expression {
	var expression Expression
	//fmt.Printf("converting_base_expression\n")
	//debugNode(node, source)
	nodeType := node.Type()
	if is_literal(node) {
		expression = convert_literal(node, source)
	} else {
		switch nodeType {
		case "ident", "ct_ident", "hash_ident", "const_ident", "at_ident":
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

		case "type":
			baseExpr := convert_base_expression(node.NextNamedSibling(), source)
			initList, ok := baseExpr.(InitializerList)
			if !ok {
				initList = InitializerList{}
			}

			expression = InlineTypeWithInitizlization{
				ASTNodeBase: NewBaseNodeBuilder().
					WithStartEnd(
						uint(node.StartPoint().Row),
						uint(node.StartPoint().Column),
						initList.ASTNodeBase.EndPos.Line,
						initList.ASTNodeBase.EndPos.Column,
					).Build(),
				Type:            convert_type(node, source),
				InitializerList: initList,
			}

		case "field_expr":
		case "type_access_expr":
		case "paren_expr":
		case "expr_block":

		case "$vacount": // literally $vacount
			expression = Literal{Value: "$vacount"}

		// Compile time calls
		// _ct_call
		case "$alignof",
			"$extnameof",
			"$nameof",
			"$offsetof",
			"$qnameof":
			expression = convert_compile_time_call(node, source)

		// _ct_arg
		case "$vaconst",
			"$vaarg",
			"$varef",
			"$vaexpr":
			expression = convert_compile_time_arg(node, source)

		// _ct_analyse
		case "$eval",
			"$is_const",
			"$sizeof",
			"$stringify":
			expression = convert_compile_time_analyse(node, source)

		case "$feature":
			next := node.NextNamedSibling()
			expression = FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(node.StartPoint(), next.EndPoint()).Build(),
				Identifier: NewIdentifierBuilder().
					WithName(node.Content(source)).
					WithSitterPos(node).
					Build(),
				Arguments: []Arg{convert_base_expression(next, source)},
			}

		case "$and",
			"$append",
			"$concat",
			"$defined",
			"$embed",
			"$or":
			next := node.NextNamedSibling()

			expression = FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(node.StartPoint(), next.EndPoint()).Build(),
				Identifier: NewIdentifierBuilder().
					WithName(node.Content(source)).
					WithSitterPos(node).
					Build(),
				Arguments: cast_expressions_to_args(
					convert_token_separated(next, ",", source, convert_expression),
				),
			}

		case "$assignable":
			// TODO

		case "lambda_declaration":
			expression = convert_lambda_declaration(node, source)

			lambda := expression.(LambdaDeclaration)
			lambda.Body = convert_compound_stmt(node.NextSibling(), source).(CompoundStatement)

			expression = lambda

			// Sequences
			/*
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
	//fmt.Printf("Converting literal %s\n", node.Type())
	switch node.Type() {
	case "string_literal", "char_literal", "raw_string_literal", "bytes_literal":
		literal = Literal{Value: node.Content(sourceCode)}
	case "integer_literal":
		literal = IntegerLiteral{Value: node.Content(sourceCode)}
	case "real_literal":
		literal = RealLiteral{Value: node.Content(sourceCode)}
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
	childCount := int(node.ChildCount())

	if is_literal(node.Child(0)) {
		return convert_literal(node.Child(0), source)
	}

	switch node.Child(0).Type() {
	case "param_path":
		param_path := node.Child(0)
		var arg Arg
		param_path_element := param_path.Child(0)

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
}

const (
	PathIdent = iota
	PathField
)

func convert_param_path(param_path *sitter.Node, source []byte) Path {
	var path Path
	param_path_element := param_path.Child(0)

	pathType := PathTypeIndexed
	for p := 0; p < int(param_path_element.ChildCount()); p++ {
		pnode := param_path_element.Child(p)
		if pnode.IsNamed() {
			if pnode.Type() == "ident" {
				pathType = PathTypeField
			}
		} else if pnode.Type() == ".." {
			pathType = PathTypeRange
		}
	}

	path = Path{
		PathType: pathType,
	}
	if pathType == PathTypeField {
		path.FieldName = param_path_element.Child(1).Content(source)
	} else if pathType == PathTypeRange {
		path.PathStart = param_path_element.Child(1).Content(source)
		path.PathEnd = param_path_element.Child(3).Content(source)

	} else {
		path.Path = param_path.Child(0).Content(source)
	}

	return path
}

func convert_flat_path(node *sitter.Node, source []byte) Expression {
	node = node.Child(0)

	if node.Type() == "type" {
		return convert_type(node, source)
	}

	base_expr := convert_base_expression(node, source)

	next := node.NextSibling()
	if next != nil {
		// base_expr + param_path
		//base_expr := convert_base_expression(node, source)
		//param_path := convert_param_path(node.NextSibling(), source)
		path := convert_param_path(next, source)
		switch path.PathType {
		case PathTypeIndexed:
			return IndexAccess{
				Array: base_expr,
				Index: path.Path,
			}
		case PathTypeField:
			return FieldAccess{
				Object: base_expr,
				Field:  path,
			}
		case PathTypeRange:
			return RangeAccess{
				Array:      base_expr,
				RangeStart: utils.StringToUint(path.PathStart),
				RangeEnd:   utils.StringToUint(path.PathEnd),
			}
		}
	}

	return base_expr
}

func cast_expressions_to_args(expressions []Expression) []Arg {
	var args []Arg

	for _, expr := range expressions {
		// Realiza una conversión de tipo de Expression a Arg
		if arg, ok := expr.(Arg); ok {
			args = append(args, arg)
		} else {
			// Si algún elemento no puede convertirse, retornamos un error
			panic(fmt.Sprintf("no se pudo convertir %v a Arg", expr))
		}
	}

	return args
}

type NodeConverterSeparated func(node *sitter.Node, source []byte) (Expression, int)

func convert_token_separated(node *sitter.Node, separator string, source []byte, convert_func NodeConverter) []Expression {
	expressions := []Expression{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == separator {
			continue
		}
		expr := convert_func(n, source)

		if expr != nil {
			expressions = append(expressions, expr)
		}
		//i += advance
	}

	return expressions
}

/*
$.compound_stmt,
$.expr_stmt,
$.declaration_stmt,
$.var_stmt,
$.return_stmt,
$.continue_stmt,
$.break_stmt,
$.switch_stmt,
$.nextcase_stmt,
$.if_stmt,
$.for_stmt,
$.foreach_stmt,
$.while_stmt,
$.do_stmt,
$.defer_stmt,
$.assert_stmt,
$.asm_block_stmt,

$.ct_echo_stmt,
$.ct_assert_stmt,
$.ct_if_stmt,
$.ct_switch_stmt,
$.ct_foreach_stmt,
$.ct_for_stmt,
*/
func convert_compound_stmt(node *sitter.Node, source []byte) Expression {
	cmpStatement := CompoundStatement{
		Statements: []Expression{},
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() != "{" && n.Type() != "}" {
			cmpStatement.Statements = append(
				cmpStatement.Statements,
				convert_statement(n, source),
			)
		}
	}

	return cmpStatement
}

func convert_statement(node *sitter.Node, source []byte) Expression {

	switch node.Type() {
	case "compound_stmt":
		return convert_compound_stmt(node, source)

	case "expr_stmt":
		return convert_expression(node.Child(0), source)
	}

	return nil
}

func convert_dummy(node *sitter.Node, source []byte) Expression {
	return nil
}

func debugNode(node *sitter.Node, source []byte) {
	if node == nil {
		fmt.Printf("Node is nil\n")
		return
	}

	fmt.Printf("%s: %s\n----- %s\n", node.Type(), node.Content(source), node)
}
