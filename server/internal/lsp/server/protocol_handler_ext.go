package server

import (
	"encoding/json"
	"errors"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const methodWorkspaceSymbolResolve = "workspaceSymbol/resolve"

type protocolHandlerWithExtensions struct {
	base                   *protocol.Handler
	workspaceSymbolResolve func(context *glsp.Context, params *protocol.SymbolInformation) (*protocol.SymbolInformation, error)
}

func (h *protocolHandlerWithExtensions) Handle(context *glsp.Context) (result any, validMethod bool, validParams bool, err error) {
	if context != nil && context.Method == methodWorkspaceSymbolResolve && h.workspaceSymbolResolve != nil {
		if h.base != nil && !h.base.IsInitialized() {
			return nil, true, true, errors.New("server not initialized")
		}

		var params protocol.SymbolInformation
		if unmarshalErr := json.Unmarshal(context.Params, &params); unmarshalErr != nil {
			return nil, true, false, unmarshalErr
		}
		resolved, resolveErr := h.workspaceSymbolResolve(context, &params)
		return resolved, true, true, resolveErr
	}

	if h.base == nil {
		return nil, false, false, nil
	}

	return h.base.Handle(context)
}
