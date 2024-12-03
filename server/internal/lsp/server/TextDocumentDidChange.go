package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (srv *Server) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	srv.state.UpdateDocument(params.TextDocument.URI, params.ContentChanges, srv.parser)

	srv.RunDiagnostics(srv.state, context.Notify, true)

	return nil
}
