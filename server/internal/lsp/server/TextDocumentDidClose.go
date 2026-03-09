package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	if !h.shouldProcessNotification(protocol.MethodTextDocumentDidClose) {
		return nil
	}
	if params == nil {
		return nil
	}

	docID := h.normalizedDocIDFromURI(params.TextDocument.URI)
	h.state.CloseDocumentByNormalizedID(docID)
	return nil
}
