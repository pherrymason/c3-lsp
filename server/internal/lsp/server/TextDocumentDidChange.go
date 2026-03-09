package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) TextDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	if !s.shouldProcessNotification(protocol.MethodTextDocumentDidChange) {
		return nil
	}
	if params == nil {
		return nil
	}

	docID := s.normalizedDocIDFromURI(params.TextDocument.URI)
	s.state.UpdateDocumentByNormalizedID(docID, params.TextDocument.Version, params.ContentChanges, s.parser)
	if len(params.ContentChanges) == 0 {
		return nil
	}

	notify := noopNotify
	if context != nil {
		notify = context.Notify
	}
	s.RunDiagnosticsQuick(s.state, notify, true, &params.TextDocument.URI)

	return nil
}
