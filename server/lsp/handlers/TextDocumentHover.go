package handlers

import (
	"github.com/pherrymason/c3-lsp/lsp/symbols"
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
	// TODO improve output by setting language
	// example: {"contents":{"kind":"markdown","value":"```go\nvar server *Server\n```"},"range":{"start":{"line":78,"character":1},"end":{"line":78,"character":7}}}

	pos := symbols.NewPositionFromLSPPosition(params.Position)
	foundSymbolOption := h.language.FindSymbolDeclarationInWorkspace(doc, pos)
	if foundSymbolOption.IsNone() {
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable
	hover := protocol.Hover{
		Contents: protocol.MarkedStringStruct{
			Language: "c3",
			Value:    foundSymbol.GetHoverInfo(),
		},
	}

	return &hover, nil
}
