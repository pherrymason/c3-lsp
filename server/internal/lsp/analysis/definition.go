package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func GetDefinitionLocation(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) []protocol.Location {

	posContext := getPositionContext(document, pos)

	if posContext.ImportStmt != nil {
		// Cursor is at import statement
		// Get all files locations that contain this module
		locations := getAllModuleLocations(posContext.ImportStmt.(*ast.Import), storage)
		if len(locations) > 0 {
			return locations
		}
	}

	symbol := FindSymbolAtPosition(pos, document.Uri, symbolTable, document.Ast)
	if symbol.IsSome() {
		s := symbol.Get()
		return []protocol.Location{
			{
				URI:   s.URI,
				Range: s.Range.ToProtocol(),
			},
		}
	}

	return []protocol.Location{}
}

func getAllModuleLocations(importStmt *ast.Import, storage *document.Storage) []protocol.Location {
	locations := []protocol.Location{}

	for _, doc := range storage.Documents {
		for _, mod := range doc.Ast.Modules {
			if mod.Name == importStmt.Path.Name {
				locations = append(locations, protocol.Location{
					URI: doc.Uri,
					Range: protocol.Range{
						Start: mod.StartPosition().ToProtocol(),
						End:   mod.EndPosition().ToProtocol(),
					},
				})
			}
		}
	}

	return locations
}
