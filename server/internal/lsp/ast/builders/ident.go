package ast_builders

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/smacker/go-tree-sitter"
)

// --
// IdentifierBuilder
// --
type IdentifierBuilder struct {
	ident       *ast.Ident
	attrBuilder NodeAttrsBuilder
}

func NewIdentifierBuilder() *IdentifierBuilder {
	return &IdentifierBuilder{
		ident:       &ast.Ident{},
		attrBuilder: *NewNodeAttributesBuilder(),
	}
}

func (i *IdentifierBuilder) WithId(nodeId ast.NodeId) *IdentifierBuilder {
	i.attrBuilder.WithId(nodeId)
	return i
}
func (i *IdentifierBuilder) WithName(name string) *IdentifierBuilder {
	i.ident.Name = name
	return i
}

func (i *IdentifierBuilder) IsCompileTime(ct bool) *IdentifierBuilder {
	i.ident.CompileTime = ct
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
func (i *IdentifierBuilder) Build() *ast.Ident {
	ident := i.ident
	ident.NodeAttributes = i.attrBuilder.Build()

	return ident
}
