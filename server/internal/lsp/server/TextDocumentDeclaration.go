package server

import (
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Go to declaration"
func (h *Server) TextDocumentDeclaration(context *glsp.Context, params *protocol.DeclarationParams) (any, error) {
	identifierOption := h.search.FindSymbolDeclarationInWorkspace(
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)

	if identifierOption.IsNone() {
		return nil, nil
	}

	symbol := identifierOption.Get()
	if !symbol.HasSourceCode() && h.options.C3.StdlibPath.IsNone() {
		return nil, nil
	}

	return protocol.Location{
		URI:   fs.ConvertPathToURI(symbol.GetDocumentURI(), h.options.C3.StdlibPath),
		Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
	}, nil
}
