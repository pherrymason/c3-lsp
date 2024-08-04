package ast

import sitter "github.com/smacker/go-tree-sitter"

// --
// ASTBaseNodeBuilder
// --
type ASTBaseNodeBuilder struct {
	bn ASTNodeBase
}

func NewBaseNodeBuilder() *ASTBaseNodeBuilder {
	return &ASTBaseNodeBuilder{
		bn: ASTNodeBase{},
	}
}
func (d *ASTBaseNodeBuilder) Build() ASTNodeBase {
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
	bi Identifier
	bn ASTBaseNodeBuilder
}

func NewIdentifierBuilder() *IdentifierBuilder {
	return &IdentifierBuilder{
		bi: Identifier{},
		bn: *NewBaseNodeBuilder(),
	}
}

func (i *IdentifierBuilder) WithName(name string) *IdentifierBuilder {
	i.bi.Name = name
	return i
}
func (i *IdentifierBuilder) WithPath(path string) *IdentifierBuilder {
	i.bi.Path = path
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

func (i *IdentifierBuilder) Build() Identifier {
	ident := i.bi
	ident.ASTNodeBase = i.bn.Build()

	return ident
}

// --
// TypeInfoBuilder
// --
type TypeInfoBuilder struct {
	bt TypeInfo
}

func NewTypeInfoBuilder() *TypeInfoBuilder {
	return &TypeInfoBuilder{
		bt: TypeInfo{},
	}
}

func (b *TypeInfoBuilder) IsBuiltin() *TypeInfoBuilder {
	b.bt.BuiltIn = true
	return b
}
func (b *TypeInfoBuilder) IsPointer() *TypeInfoBuilder {
	b.bt.Pointer = 1
	return b
}

func (b *TypeInfoBuilder) WithName(name string) *TypeInfoBuilder {
	b.bt.Identifier.Name = name

	return b
}

func (b *TypeInfoBuilder) WithNameStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.bt.Identifier.StartPos = Position{startRow, startCol}
	b.bt.Identifier.EndPos = Position{endRow, endCol}
	return b
}

func (b *TypeInfoBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *TypeInfoBuilder {
	b.bt.ASTNodeBase.StartPos = Position{startRow, startCol}
	b.bt.ASTNodeBase.EndPos = Position{endRow, endCol}
	return b
}

func (i *TypeInfoBuilder) Build() TypeInfo {
	return i.bt
}
