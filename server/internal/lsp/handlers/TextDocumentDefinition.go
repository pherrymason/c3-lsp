package handlers

import (
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Returns: Location | []Location | []LocationLink | nil
func (h *Handlers) TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	identifierOption := h.search.FindSymbolDeclarationInWorkspace(
		params.TextDocument.URI,
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)

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
