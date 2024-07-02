package handlers

import (
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Completion"
// Returns: []CompletionItem | CompletionList | nil
func (h *Handlers) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	suggestions := h.search.BuildCompletionList(
		params.TextDocument.URI,
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)
	return suggestions, nil
}
