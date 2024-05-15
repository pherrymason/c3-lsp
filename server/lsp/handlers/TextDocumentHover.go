package handlers

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (h *Handlers) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, ok := h.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	//server.server.Log.Debug(fmt.Sprint("HOVER requested on ", len(doc.Content), params.Position.IndexIn(doc.Content)))
	hoverOption := h.language.FindHoverInformation(doc, params)
	if hoverOption.IsNone() {
		//server.server.Log.Debug(fmt.Sprint("Error trying to find word: ", err))
		return nil, nil
	}

	hover := hoverOption.Get()
	return &hover, nil
}
