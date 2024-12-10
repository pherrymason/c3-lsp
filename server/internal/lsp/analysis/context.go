package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
)

// PositionContext defines information about what's in current position
type PositionContext struct {
	Pos  lsp.Position
	kind uint // Element type the cursor is at

	IsLiteral          bool
	IsIdentifier       bool
	IsModuleIdentifier bool
	ImportStmt         ast.Node
}
