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

	symbol := identifierOption.Get()
	if !symbol.HasSourceCode() {
		return nil, nil
	}

	return protocol.Location{
		URI:   symbol.GetDocumentURI(),
		Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
	}, nil
}
