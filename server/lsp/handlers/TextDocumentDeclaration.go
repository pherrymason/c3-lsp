package handlers

import (
	_prot "github.com/pherrymason/c3-lsp/lsp/protocol"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Go to declaration"
func (h *Handlers) TextDocumentDeclaration(context *glsp.Context, params *protocol.DeclarationParams) (any, error) {

	doc, ok := h.documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	identifierOption := h.language.FindSymbolDeclarationInWorkspace(doc, symbols.NewPositionFromLSPPosition(params.Position))

	if identifierOption.IsNone() {
		return nil, nil
	}

	indexable := identifierOption.Get()
	return protocol.Location{
		URI:   indexable.GetDocumentURI(),
		Range: _prot.Lsp_NewRangeFromRange(indexable.GetIdRange()),
	}, nil
}
