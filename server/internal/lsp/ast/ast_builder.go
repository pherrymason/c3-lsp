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
