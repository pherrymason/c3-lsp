package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func BuildCompletionList(document *document.Document, pos lsp.Position, storage *document.Storage, symbolTable *SymbolTable) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	return items
}
