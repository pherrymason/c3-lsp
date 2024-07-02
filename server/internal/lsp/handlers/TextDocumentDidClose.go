package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Handlers) TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	h.state.Close(params.TextDocument.URI)
	return nil
}
