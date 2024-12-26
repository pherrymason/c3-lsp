package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go/token"
)

type Location struct {
	Uri   protocol.URI
	Range protocol.Range
}

func PositionInNode(node ast.Node, pos lsp.Position) bool {
	char := pos.Column
	line := pos.Line

	return node != nil &&
		node.StartPosition().Column <= char &&
		node.StartPosition().Line <= line &&
		node.EndPosition().Column >= char &&
		node.EndPosition().Line >= line
}

func getPositionContext(document *document.Document, pos lsp.Position) PositionContext {
	posContext := PositionContext{
		Pos: pos,
	}

	for _, mod := range document.Ast.Modules {
		for _, include := range mod.Imports {
			if PositionInNode(include, pos) {
				posContext.ImportStmt = include
			}
		}
	}

	return posContext
}

func FindNodeAtPosition(n ast.Node, fset *token.FileSet, pos lsp.Position) ast.Node {
	if n == nil {
		return nil
	}

	// Convertimos la posición del nodo a coordenadas (línea y columna)
	start := n.StartPosition()
	end := n.EndPosition()

	// Verificamos si la posición está dentro del rango del nodo
	if (start.Line < pos.Line || (start.Line == pos.Line && start.Column <= pos.Column)) &&
		(end.Line > pos.Line || (end.Line == pos.Line && end.Column >= pos.Column)) {

	}

	return n
}
