package server

import (
	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Completion"
// Returns: []CompletionItem | CompletionList | nil
func (srv *Server) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {

	cursorContext := ctx.BuildFromDocumentPosition(
		params.Position,
		utils.NormalizePath(params.TextDocument.URI),
		srv.state,
	)

	suggestions := srv.search.BuildCompletionList(
		cursorContext,
		srv.state,
	)
	return suggestions, nil
}
