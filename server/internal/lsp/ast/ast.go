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

type ASTNodeBase struct {
	StartPos, EndPos Position
	Attributes       []string
}

func (n ASTNodeBase) Start() Position {
	return n.StartPos
}

func (n ASTNodeBase) End() Position {
	return n.EndPos
}

func (n *ASTNodeBase) SetPos(start sitter.Point, end sitter.Point) {
	n.StartPos = Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.EndPos = Position{Line: uint(end.Row), Column: uint(end.Column)}
}

type ASTNode interface {
	Start() Position
	End() Position
}

type File struct {
	ASTNodeBase
	Name    string
	Modules []Module
}

type Module struct {
	ASTNodeBase
	Name              string
	GenericParameters []string
	Functions         []Declaration
	Macros            []Declaration
	Declarations      []Declaration
	Imports           []string
}

type Declaration interface {
	ASTNode
}

type VariableDecl struct {
	ASTNodeBase
	Names []Identifier
	Type  TypeInfo
	//Initializer Initializer
}

type ConstDecl struct {
	ASTNodeBase
	Names []Identifier
	Type  TypeInfo
}

type EnumDecl struct {
	ASTNodeBase
	Name       string
	BaseType   TypeInfo
	Properties []EnumProperty
	Members    []EnumMember
}

type EnumProperty struct {
	ASTNodeBase
	Type TypeInfo
	Name Identifier
}

type EnumMember struct {
	ASTNodeBase
	Name  Identifier
	Value CompositeLiteral
}

type PropertyValue struct {
	ASTNodeBase
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
	ASTNodeBase
	Name        string
	BackingType option.Option[TypeInfo]
	Members     []StructMemberDecl
	StructType  StructType
	Implements  []string
}

type StructMemberDecl struct {
	ASTNodeBase
	Names     []Identifier
	Type      TypeInfo
	BitRange  option.Option[[2]uint]
	IsInlined bool
}

type FaultDecl struct {
	ASTNodeBase
	Name        Identifier
	BackingType option.Option[TypeInfo]
	Members     []FaultMember
}

type FaultMember struct {
	ASTNodeBase
	Name Identifier
}

type DefDecl struct {
	ASTNodeBase
	Name           Identifier
	resolvesTo     string
	resolvesToType option.Option[TypeInfo]
}

type MacroDecl struct {
	ASTNodeBase
	Signature MacroSignature
	Body      Block
}

type MacroSignature struct {
	Name       Identifier
	Parameters []FunctionParameter
}

type FunctionDecl struct {
	ASTNodeBase
	ParentTypeId option.Option[Identifier]
	Signature    FunctionSignature
	Body         Block
}

type FunctionSignature struct {
	ASTNodeBase
	Name       Identifier
	Parameters []FunctionParameter
	ReturnType TypeInfo
}

type FunctionParameter struct {
	ASTNodeBase
	Name Identifier
	Type TypeInfo
}

type Block struct {
	ASTNodeBase
	Statements []ASTNode
}

type FunctionCall struct {
	ASTNodeBase
}

type InterfaceDecl struct {
	ASTNodeBase
	Name    Identifier
	Methods []FunctionSignature
}

type TypeInfo struct {
	ASTNodeBase
	ResolveStatus int
	Identifier    Identifier
	Pointer       uint
	Optional      bool
	BuiltIn       bool
	Generics      []TypeInfo
}

type Identifier struct {
	ASTNodeBase
	Name string
	Path string
}

type Expression interface {
	ASTNode
}

type Literal struct {
	ASTNodeBase
	Value string
}
type NumberLiteral struct {
	ASTNodeBase
	Value uint
}

type BoolLiteral struct {
	ASTNodeBase
	Value bool
}

type CompositeLiteral struct {
	ASTNodeBase
	Values []Expression
}

// BinaryExpr representa una expresión binaria (como suma, resta, etc.)
type BinaryExpr struct {
	ASTNodeBase
	Left     ASTNode
	Operator string
	Right    ASTNode
}
