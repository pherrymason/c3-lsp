package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Handlers) TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	doc, err := h.documents.Open(*params, context.Notify)
	if err != nil {
		//glspServer.Log.Debug("Could not open file document.")
		return err
	}

	if doc != nil {
		h.language.RefreshDocumentIdentifiers(doc, h.parser)
	}

	return nil
}
