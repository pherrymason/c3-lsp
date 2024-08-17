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

type ASTBaseNode struct {
	StartPos, EndPos Position
	Attributes       []string
}

func (n ASTBaseNode) Start() Position {
	return n.StartPos
}

func (n ASTBaseNode) End() Position {
	return n.EndPos
}

func (n *ASTBaseNode) SetPos(start sitter.Point, end sitter.Point) {
	n.StartPos = Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.EndPos = Position{Line: uint(end.Row), Column: uint(end.Column)}
}

type ASTNode interface {
	Start() Position
	End() Position
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
	Names []Identifier
	Type  TypeInfo
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
