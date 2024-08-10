package ast

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
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
	base_expr := convert_base_expression(node, source)
	if base_expr != nil {
		return base_expr
	}

	return nil
}

func convert_base_expression(node *sitter.Node, source []byte) Expression {
	var expression Expression
	//fmt.Printf("Converting expression %s\n", node.Type())
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
			"$defined",
			"$sizeof",
			"$stringify",
			"$is_const":
			expression = convert_compile_analyse(node, source)

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

		case "$and", "$or":
			next := node.NextNamedSibling()
			debugNode(next, source)
			expression = FunctionCall{
				ASTNodeBase: NewBaseNodeBuilder().WithSitterPosRange(node.StartPoint(), next.EndPoint()).Build(),
				Identifier: NewIdentifierBuilder().
					WithName(node.Content(source)).
					WithSitterPos(node).
					Build(),
				Arguments: cast_expressions_to_args(
					convert_token_separated(next, ",", source, convert_decl_or_expr),
				),
			}

		case "$assignable":
			// TODO

		case "$embed":
			// TODO

		case "lambda_declaration":
			expression = convert_lambda_declaration(node, source)

			lambda := expression.(LambdaDeclaration)
			lambda.Body = Block{
				Statements: convert_compound_stmt(node.NextSibling(), source),
			}
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

func convert_compile_analyse(node *sitter.Node, source []byte) Expression {
	// comma_decl_or_expr
	debugNode(node, source)
	decl_or_expr_node := node.NextNamedSibling()
	debugNode(decl_or_expr_node, source)

	expressions := convert_token_separated(decl_or_expr_node, ",", source, convert_decl_or_expr)

	funcCall := FunctionCall{
		ASTNodeBase: NewBaseNodeBuilder().
			WithSitterPosRange(node.StartPoint(), decl_or_expr_node.NextSibling().EndPoint()).
			Build(),
		Identifier: NewIdentifierBuilder().
			WithName(node.Content(source)).
			WithSitterPos(node).
			Build(),
		Arguments: cast_expressions_to_args(expressions),
	}

	return funcCall
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

func convert_token_separated(node *sitter.Node, separator string, source []byte, convert_func NodeConverterSeparated) []Expression {
	expressions := []Expression{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		debugNode(n, source)
		if n.Type() == separator {
			continue
		}
		expr, advance := convert_func(n, source)

		if expr != nil {
			expressions = append(expressions, expr)
		}
		i += advance
	}

	return expressions
}

func convert_decl_or_expr(node *sitter.Node, source []byte) (Expression, int) {
	debugNode(node, source)
	if node.Type() == "var_decl" {
		// TODO
	} else if node.Type() == "type" || node.Type() == "optional_type" {
		typeInfo := convert_type(node, source)
		if node.NextSibling().Type() == "local_decl_after_type" {
			decl := node.NextNamedSibling()
			name := convert_expression(decl.ChildByFieldName("name"), source)
			if name == nil {
				panic("Expected ident inside local_decl_after_type")
			}

			next := decl.NextNamedSibling()
			for {
				if next == nil {
					break
				}

				if next.Type() == "attributes" {
					// TODO
				}

				next = next.NextSibling()
			}
			right := node.Parent().ChildByFieldName("right")
			var initializer Expression
			if right != nil {
				initializer = convert_expression(right, source)
			}

			return VariableDecl{
				ASTNodeBase: NewBaseNodeBuilder().
					WithSitterPosRange(node.StartPoint(), node.NextSibling().EndPoint()).
					Build(),
				Type:        typeInfo,
				Names:       []Identifier{name.(Identifier)},
				Initializer: initializer,
			}, 1

		}
		return typeInfo, 0
	} else {
		return convert_expression(node, source), 0
	}

	return nil, 0
}

func convert_lambda_declaration(node *sitter.Node, source []byte) Expression {
	debugNode(node, source)
	rType := option.None[TypeInfo]()
	parameters := []FunctionParameter{}

	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		switch n.Type() {
		case "type", "optional_type":
			r := convert_type(n, source)
			rType = option.Some[TypeInfo](r)
		case "fn_parameter_list":

		case "attributes":
			// TODO
		}
	}

	return LambdaDeclaration{
		ASTNodeBase: NewBaseNodeBuilder().WithSitterPos(node).Build(),
		ReturnType:  rType,
		Parameters:  parameters,
	}
}

func convert_compound_stmt(node *sitter.Node, source []byte) []Expression {

	return []Expression{}
}

func debugNode(node *sitter.Node, source []byte) {
	fmt.Printf("%s: %s\n----- %s\n", node.Type(), node.Content(source), node)
}
