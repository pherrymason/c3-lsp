package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (srv *Server) TextDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	srv.state.CloseDocument(params.TextDocument.URI)
	return nil
}
