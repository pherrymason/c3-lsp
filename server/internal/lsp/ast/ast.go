package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

type Position struct {
	Line, Column uint
}

const (
	ResolveStatusPending = iota
	ResolveStatusDone
)

type NodeType int

const (
	TypeFile = iota
	TypeModule
	TypeIdentifier
	TypeTypeInfo
	TypeDefDecl
	TypeVariableDecl
	TypeEnumDecl
	TypeEnumProperty
	TypeEnumMember
	TypeStructDecl
	TypeStructMemberDecl
	TypeFaultDecl
	TypeFaultMemberDecl
	TypeConstDecl
	TypeInterfaceDecl
	TypeMacroDecl
	TypeFunctionDecl
	TypeFunctionSignature
	TypeFunctionParameter
	TypeLambdaDecl
	TypeLambdaExpr
	TypeAssignmentStatement
	TypeBinaryExpr
	TypeTernaryExpr
	TypeElvisOrElseExpr
	TypeOptionalExpr
	TypeUnaryExpr
	TypeUpdateExpr
	TypeSubscriptExpr
	TypeCastExpr
	TypeRethrowExpr
	TypeCallExpr
	TypeTrailingGenericExpr
	TypeInlineTypeWithInitExpr
	TypeInitializerList
	TypeReturnStatement
	TypeCompoundStatement
	TypeContinueStatement
	TypeBreakStatement
	TypeSwitchStatement
	TypeSwitchCaseStatement
	TypeSwitchCaseRangeExpr
	TypeNextCaseStatement
	TypeIfStatement
	TypeElseStatement
	TypeForStatement
	TypeForeachStatement
	TypeWhileStatement
	TypeDoStatement
	TypeDeferStatement
	TypeAssertStatement
	TypeFunctionCallExpr
)

type ASTNode interface {
	TypeNode() NodeType
	StartPosition() Position
	EndPosition() Position
}

type ASTBaseNode struct {
	StartPos, EndPos Position
	Attributes       []string
	Kind             NodeType
}

func (n ASTBaseNode) TypeNode() NodeType {
	return n.Kind
}

func (n ASTBaseNode) StartPosition() Position {
	return n.StartPos
}
func (n ASTBaseNode) EndPosition() Position {
	return n.EndPos
}

func (n *ASTBaseNode) SetPos(start sitter.Point, end sitter.Point) {
	n.StartPos = Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.EndPos = Position{Line: uint(end.Row), Column: uint(end.Column)}
}

type File struct {
	ASTBaseNode
	Name    string
	Modules []Module
}

type Module struct {
	ASTBaseNode
	Name              string
	GenericParameters []string
	Functions         []Declaration
	Macros            []Declaration
	Declarations      []Declaration
	Variables         []VariableDecl
	Imports           []Import
}

type Import struct {
	ASTBaseNode
	Path string
}

type Declaration interface {
	ASTNode
}

type VariableDecl struct {
	ASTBaseNode
	Names       []Identifier
	Type        TypeInfo
	Initializer Expression
}

type ConstDecl struct {
	ASTBaseNode
	Names       []Identifier
	Type        option.Option[TypeInfo]
	Initializer Expression
}

type EnumDecl struct {
	ASTBaseNode
	Name       string
	BaseType   TypeInfo
	Properties []EnumProperty
	Members    []EnumMember
}

type EnumProperty struct {
	ASTBaseNode
	Type TypeInfo
	Name Identifier
}

type EnumMember struct {
	ASTBaseNode
	Name  Identifier
	Value CompositeLiteral
}

type PropertyValue struct {
	ASTBaseNode
	Name  string
	Value Expression
}

const (
	StructTypeNormal = iota
	StructTypeUnion
	StructTypeBitStruct
)

type StructType int

type StructDecl struct {
	ASTBaseNode
	Name        string
	BackingType option.Option[TypeInfo]
	Members     []StructMemberDecl
	StructType  StructType
	Implements  []string
}

type StructMemberDecl struct {
	ASTBaseNode
	Names     []Identifier
	Type      TypeInfo
	BitRange  option.Option[[2]uint]
	IsInlined bool
}

type FaultDecl struct {
	ASTBaseNode
	Name        Identifier
	BackingType option.Option[TypeInfo]
	Members     []FaultMember
}

type FaultMember struct {
	ASTBaseNode
	Name Identifier
}

type DefDecl struct {
	ASTBaseNode
	Name           Identifier
	resolvesTo     string
	resolvesToType option.Option[TypeInfo]
}

type MacroDecl struct {
	ASTBaseNode
	Signature MacroSignature
	Body      Block
}

type MacroSignature struct {
	Name       Identifier
	Parameters []FunctionParameter
}

type LambdaDeclaration struct {
	ASTBaseNode
	Parameters []FunctionParameter
	ReturnType option.Option[TypeInfo]
	Body       Expression
}

type FunctionDecl struct {
	ASTBaseNode
	ParentTypeId option.Option[Identifier]
	Signature    FunctionSignature
	Body         Expression
}

type FunctionSignature struct {
	ASTBaseNode
	Name       Identifier
	Parameters []FunctionParameter
	ReturnType TypeInfo
}

type FunctionParameter struct {
	ASTBaseNode
	Name Identifier
	Type TypeInfo
}

type Block struct {
	ASTBaseNode
	Declarations []Declaration
	Statements   []Expression
}

type FunctionCall struct {
	ASTBaseNode
	Identifier       Expression
	GenericArguments option.Option[[]Expression]
	Arguments        []Arg
	TrailingBlock    option.Option[CompoundStatement]
}

type InterfaceDecl struct {
	ASTBaseNode
	Name    Identifier
	Methods []FunctionSignature
}

type TypeInfo struct {
	ASTBaseNode
	ResolveStatus int
	Identifier    Identifier
	Pointer       uint
	Optional      bool
	BuiltIn       bool
	Static        bool
	Reference     bool
	TLocal        bool
	Generics      []TypeInfo
}

type Identifier struct {
	ASTBaseNode
	Name string
	Path string
}

// Used only as a temporal container.
// It is decomposed and its info extracted to build other ast nodes.
type TrailingGenericsExpr struct {
	ASTBaseNode
	Identifier       Identifier
	GenericArguments []Expression
}

type Literal struct {
	ASTBaseNode
	Value string
}
type IntegerLiteral struct {
	ASTBaseNode
	Value string
}
type RealLiteral struct {
	ASTBaseNode
	Value string
}

type BoolLiteral struct {
	ASTBaseNode
	Value bool
}
type CompositeLiteral struct {
	ASTBaseNode
	Values []Expression
}

type InitializerList struct {
	ASTBaseNode
	Args []Expression
}

const (
	PathTypeIndexed = iota
	PathTypeField
	PathTypeRange
)

type Path struct {
	ASTBaseNode
	PathType  int
	Path      string
	PathStart string
	PathEnd   string
	FieldName string
}

type Arg interface {
	ASTNode
}
type ArgParamPathSet struct {
	ASTBaseNode
	Path string
	Expr Expression
}

type ArgFieldSet struct {
	ASTBaseNode
	FieldName string
	Expr      Expression
}

func (arg *ArgFieldSet) SetExpr(expr Expression) {
	arg.Expr = expr
}

type IndexAccess struct {
	ASTBaseNode
	Array Expression
	Index string
}

// TODO Replace by RangeIndex
type RangeAccess struct {
	ASTBaseNode
	Array      Expression
	RangeStart uint
	RangeEnd   uint
}

type FieldAccess struct {
	ASTBaseNode
	Object Expression
	Field  Expression
}

type CompoundStatement struct {
	ASTBaseNode
	Statements []Expression
}

type ReturnStatement struct {
	ASTBaseNode
	Return option.Option[Expression]
}

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

type AssertStatement struct {
	ASTBaseNode
	Assertions []Expression
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

// BinaryExpression representa una expresión binaria (como suma, resta, etc.)
type BinaryExpression struct {
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
