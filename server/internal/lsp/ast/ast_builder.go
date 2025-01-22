package ast

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
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

func NewAttrNodeFromSitterNode(nodeId NodeId, node *sitter.Node) NodeAttributes {
	builder := NewNodeAttributesBuilder().
		WithId(nodeId).
		WithRange(lsp.NewRangeFromSitterNode(node))

	return builder.Build()
}
func (d *NodeAttrsBuilder) Build() NodeAttributes {
	return d.bn
}

func (d *NodeAttrsBuilder) WithId(id NodeId) *NodeAttrsBuilder {
	d.bn.Id = id
	return d
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
	ident       *Ident
	attrBuilder NodeAttrsBuilder
}

func NewIdentifierBuilder() *IdentifierBuilder {
	return &IdentifierBuilder{
		ident:       &Ident{},
		attrBuilder: *NewNodeAttributesBuilder(),
	}
}

func (i *IdentifierBuilder) WithId(nodeId NodeId) *IdentifierBuilder {
	i.attrBuilder.WithId(nodeId)
	return i
}
func (i *IdentifierBuilder) WithName(name string) *IdentifierBuilder {
	i.ident.Name = name
	return i
}

func (i *IdentifierBuilder) WithSitterPos(node *sitter.Node) *IdentifierBuilder {
	i.attrBuilder.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	i.attrBuilder.WithRange(lsp.NewRangeFromSitterNode(node))
	return i
}

func (i *IdentifierBuilder) WithSitterRange(node *sitter.Node) *IdentifierBuilder {
	i.attrBuilder.WithRange(lsp.NewRangeFromSitterNode(node))
	return i
}

func (i *IdentifierBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *IdentifierBuilder {
	i.WithRange(lsp.NewRange(startRow, startCol, endRow, endCol))
	return i
}

func (i *IdentifierBuilder) WithRange(pathRange lsp.Range) *IdentifierBuilder {
	i.attrBuilder.WithRange(pathRange)
	return i
}
func (i *IdentifierBuilder) Build() Ident {
	ident := i.ident
	ident.NodeAttributes = i.attrBuilder.Build()

	return *ident
}

func (i *IdentifierBuilder) BuildPtr() *Ident {
	ident := i.ident
	ident.NodeAttributes = i.attrBuilder.Build()

	return ident
}

// --
// TypeInfoBuilder
// --
type TypeInfoBuilder struct {
	typeInfo TypeInfo
}

func NewTypeInfoBuilder() *TypeInfoBuilder {
	return &TypeInfoBuilder{
		typeInfo: TypeInfo{
			NodeAttributes: NodeAttributes{},
			Identifier: &Ident{
				ModulePath: nil,
			},
		},
	}
}

func (b *TypeInfoBuilder) IsOptional() *TypeInfoBuilder {
	b.typeInfo.Optional = true
	return b
}

func (b *TypeInfoBuilder) IsReference() *TypeInfoBuilder {
	b.typeInfo.Reference = true
	return b
}
func (b *TypeInfoBuilder) IsBuiltin() *TypeInfoBuilder {
	b.typeInfo.BuiltIn = true
	return b
}
func (b *TypeInfoBuilder) IsStatic() *TypeInfoBuilder {
	b.typeInfo.Static = true
	return b
}
func (b *TypeInfoBuilder) IsPointer() *TypeInfoBuilder {
	b.typeInfo.Pointer = 1
	return b
}

func (b *TypeInfoBuilder) WithGeneric(name string, startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.Generics = append(
		b.typeInfo.Generics,
		NewTypeInfoBuilder().
			WithName(name).
			WithNameStartEnd(startRow, startCol, endRow, endCol).
			WithStartEnd(startRow, startCol, endRow, endCol).
			Build(),
	)

	return b
}

func (b *TypeInfoBuilder) WithName(name string) *TypeInfoBuilder {
	b.typeInfo.Identifier.Name = name

	return b
}

func (b *TypeInfoBuilder) WithPath(path *Ident) *TypeInfoBuilder {
	b.typeInfo.Identifier.ModulePath = path

	return b
}

func (b *TypeInfoBuilder) WithNameStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.Identifier.Range.Start = lsp.Position{startRow, startCol}
	b.typeInfo.Identifier.Range.End = lsp.Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.typeInfo.NodeAttributes.Range.Start = lsp.Position{startRow, startCol}
	b.typeInfo.NodeAttributes.Range.End = lsp.Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) Build() TypeInfo {
	return b.typeInfo
}

// --
// DefDeclBuilder
// --
type DefDeclBuilder struct {
	d DefDecl
	a NodeAttrsBuilder
}

func NewDefDeclBuilder(nodeId NodeId) *DefDeclBuilder {
	return &DefDeclBuilder{
		d: DefDecl{},
		a: *NewNodeAttributesBuilder(),
	}
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

func (b *DefDeclBuilder) WithExpression(expression Expression) *DefDeclBuilder {
	b.d.Expr = expression
	return b
}

func (b *DefDeclBuilder) Build() DefDecl {
	def := b.d
	def.NodeAttributes = b.a.Build()

	return def
}
