package server

import (
	stdctx "context"

	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Returns: Location | []Location | []LocationLink | nil
func (h *Server) TextDocumentTypeDefinition(context *glsp.Context, params *protocol.TypeDefinitionParams) (any, error) {
	h.ensureDocumentIndexed(params.TextDocument.URI)

	cursorContext := ctx.BuildFromDocumentPosition(params.Position, params.TextDocument.URI, h.state)
	if cursorContext.IsLiteral {
		return nil, nil
	}

	identifierOption := h.search.FindSymbolDeclarationInWorkspace(
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)
	docID := utils.NormalizePath(params.TextDocument.URI)
	pos := symbols.NewPositionFromLSPPosition(params.Position)
	if identifierOption.IsNone() {
		identifierOption = h.resolveSymbolCommonFallbacks(stdctx.Background(), docID, pos)
	}

	if identifierOption.IsNone() {
		return nil, nil
	}

	symbol := identifierOption.Get()
	if isNilIndexable(symbol) {
		return nil, nil
	}
	if !symbol.HasSourceCode() && h.options.C3.StdlibPath.IsNone() {
		return nil, nil
	}

	return protocol.Location{
		URI:   fs.ConvertPathToURI(symbol.GetDocumentURI(), h.options.C3.StdlibPath),
		Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
	}, nil
}
