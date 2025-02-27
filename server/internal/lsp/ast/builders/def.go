package ast_builders

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/smacker/go-tree-sitter"
)

// --
// DefDeclBuilder
// --
type DefDeclBuilder struct {
	def ast.DefDecl
	a   NodeAttrsBuilder
}

func NewDefDeclBuilder(nodeId ast.NodeId) *DefDeclBuilder {
	return &DefDeclBuilder{
		def: ast.DefDecl{
			Ident: NewIdentifierBuilder().Build(),
		},
		a: *NewNodeAttributesBuilder(),
	}
}

func (b *DefDeclBuilder) WithSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.a.WithSitterStartEnd(node.StartPoint(), node.EndPoint())
	return b
}

func (b *DefDeclBuilder) WithName(name string) *DefDeclBuilder {
	b.def.Ident.Name = name
	return b
}
func (b *DefDeclBuilder) WithIdentifierSitterPos(node *sitter.Node) *DefDeclBuilder {
	b.def.Ident.Range.Start = lsp.Position{uint(node.StartPoint().Row), uint(node.StartPoint().Column)}
	b.def.Ident.Range.End = lsp.Position{uint(node.EndPoint().Row), uint(node.EndPoint().Column)}

	return b
}

func (b *DefDeclBuilder) WithExpression(expression ast.Expression) *DefDeclBuilder {
	b.def.Expr = expression
	return b
}

func (b *DefDeclBuilder) Build() ast.DefDecl {
	def := b.def
	def.NodeAttributes = b.a.Build()

	return def
}
