package server

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis"
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/featureflags"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Returns: Location | []Location | []LocationLink | nil
func (srv *Server) TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	if featureflags.IsActive(featureflags.UseGeneratedAST) {
		doc, _ := srv.documents.GetDocument(params.TextDocument.URI)
		locations := analysis.GetDefinitionLocation(doc, lsp.NewLSPPosition(params.Position), srv.documents, srv.symbolTable)
		return locations, nil
	}

	identifierOption := srv.search.FindSymbolDeclarationInWorkspace(
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
		srv.state,
	)

	if identifierOption.IsNone() {
		return nil, nil
	}

	symbol := identifierOption.Get()
	if !symbol.HasSourceCode() && srv.options.C3.StdlibPath.IsNone() {
		return nil, nil
	}

	return protocol.Location{
		URI:   fs.ConvertPathToURI(symbol.GetDocumentURI(), srv.options.C3.StdlibPath),
		Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
	}, nil
}
