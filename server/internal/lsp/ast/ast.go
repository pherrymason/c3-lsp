package ast

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

// This package is heavily inspired by the official go/ast.go package.
// Some comment descriptions might be literal copy-pastes where they apply.

const (
	ResolveStatusPending = iota
	ResolveStatusDone

	// literals
	NULL
	INT   // 12345
	FLOAT // 123.45
	IMAG  // 123.45i
	CHAR  // 'a'
	STRING
	BOOLEAN
)

type Token int

// ----------------------------------------------------------------------------
// Interfaces
//
// There are 3 main classes of nodes:
// - Expressions and types nodes
// - Statement nodes
// - Declaration nodes
//
// All nodes contain position information marking the beginning and end of the
// corresponding source text segment.

type Node interface {
	StartPosition() lsp.Position
	EndPosition() lsp.Position
}

type Expression interface {
	Node
	exprNode()
}

type Declaration interface {
	Node
	declNode()
}

type Statement interface {
	Node
	stmtNode()
}

type EmptyNode struct {
	NodeAttributes
}

func (n *EmptyNode) declNode() {}
func (n *EmptyNode) exprNode() {}
func (n *EmptyNode) stmtNode() {}

// NodeAttributes is a struct that contains the common information all
// AST Nodes contains, like position or other attributes
type NodeAttributes struct {
	StartPos, EndPos lsp.Position
	Attributes       []string
}

func (n NodeAttributes) StartPosition() lsp.Position { return n.StartPos }
func (n NodeAttributes) EndPosition() lsp.Position   { return n.EndPos }

func ChangeNodePosition(n *NodeAttributes, start sitter.Point, end sitter.Point) {
	n.StartPos = lsp.Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.EndPos = lsp.Position{Line: uint(end.Row), Column: uint(end.Column)}
} /*
func (n *NodeAttributes) SetPos(start sitter.Point, end sitter.Point) {
	n.StartPos = Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.EndPos = Position{Line: uint(end.Row), Column: uint(end.Column)}
}*/

type File struct {
	NodeAttributes
	Name    string
	Modules []Module
}

type Module struct {
	NodeAttributes
	Name              string
	GenericParameters []string
	Declarations      []Declaration // Top level declarations
	Imports           []*Import     // Imports in this file
}

type Import struct {
	NodeAttributes
	Path string
}

func (*Import) stmtNode() {}

type EnumProperty struct {
	NodeAttributes
	Type TypeInfo
	Name Ident
}

type EnumMember struct {
	NodeAttributes
	Name  Ident
	Value CompositeLiteral
}

type PropertyValue struct {
	NodeAttributes
	Name  string
	Value Expression
}

type StructMemberDecl struct {
	NodeAttributes
	Names     []Ident
	Type      TypeInfo
	BitRange  option.Option[[2]uint]
	IsInlined bool
}

type FaultMember struct {
	NodeAttributes
	Name Ident
}

type MacroSignature struct {
	Name       Ident
	Parameters []FunctionParameter
}

type FunctionParameter struct {
	NodeAttributes
	Name Ident
	Type TypeInfo
}

// Block
// Only used in MacroDecl.Body
type Block struct {
	NodeAttributes
	Declarations []Declaration
	Statements   []Expression
}

type DeclOrExpr struct {
	NodeAttributes
	Decl Declaration
	Expr Expression
	Stmt Statement
}

func (*DeclOrExpr) exprNode() {}
func (*DeclOrExpr) declNode() {}
