package ast

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/pkg/option"
	sitter "github.com/smacker/go-tree-sitter"
)

// --
// NodeAttrsBuilder
// --
type NodeAttrsBuilder struct {
	bn NodeAttributes
}

func NewNodeAttributesBuilder() *NodeAttrsBuilder {
	return &NodeAttrsBuilder{
		bn: NodeAttributes{},
	}
}

func NewBaseNodeFromSitterNode(node *sitter.Node) NodeAttributes {
	builder := NewNodeAttributesBuilder().
		WithRange(lsp.NewRangeFromSitterNode(node))

	return builder.Build()
}
func (d *NodeAttrsBuilder) Build() NodeAttributes {
	return d.bn
}

func (d *NodeAttrsBuilder) WithSitterStartEnd(start sitter.Point, end sitter.Point) *NodeAttrsBuilder {
	d.bn.Range.Start = lsp.Position{
		Column: uint(start.Column),
		Line:   uint(start.Row),
	}
	d.bn.Range.End = lsp.Position{
		Column: uint(end.Column),
		Line:   uint(end.Row),
	}
	return d
}

func (i *NodeAttrsBuilder) WithSitterPos(node *sitter.Node) *NodeAttrsBuilder {
	i.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	return i
}

func (d *NodeAttrsBuilder) WithRangePositions(startRow uint, startCol uint, endRow uint, endCol uint) *NodeAttrsBuilder {
	d.bn.Range.Start = lsp.Position{startRow, startCol}
	d.bn.Range.End = lsp.Position{endRow, endCol}
	return d
}

func (d *NodeAttrsBuilder) WithRange(aRange lsp.Range) *NodeAttrsBuilder {
	d.bn.Range = aRange
	return d
}

// --
// IdentifierBuilder
// --
type IdentifierBuilder struct {
	bi *Ident
	bn NodeAttrsBuilder
}

func NewIdentifierBuilder() *IdentifierBuilder {
	return &IdentifierBuilder{
		bi: &Ident{},
		bn: *NewNodeAttributesBuilder(),
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
	i.bn.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	i.bn.WithRange(lsp.NewRangeFromSitterNode(node))
	return i
}

func (i *IdentifierBuilder) WithSitterRange(node *sitter.Node) *IdentifierBuilder {
	i.bn.WithRange(lsp.NewRangeFromSitterNode(node))
	return i
}

func (i *IdentifierBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *IdentifierBuilder {
	i.bn.WithRange(lsp.NewRange(startRow, startCol, endRow, endCol))
	return i
}

func (i *IdentifierBuilder) Build() Ident {
	ident := i.bi
	ident.NodeAttributes = i.bn.Build()

	return *ident
}
func (i *IdentifierBuilder) BuildPtr() *Ident {
	ident := i.bi
	ident.NodeAttributes = i.bn.Build()

	return ident
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
	b.t.Identifier.Range.Start = lsp.Position{startRow, startCol}
	b.t.Identifier.Range.End = lsp.Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.t.NodeAttributes.Range.Start = lsp.Position{startRow, startCol}
	b.t.NodeAttributes.Range.End = lsp.Position{endRow, endCol}
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
	a NodeAttrsBuilder
}

func NewDefDeclBuilder() *DefDeclBuilder {
	return &DefDeclBuilder{
		d: DefDecl{},
		a: *NewNodeAttributesBuilder(),
	}
}

func (b *DefDeclBuilder) WithResolvesToType(typeInfo TypeInfo) *DefDeclBuilder {
	b.d.ResolvesToType = option.Some(typeInfo)
	return b
}

func (b *DefDeclBuilder) WithResolvesTo(resolvesTo string) *DefDeclBuilder {
	b.d.ResolvesTo = resolvesTo
	return b
}

func (b *DefDeclBuilder) WithSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.a.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	return b
}

func (b *DefDeclBuilder) WithName(name string) *DefDeclBuilder {
	b.d.Name.Name = name
	return b
}
func (b *DefDeclBuilder) WithIdentifierSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.d.Name.Range.Start = lsp.Position{uint(node.StartPoint().Row), uint(node.StartPoint().Column)}
	b.d.Name.Range.End = lsp.Position{uint(node.EndPoint().Row), uint(node.EndPoint().Column)}

	return b
}

func (b *DefDeclBuilder) Build() DefDecl {
	def := b.d
	def.NodeAttributes = b.a.Build()

	return def
}
