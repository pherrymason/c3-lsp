package server

import (
	stdctx "context"
	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	_prot "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"time"
)

// Returns: Location | []Location | []LocationLink | nil
func (h *Server) TextDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	return h.textDocumentDefinitionWithTrace(context, params, "", stdctx.Background())
}

func (h *Server) textDocumentDefinitionWithTrace(context *glsp.Context, params *protocol.DefinitionParams, trace string, requestCtx stdctx.Context) (any, error) {
	if requestCtx == nil {
		requestCtx = stdctx.Background()
	}

	start := time.Now()
	ensureDuration := time.Duration(0)
	contextDuration := time.Duration(0)
	resolveDuration := time.Duration(0)
	defer func() {
		if h.server != nil {
			perfLogf(
				h.server.Log,
				"textDocument/definition",
				start,
				"phase=total %s uri=%s line=%d char=%d ensure=%s build_context=%s resolve=%s",
				trace,
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
				ensureDuration,
				contextDuration,
				resolveDuration,
			)
		}
	}()

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	ensureStart := time.Now()
	h.ensureDocumentIndexed(params.TextDocument.URI)
	ensureDuration = time.Since(ensureStart)

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	ctxStart := time.Now()
	cursorContext := ctx.BuildFromDocumentPosition(params.Position, params.TextDocument.URI, h.state)
	contextDuration = time.Since(ctxStart)
	if cursorContext.IsLiteral {
		return nil, nil
	}

	resolveStart := time.Now()
	identifierOption := h.findSymbolDeclarationWithContext(
		requestCtx,
		utils.NormalizePath(params.TextDocument.URI),
		symbols.NewPositionFromLSPPosition(params.Position),
	)
	docID := utils.NormalizePath(params.TextDocument.URI)
	pos := symbols.NewPositionFromLSPPosition(params.Position)
	if identifierOption.IsNone() {
		identifierOption = h.resolveSymbolCommonFallbacks(requestCtx, docID, pos)
	}

	if identifierOption.IsNone() {
		resolveDuration = time.Since(resolveStart)
		return nil, nil
	}

	symbol := identifierOption.Get()
	if isNilIndexable(symbol) {
		resolveDuration = time.Since(resolveStart)
		return nil, nil
	}
	if !symbol.HasSourceCode() && h.options.C3.StdlibPath.IsNone() {
		resolveDuration = time.Since(resolveStart)
		return nil, nil
	}

	resolveDuration = time.Since(resolveStart)

	return protocol.Location{
		URI:   fs.ConvertPathToURI(symbol.GetDocumentURI(), h.options.C3.StdlibPath),
		Range: _prot.Lsp_NewRangeFromRange(symbol.GetIdRange()),
	}, nil
}
