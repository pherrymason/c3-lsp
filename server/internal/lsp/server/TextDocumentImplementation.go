package server

import (
	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Go to implementation"
func (h *Server) TextDocumentImplementation(context *glsp.Context, params *protocol.ImplementationParams) (any, error) {
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

	if identifierOption.IsNone() {
		return nil, nil
	}

	symbol := identifierOption.Get()

	implementations := h.search.FindImplementationsInWorkspace(
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)

	if len(implementations) == 0 {
		if function, ok := symbol.(*symbols.Function); ok && function.FunctionType() == symbols.UserDefined {
			if !symbol.HasSourceCode() && h.options.C3.StdlibPath.IsNone() {
				return nil, nil
			}

			return []protocol.Location{{
				URI:   fs.ConvertPathToURI(symbol.GetDocumentURI(), h.options.C3.StdlibPath),
				Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
			}}, nil
		}

		return nil, nil
	}

	locations := make([]protocol.Location, 0, len(implementations))
	for _, impl := range implementations {
		if !impl.HasSourceCode() && h.options.C3.StdlibPath.IsNone() {
			continue
		}

		locations = append(locations, protocol.Location{
			URI:   fs.ConvertPathToURI(impl.GetDocumentURI(), h.options.C3.StdlibPath),
			Range: _prot.Lsp_NewRangeFromRange(impl.GetIdRange()),
		})
	}

	if len(locations) == 0 {
		return nil, nil
	}

	return locations, nil
}
