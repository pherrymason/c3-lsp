package factory

import "github.com/pherrymason/c3-lsp/internal/lsp/ast"

type NullIDGenerator struct{}

func (NullIDGenerator) GenerateID() ast.NodeId { return 0 }

func newTestAstConverter() *ASTConverter {
	c := &ASTConverter{
		idGenerator: &NullIDGenerator{},
	}

	return c
}
