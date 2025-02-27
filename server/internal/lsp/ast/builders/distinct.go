package ast_builders

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

// DistinctBuilder builder
type DistinctBuilder struct {
	distinct     *ast.GenDecl
	a            NodeAttrsBuilder
	ident        *IdentifierBuilder
	distinctType *ast.DistinctType
}

func NewDistinctBuilder(nodeId ast.NodeId) *DistinctBuilder {
	attrs := *NewNodeAttributesBuilder()
	attrs.WithId(nodeId)

	return &DistinctBuilder{
		distinct: &ast.GenDecl{
			Token: ast.DISTINCT,
		},
		a:            attrs,
		distinctType: &ast.DistinctType{},
		ident:        NewIdentifierBuilder(),
	}
}

func (b *DistinctBuilder) WithBaseType(typeNode *ast.TypeInfo) *DistinctBuilder {
	b.distinctType.Value = typeNode
	return b
}

func (b *DistinctBuilder) WithName(name string) *DistinctBuilder {
	b.ident.WithName(name)
	return b
}
func (b *DistinctBuilder) WithIdentifierRange(node *sitter.Node) *DistinctBuilder {
	b.ident.WithSitterPos(node)
	return b
}

func (b *DistinctBuilder) WithInline(inline bool) *DistinctBuilder {
	b.distinctType.IsInline = inline
	return b
}

func (b *DistinctBuilder) WithSitterPos(node *sitter.Node) *DistinctBuilder {
	b.a.WithSitterPos(node)
	return b
}
func (b *DistinctBuilder) Build() *ast.GenDecl {
	spec := &ast.TypeSpec{
		Ident:           b.ident.Build(),
		TypeDescription: b.distinctType,
	}

	b.distinct.Spec = spec
	b.distinct.NodeAttributes = b.a.Build()
	return b.distinct
}
