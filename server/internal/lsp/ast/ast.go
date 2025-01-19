package ast

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
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

	// Types
	VAR
	CONST
	STRUCT
	UNION
	ENUM
	FAULT
	DEF
	FUNCTION
	FIELD
)

type Token int
type NodeId uint

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
	GetRange() lsp.Range
	GetId() NodeId
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
	Range      lsp.Range
	Attributes []string
	Id         NodeId
}

func (n NodeAttributes) StartPosition() lsp.Position { return n.Range.Start }
func (n NodeAttributes) EndPosition() lsp.Position   { return n.Range.End }
func (n NodeAttributes) GetRange() lsp.Range         { return n.Range }
func (n NodeAttributes) GetId() NodeId               { return n.Id }

func ChangeNodePosition(n *NodeAttributes, start sitter.Point, end sitter.Point) {
	n.Range.Start = lsp.Position{Line: uint(start.Row), Column: uint(start.Column)}
	n.Range.End = lsp.Position{Line: uint(end.Row), Column: uint(end.Column)}
}

type File struct {
	NodeAttributes
	URI     string
	Modules []Module
}

func NewFile(nodeId NodeId, uri protocol.URI, aRange lsp.Range, modules []Module) *File {
	node := &File{
		URI: uri,
		NodeAttributes: NewNodeAttributesBuilder().
			WithId(nodeId).
			WithRange(aRange).Build(),
		Modules: modules,
	}

	return node
}
func (f *File) AddModule(module Module) {
	f.Modules = append(f.Modules, module)
}

type Module struct {
	NodeAttributes
	Name              string
	GenericParameters []string
	Declarations      []Declaration // Top level declarations
	Imports           []*Import     // Imports in this file
}

func NewModule(nodeId NodeId, name string, aRange lsp.Range, file *File) *Module {
	return &Module{
		Name: name,
		NodeAttributes: NodeAttributes{
			Id:    nodeId,
			Range: aRange,
		},
	}
}

type Import struct {
	NodeAttributes
	Path string
}

func (*Import) stmtNode() {}

// Deprecated: using GenDecl for enums
type EnumProperty struct {
	NodeAttributes
	Type TypeInfo
	Name Ident
}

// EnumMember
// Deprecated: using GenDecl for enums
type EnumMember struct {
	NodeAttributes
	Name  Ident
	Value CompositeLiteral
}

// Deprecated not used
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
