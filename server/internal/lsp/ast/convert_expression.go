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
	//fmt.Print("convert_expression:\n")
	//debugNode(node, source)
	return anyOf([]NodeRule{
		NodeOfType("assignment_expr"),
		NodeOfType("ternary_expr"),
		NodeChildWithSequenceOf([]NodeRule{
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
}

func convert_base_expression(node *sitter.Node, source []byte) Expression {
	var expression Expression
	//debugNode(node, source)

	return anyOf([]NodeRule{
		NodeOfType("string_literal"),
		NodeOfType("char_literal"),
		NodeOfType("raw_string_literal"),
		NodeOfType("integer_literal"),
		NodeOfType("real_literal"),
		NodeOfType("bytes_literal"),
		NodeOfType("true"),
		NodeOfType("false"),
		NodeOfType("null"),

		NodeOfType("ident"),
		NodeOfType("ct_ident"),
		NodeOfType("hash_ident"),
		NodeOfType("const_ident"),
		NodeOfType("at_ident"),
		NodeOfType("module_ident_expr"),
		NodeOfType("bytes_expr"),
		NodeOfType("builtin"),
		NodeOfType("unary_expr"),
		NodeOfType("initializer_list"),
		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("type"),
			NodeOfType("initializer_list"),
		}, "..type_with_initializer_list.."),

		NodeOfType("field_expr"),       // TODO
		NodeOfType("type_access_expr"), // TODO
		NodeOfType("paren_expr"),       // TODO
		NodeOfType("expr_block"),       // TODO

		NodeOfType("$vacount"),

		NodeOfType("$alignof"),
		NodeOfType("$extnameof"),
		NodeOfType("$nameof"),
		NodeOfType("$offsetof"),
		NodeOfType("$qnameof"),

		NodeOfType("$vaconst"),
		NodeOfType("$vaarg"),
		NodeOfType("$varef"),
		NodeOfType("$vaexpr"),

		NodeOfType("$eval"),
		NodeOfType("$is_const"),
		NodeOfType("$sizeof"),
		NodeOfType("$stringify"),

		NodeOfType("$feature"),

		NodeOfType("$and"),
		NodeOfType("$append"),
		NodeOfType("$concat"),
		NodeOfType("$defined"),
		NodeOfType("$embed"),
		NodeOfType("$or"),

		NodeOfType("$assignable"), // TODO

		NodeSiblingsWithSequenceOf([]NodeRule{
			NodeOfType("lambda_declaration"),
			NodeOfType("compound_stmt"),
		}, "..lambda_declaration_with_body.."),
	}, node, source)

	return expression
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
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
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
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Left:        left,
		Operator:    operator,
		Right:       right,
	}
}

func convert_bytes_expr(node *sitter.Node, source []byte) Expression {
	return convert_literal(node.Child(0), source)
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
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Condition:   condition,
		Consequence: convert_expression(node.ChildByFieldName("consequence"), source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_elvis_orelse_expr(node *sitter.Node, source []byte) Expression {
	conditionNode := node.ChildByFieldName("condition")

	return TernaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Condition:   convert_expression(conditionNode, source),
		Consequence: convert_ident(conditionNode, source),
		Alternative: convert_expression(node.ChildByFieldName("alternative"), source),
	}
}

func convert_optional_expr(node *sitter.Node, source []byte) Expression {
	operatorNode := node.ChildByFieldName("operator")
	operator := operatorNode.Content(source)
	if operatorNode.NextSibling() != nil && operatorNode.NextSibling().Type() == "!" {
		operator += "!"
	}

	argumentNode := node.ChildByFieldName("argument")
	return OptionalExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    operator,
		Argument:    convert_expression(argumentNode, source),
	}
}

func convert_unary_expr(node *sitter.Node, source []byte) Expression {
	return UnaryExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_update_expr(node *sitter.Node, source []byte) Expression {
	return UpdateExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_subscript_expr(node *sitter.Node, source []byte) Expression {
	var index Expression
	indexNode := node.ChildByFieldName("index")
	if indexNode != nil {
		index = convert_expression(indexNode, source)
	} else {
		rangeNode := node.ChildByFieldName("range")
		if rangeNode != nil {
			index = convert_range_expr(rangeNode, source)
		}
	}

	return SubscriptExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Index:       index,
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_cast_expr(node *sitter.Node, source []byte) Expression {
	return CastExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Type:        convert_type(node.ChildByFieldName("type"), source).(TypeInfo),
		Argument:    convert_expression(node.ChildByFieldName("value"), source),
	}
}

func convert_rethrow_expr(node *sitter.Node, source []byte) Expression {
	return RethrowExpression{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Operator:    node.ChildByFieldName("operator").Content(source),
		Argument:    convert_expression(node.ChildByFieldName("argument"), source),
	}
}

func convert_call_expr(node *sitter.Node, source []byte) Expression {

	invocationNode := node.ChildByFieldName("arguments")
	args := []Arg{}
	for i := 0; i < int(invocationNode.ChildCount()); i++ {
		n := invocationNode.Child(i)
		if n.Type() == "arg" {
			debugNode(n, source)
			args = append(args, convert_arg(n, source))
		}
	}

	trailingNode := node.ChildByFieldName("trailing")
	compoundStmt := option.None[CompoundStatement]()
	if trailingNode != nil {
		compoundStmt = option.Some(convert_compound_stmt(trailingNode, source).(CompoundStatement))
	}

	expr := convert_expression(node.ChildByFieldName("function"), source)
	var identifier Expression
	genericArguments := option.None[[]Expression]()
	switch expr.(type) {
	case Identifier:
		identifier = expr
	case TrailingGenericsExpr:
		identifier = expr.(TrailingGenericsExpr).Identifier
		ga := expr.(TrailingGenericsExpr).GenericArguments
		genericArguments = option.Some(ga)
	}

	return FunctionCall{
		ASTBaseNode:      NewBaseNodeFromSitterNode(node),
		Identifier:       identifier,
		GenericArguments: genericArguments,
		Arguments:        args,
		TrailingBlock:    compoundStmt,
	}
}

/*
trailing_generic_expr: $ => prec.right(PREC.TRAILING, seq(
field('argument', $._expr),
field('operator', $.generic_arguments),
)),
*/
func convert_trailing_generic_expr(node *sitter.Node, source []byte) Expression {
	argNode := node.ChildByFieldName("argument")
	expr := convert_expression(argNode, source)

	operator := convert_generic_arguments(node.ChildByFieldName("operator"), source)

	return TrailingGenericsExpr{
		ASTBaseNode:      NewBaseNodeFromSitterNode(node),
		Identifier:       expr.(Identifier),
		GenericArguments: operator,
	}
}

func convert_generic_arguments(node *sitter.Node, source []byte) []Expression {
	args := []Expression{}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)

		switch n.Type() {
		case "(<", ">)", ",":
			//ignore
		case "type":
			args = append(args, convert_type(n, source))

		default:
			args = append(args, convert_expression(n, source))
		}
	}

	return args
}

func convert_type_with_initializer_list(node *sitter.Node, source []byte) Expression {
	baseExpr := convert_base_expression(node.NextNamedSibling(), source)
	initList, ok := baseExpr.(InitializerList)
	if !ok {
		initList = InitializerList{}
	}

	expression := InlineTypeWithInitizlization{
		ASTBaseNode: NewBaseNodeBuilder().
			WithStartEnd(
				uint(node.StartPoint().Row),
				uint(node.StartPoint().Column),
				initList.ASTBaseNode.EndPos.Line,
				initList.ASTBaseNode.EndPos.Column,
			).Build(),
		Type:            convert_type(node, source).(TypeInfo),
		InitializerList: initList,
	}

	return expression
}

func convert_module_ident_expr(node *sitter.Node, source []byte) Expression {
	return NewIdentifierBuilder().
		WithName(node.ChildByFieldName("ident").Content(source)).
		WithPath(node.Child(0).Child(0).Content(source)).
		WithSitterPos(node).Build()
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

func convert_as_literal(node *sitter.Node, source []byte) Expression {
	return Literal{Value: node.Content(source)}
}

func convert_initializer_list(node *sitter.Node, source []byte) Expression {
	initList := InitializerList{
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == "arg" {
			initList.Args = append(initList.Args, convert_arg(n, source))
		}
	}

	return initList
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
	default:

		// try expr
		expr := convert_expression(node.Child(0), source)
		if expr != nil {
			return expr
		}
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

func convert_range_expr(node *sitter.Node, source []byte) Expression {
	leftNode := node.ChildByFieldName("left")
	rightNode := node.ChildByFieldName("right")

	left := option.None[uint]()
	right := option.None[uint]()
	if leftNode != nil {
		left = option.Some(utils.StringToUint(leftNode.Content(source)))
	}
	if rightNode != nil {
		right = option.Some(utils.StringToUint(rightNode.Content(source)))
	}

	return RangeIndex{
		Start: left,
		End:   right,
	}
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
		ASTBaseNode: NewBaseNodeFromSitterNode(node),
		Statements:  []Expression{},
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