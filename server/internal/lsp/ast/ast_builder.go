package ast

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

// --
// ASTBaseNodeBuilder
// --
type ASTBaseNodeBuilder struct {
	bn NodeAttributes
}

func NewBaseNodeBuilder() *ASTBaseNodeBuilder {
	return &ASTBaseNodeBuilder{
		bn: NodeAttributes{},
	}
}

func NewBaseNodeFromSitterNode(node *sitter.Node) NodeAttributes {
	builder := NewBaseNodeBuilder().
		WithSitterPos(node)

	return builder.Build()
}
func (d *ASTBaseNodeBuilder) Build() NodeAttributes {
	return d.bn
}

func (d *ASTBaseNodeBuilder) WithSitterPosRange(start sitter.Point, end sitter.Point) *ASTBaseNodeBuilder {
	d.bn.StartPos = Position{
		Column: uint(start.Column),
		Line:   uint(start.Row),
	}
	d.bn.EndPos = Position{
		Column: uint(end.Column),
		Line:   uint(end.Row),
	}
	return d
}

func (i *ASTBaseNodeBuilder) WithSitterPos(node *sitter.Node) *ASTBaseNodeBuilder {
	i.WithSitterPosRange(node.StartPoint(), node.EndPoint())
	return i
}

func (d *ASTBaseNodeBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *ASTBaseNodeBuilder {
	d.bn.StartPos = Position{startRow, startCol}
	d.bn.EndPos = Position{endRow, endCol}
	return d
}

// --
// IdentifierBuilder
// --
type IdentifierBuilder struct {
	bi *Ident
	bn ASTBaseNodeBuilder
}

func NewIdentifierBuilder() *IdentifierBuilder {
	return &IdentifierBuilder{
		bi: &Ident{},
		bn: *NewBaseNodeBuilder(),
	}
}

func (i *IdentifierBuilder) WithName(name string) *IdentifierBuilder {
	i.bi.Name = name
	return i
}
func (i *IdentifierBuilder) WithPath(path string) *IdentifierBuilder {
	i.bi.ModulePath = path
	return i
}

func (i *IdentifierBuilder) WithSitterPos(node *sitter.Node) *IdentifierBuilder {
	i.bn.WithSitterPosRange(node.StartPoint(), node.EndPoint())
	return i
}

func (i *IdentifierBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *IdentifierBuilder {
	i.bn.WithStartEnd(startRow, startCol, endRow, endCol)
	return i
}

func (i *IdentifierBuilder) Build() Ident {
	ident := i.bi
	ident.NodeAttributes = i.bn.Build()

	return *ident
}

// --
// TypeInfoBuilder
// --
type TypeInfoBuilder struct {
	t TypeInfo
}

func NewTypeInfoBuilder() *TypeInfoBuilder {
	return &TypeInfoBuilder{
		t: TypeInfo{
			NodeAttributes: NodeAttributes{},
		},
	}
}

func (b *TypeInfoBuilder) IsOptional() *TypeInfoBuilder {
	b.t.Optional = true
	return b
}

func (b *TypeInfoBuilder) IsReference() *TypeInfoBuilder {
	b.t.Reference = true
	return b
}
func (b *TypeInfoBuilder) IsBuiltin() *TypeInfoBuilder {
	b.t.BuiltIn = true
	return b
}
func (b *TypeInfoBuilder) IsStatic() *TypeInfoBuilder {
	b.t.Static = true
	return b
}
func (b *TypeInfoBuilder) IsPointer() *TypeInfoBuilder {
	b.t.Pointer = 1
	return b
}

func (b *TypeInfoBuilder) WithGeneric(name string, startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.t.Generics = append(
		b.t.Generics,
		NewTypeInfoBuilder().
			WithName(name).
			WithNameStartEnd(startRow, startCol, endRow, endCol).
			WithStartEnd(startRow, startCol, endRow, endCol).
			Build(),
	)

	return b
}

func (b *TypeInfoBuilder) WithName(name string) *TypeInfoBuilder {
	b.t.Identifier.Name = name

	return b
}

func (b *TypeInfoBuilder) WithPath(path string) *TypeInfoBuilder {
	b.t.Identifier.ModulePath = path

	return b
}

func (b *TypeInfoBuilder) WithNameStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.t.Identifier.StartPos = Position{startRow, startCol}
	b.t.Identifier.EndPos = Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.t.NodeAttributes.StartPos = Position{startRow, startCol}
	b.t.NodeAttributes.EndPos = Position{endRow, endCol}
	return b
}

func (i *TypeInfoBuilder) Build() TypeInfo {
	return i.t
}

// --
// DefDeclBuilder
// --
type DefDeclBuilder struct {
	d DefDecl
	a ASTBaseNodeBuilder
}

func NewDefDeclBuilder() *DefDeclBuilder {
	return &DefDeclBuilder{
		d: DefDecl{},
		a: *NewBaseNodeBuilder(),
	}
}

func (b *DefDeclBuilder) WithResolvesToType(typeInfo TypeInfo) *DefDeclBuilder {
	b.d.resolvesToType = option.Some(typeInfo)
	return b
}

func (b *DefDeclBuilder) WithResolvesTo(resolvesTo string) *DefDeclBuilder {
	b.d.resolvesTo = resolvesTo
	return b
}

func (b *DefDeclBuilder) WithSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.a.WithSitterPosRange(node.StartPoint(), node.EndPoint())
	return b
}

func (b *DefDeclBuilder) WithName(name string) *DefDeclBuilder {
	b.d.Name.Name = name
	return b
}
func (b *DefDeclBuilder) WithIdentifierSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.d.Name.StartPos = Position{uint(node.StartPoint().Row), uint(node.StartPoint().Column)}
	b.d.Name.EndPos = Position{uint(node.EndPoint().Row), uint(node.EndPoint().Column)}

	return b
}

func (b *DefDeclBuilder) Build() DefDecl {
	def := b.d
	def.NodeAttributes = b.a.Build()

	return def
}
