package handlers

import (
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Completion"
// Returns: []CompletionItem | CompletionList | nil
func (h *Handlers) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	doc, ok := h.documents.Get(params.TextDocumentPositionParams.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	suggestions := h.language.BuildCompletionList(doc, symbols.NewPositionFromLSPPosition(params.Position))

	return suggestions, nil
}
