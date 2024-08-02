package ast

import sitter "github.com/smacker/go-tree-sitter"

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

func (d *ASTBaseNodeBuilder) WithStartEnd(startRow uint, startCol uint, endRow uint, endCol uint) *ASTBaseNodeBuilder {
	d.bn.StartPos = Position{startRow, startCol}
	d.bn.EndPos = Position{endRow, endCol}
	return d
}

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
