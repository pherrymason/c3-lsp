package server

import (
	stdctx "context"
	"fmt"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type completionItemLabelDetails struct {
	Detail      *string `json:"detail,omitempty"`
	Description *string `json:"description,omitempty"`
}

type completionItemWithLabelDetails struct {
	protocol.CompletionItem
	LabelDetails *completionItemLabelDetails `json:"labelDetails,omitempty"`
}

func completionKindDescription(kind *protocol.CompletionItemKind) *string {
	if kind == nil {
		return nil
	}

	switch *kind {
	case protocol.CompletionItemKindFunction:
		return cast.ToPtr("function")
	case protocol.CompletionItemKindMethod:
		return cast.ToPtr("method")
	case protocol.CompletionItemKindStruct:
		return cast.ToPtr("struct")
	case protocol.CompletionItemKindModule:
		return cast.ToPtr("module")
	case protocol.CompletionItemKindField:
		return cast.ToPtr("field")
	case protocol.CompletionItemKindVariable:
		return cast.ToPtr("variable")
	case protocol.CompletionItemKindKeyword:
		return cast.ToPtr("keyword")
	default:
		return nil
	}
}

func signatureDocumentation(detail string) protocol.MarkupContent {
	return protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: "```c3\n" + detail + "\n```",
	}
}

type structCompletionMode int

const (
	structCompletionNone structCompletionMode = iota
	structCompletionDeclaration
	structCompletionValue
)

func completionDedupCount(items []protocol.CompletionItem) int {
	seen := make(map[string]struct{}, len(items))
	duplicates := 0
	for _, item := range items {
		kind := ""
		if item.Kind != nil {
			kind = fmt.Sprintf("%d", *item.Kind)
		}

		detail := ""
		if item.Detail != nil {
			detail = *item.Detail
		}

		key := item.Label + "|" + kind + "|" + detail
		if _, ok := seen[key]; ok {
			duplicates++
			continue
		}
		seen[key] = struct{}{}
	}

	return duplicates
}

// Support "Completion"
// Returns: []CompletionItem | CompletionList | nil
func (h *Server) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	return h.textDocumentCompletionWithTrace(context, params, "", stdctx.Background())
}

func (h *Server) textDocumentCompletionWithTrace(_ *glsp.Context, params *protocol.CompletionParams, trace string, requestCtx stdctx.Context) (any, error) {
	if requestCtx == nil {
		requestCtx = stdctx.Background()
	}

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	totalStart := time.Now()
	completionCtx := h.buildCompletionContextWithCancel(params, requestCtx)

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	triggerKind, triggerChar := completionTriggerKey(params)
	stateRevision := uint64(0)
	if h.state != nil {
		stateRevision = h.state.Revision()
	}

	cacheKey := completionCacheKey{
		DocURI:         string(params.TextDocument.URI),
		DocVersion:     completionCtx.docVersion,
		StateRevision:  stateRevision,
		Line:           params.Position.Line,
		Character:      params.Position.Character,
		TriggerKind:    triggerKind,
		TriggerChar:    triggerChar,
		SymbolAtCursor: completionCtx.symbolInPosition,
		SnippetSupport: completionCtx.snippetSupport,
	}

	if cached, ok := h.completionCacheGet(cacheKey); ok {
		if h.server != nil {
			perfLogf(
				h.server.Log,
				"textDocument/completion",
				totalStart,
				"phase=total cache_hit=true %s uri=%s line=%d char=%d build_context=%s search_build_list=%s render_items=%s suggestions_in=%d suggestions_out=%d struct_field_lookups=%d dedup_count=%d",
				trace,
				params.TextDocument.URI,
				params.Position.Line,
				params.Position.Character,
				completionCtx.buildContextTime,
				time.Duration(0),
				time.Duration(0),
				len(cached),
				len(cached),
				0,
				0,
			)
		}

		return cached, nil
	}

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	searchStart := time.Now()
	completionCtx.suggestions = h.search.BuildCompletionList(completionCtx.cursor, h.state)
	completionCtx.searchDuration = time.Since(searchStart)

	select {
	case <-requestCtx.Done():
		return nil, nil
	default:
	}

	renderStart := time.Now()
	items, renderStats, cancelled := h.renderCompletionItemsWithStats(completionCtx, requestCtx)
	if cancelled {
		return nil, nil
	}
	renderDuration := time.Since(renderStart)
	h.completionCacheSet(cacheKey, items)

	if h.server != nil {
		perfLogf(
			h.server.Log,
			"textDocument/completion",
			totalStart,
			"phase=total cache_hit=false %s uri=%s line=%d char=%d build_context=%s search_build_list=%s render_items=%s suggestions_in=%d suggestions_out=%d struct_field_lookups=%d dedup_count=%d",
			trace,
			params.TextDocument.URI,
			params.Position.Line,
			params.Position.Character,
			completionCtx.buildContextTime,
			completionCtx.searchDuration,
			renderDuration,
			len(completionCtx.suggestions),
			len(items),
			renderStats.structFieldLookups,
			completionDedupCount(completionCtx.suggestions),
		)
	}

	return items, nil
}
