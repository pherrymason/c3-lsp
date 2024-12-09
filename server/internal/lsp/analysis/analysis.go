package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type PositionContext struct {
	Pos  lsp.Position
	kind uint // Element type the cursor is at
}

type Location struct {
	Uri   protocol.URI
	Range protocol.Range
}

func PositionInNode(node ast.Node, pos protocol.Position) bool {
	char := uint(pos.Character)
	line := uint(pos.Line)

	return node != nil &&
		node.StartPosition().Column >= char &&
		node.StartPosition().Line >= line &&
		node.EndPosition().Column <= char &&
		node.EndPosition().Line <= line
}
