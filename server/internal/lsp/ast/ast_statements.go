package ast

import "github.com/pherrymason/c3-lsp/pkg/option"

type Statement interface {
	ASTNode
}

type ExpressionStatement struct {
	ASTBaseNode
	Expr Expression
}

type AssignmentStatement struct {
	ASTBaseNode
	Left     Expression
	Right    Expression
	Operator string
}

type ContinueStatement struct {
	ASTBaseNode
	Label option.Option[string]
}

type BreakStatement struct {
	ASTBaseNode
	Label option.Option[string]
}

type SwitchStatement struct {
	ASTBaseNode
	Label     option.Option[string]
	Condition Expression
	Cases     []SwitchCase
	Default   []Statement
}

type SwitchCase struct {
	ASTBaseNode
	Value      Expression
	Statements []Statement
}

type SwitchCaseRange struct {
	ASTBaseNode
	Start Expression
	End   Expression
}

type Nextcase struct {
	ASTBaseNode
	Label option.Option[string]
	Value Expression
}

type IfStatement struct {
	ASTBaseNode
	Label     option.Option[string]
	Condition []Expression
	Statement Statement
	Else      ElseStatement
}

type ElseStatement struct {
	ASTBaseNode
	Statement Statement
}

type ForStatement struct {
	ASTBaseNode
	Label       option.Option[string]
	Initializer []Expression
	Condition   Expression
	Update      []Expression
	Body        Statement
}

type ForeachStatement struct {
	ASTBaseNode
	Value      ForeachValue
	Index      ForeachValue
	Collection Expression
	Body       Statement
}

type ForeachValue struct {
	Type       TypeInfo
	Identifier Identifier
}

type WhileStatement struct {
	ASTBaseNode
	Condition []Expression
	Body      Statement
}

type DoStatement struct {
	ASTBaseNode
	Condition Expression
	Body      Statement
}

type DeferStatement struct {
	ASTBaseNode
	Statement Statement
}

type TernaryExpression struct {
	ASTBaseNode
	Condition   Expression
	Consequence Expression
	Alternative Expression
}

type UpdateExpression struct {
	ASTBaseNode
	Operator string
	Argument Expression
}

type SubscriptExpression struct {
	ASTBaseNode
	Argument Expression
	Index    Expression // Index can be another expression:
	//  - IntegerLiteral
	//  - RangeIndex
	//  - Identifier
	//  - CallExpression
	//  - ...
}
type RangeIndex struct {
	ASTBaseNode

	Start option.Option[uint]
	End   option.Option[uint]
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
	ASTBaseNode
	Operator string
	Argument Expression
}

// BinaryExpr representa una expresión binaria (como suma, resta, etc.)
type BinaryExpr struct {
	ASTBaseNode
	Left     ASTNode
	Operator string
	Right    ASTNode
}

type OptionalExpression struct {
	ASTBaseNode
	Argument Expression
	Operator string
}

type CastExpression struct {
	ASTBaseNode
	Type     TypeInfo
	Argument Expression
}

type RethrowExpression struct {
	ASTBaseNode
	Operator string
	Argument Expression
}

type InlineTypeWithInitizlization struct {
	ASTBaseNode
	Type            TypeInfo
	InitializerList InitializerList
}
