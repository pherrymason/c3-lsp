package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Handlers) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	h.state.UpdateDocument(params.TextDocument.URI, params.ContentChanges, h.parser)
	return nil
}
