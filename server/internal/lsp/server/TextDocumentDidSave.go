package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (srv *Server) TextDocumentDidSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	srv.RunDiagnostics(srv.state, ctx.Notify, true)
	return nil
}
