package server

import (
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Go to implementation"
func (h *Server) TextDocumentImplementation(context *glsp.Context, params *protocol.ImplementationParams) (any, error) {
	implementations := h.search.FindImplementationsInWorkspace(
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
		h.state,
	)

	if len(implementations) == 0 {
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
