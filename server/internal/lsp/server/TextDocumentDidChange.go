package server

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	h.state.UpdateDocument(params.TextDocument.URI, params.ContentChanges, h.parser)

	project_state.RefreshDiagnostics(h.state, context.Notify, false)

	return nil
}
