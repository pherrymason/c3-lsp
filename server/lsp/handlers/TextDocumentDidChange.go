package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Handlers) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	doc, ok := h.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil
	}

	doc.ApplyChanges(params.ContentChanges)

	h.language.RefreshDocumentIdentifiers(doc, h.parser)
	return nil
}
