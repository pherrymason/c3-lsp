package server

import (
	"github.com/pherrymason/c3-lsp/pkg/featureflags"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (srv *Server) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {

	if featureflags.IsActive(featureflags.UseGeneratedAST) {
		srv.workspace.UpdateDocument(params.TextDocument.URI, params.ContentChanges, uint(params.TextDocument.Version))
		return nil
	}

	srv.state.UpdateDocument(params.TextDocument.URI, params.ContentChanges, srv.parser)
	srv.RunDiagnostics(srv.state, context.Notify, true)

	return nil
}
