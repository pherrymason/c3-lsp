package ast_builders

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/smacker/go-tree-sitter"
)

// --
// NodeAttrsBuilder
// --
type NodeAttrsBuilder struct {
	bn ast.NodeAttributes
}

func NewNodeAttributesBuilder() *NodeAttrsBuilder {
	return &NodeAttrsBuilder{
		bn: ast.NodeAttributes{},
	}
}

func NewAttrNodeFromSitterNode(nodeId ast.NodeId, node *sitter.Node) ast.NodeAttributes {
	builder := NewNodeAttributesBuilder().
		WithId(nodeId).
		WithRange(lsp.NewRangeFromSitterNode(node))

	return builder.Build()
}
func (d *NodeAttrsBuilder) Build() ast.NodeAttributes {
	return d.bn
}

func (d *NodeAttrsBuilder) WithId(id ast.NodeId) *NodeAttrsBuilder {
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

func (d *NodeAttrsBuilder) WithSitterPos(node *sitter.Node) *NodeAttrsBuilder {
	d.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	return d
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

func (d *NodeAttrsBuilder) WithDocComment(docComment string) *NodeAttrsBuilder {
	d.bn.DocComment = option.Some(&ast.DocComment{
		Body:      docComment,
		Contracts: []*ast.DocCommentContract{},
	})
	return d
}
