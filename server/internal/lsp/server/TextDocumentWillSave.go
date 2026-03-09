package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) TextDocumentWillSave(_ *glsp.Context, params *protocol.WillSaveTextDocumentParams) error {
	if !s.shouldProcessNotification(protocol.MethodTextDocumentWillSave) {
		return nil
	}
	if params == nil {
		return nil
	}

	return nil
}

func (s *Server) TextDocumentWillSaveWaitUntil(ctx *glsp.Context, params *protocol.WillSaveTextDocumentParams) ([]protocol.TextEdit, error) {
	if params == nil {
		return []protocol.TextEdit{}, nil
	}

	if !s.options.Formatting.WillSaveWaitUntil {
		return []protocol.TextEdit{}, nil
	}

	return s.TextDocumentFormatting(ctx, &protocol.DocumentFormattingParams{
		TextDocument: params.TextDocument,
	})
}
