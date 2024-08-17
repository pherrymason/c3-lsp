package ast

type Statement interface {
	ASTNode
}

type ExpressionStatement struct {
	ASTNodeBase
	Expr Expression
}

type AssignmentStatement struct {
	ASTNodeBase
	Left     Expression
	Right    Expression
	Operator string
}

type TernaryExpression struct {
	ASTNodeBase
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

type Expression interface {
	ASTNode
}

/*
*
assignment_expr,
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
$._base_expr

	'true',
	'false',
	'null',
	$.builtin,
	$.integer_literal,
	$.real_literal,
	$.char_literal,
	$.string_literal,
	$.raw_string_literal,
	$.string_expr,
	$.bytes_expr,

	$._ident_expr,
	$._local_ident_expr,

	$.initializer_list,
	seq($.type, $.initializer_list),

	$.module_ident_expr,
	$.field_expr,
	$.type_access_expr,
	$.paren_expr,
	$.expr_block,

	'$vacount',
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
type UnaryExpression struct {
	ASTNodeBase
	Operator   string
	Expression Expression
}

// BinaryExpr representa una expresión binaria (como suma, resta, etc.)
type BinaryExpr struct {
	ASTNodeBase
	Left     ASTNode
	Operator string
	Right    ASTNode
}

type OptionalExpression struct {
	ASTNodeBase
	Argument Expression
	Operator string
}

type CastExpression struct {
	ASTNodeBase
	Type  TypeInfo
	Value Expression
}

type InlineTypeWithInitizlization struct {
	ASTNodeBase
	Type            TypeInfo
	InitializerList InitializerList
}
