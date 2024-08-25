package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	s.state.UpdateDocument(params.TextDocument.URI, params.ContentChanges, s.parser)

	s.RefreshDiagnostics(s.state, context.Notify, false)

	return nil
}