package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentCodeAction(_ *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	if params == nil {
		return []protocol.CodeAction{}, nil
	}

	return []protocol.CodeAction{}, nil
}

func (h *Server) CodeActionResolve(_ *glsp.Context, params *protocol.CodeAction) (*protocol.CodeAction, error) {
	if params == nil {
		return nil, nil
	}

	resolved := *params
	return &resolved, nil
}
