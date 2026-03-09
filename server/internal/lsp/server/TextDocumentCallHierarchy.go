package server

import (
	"fmt"
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentPrepareCallHierarchy(_ *glsp.Context, params *protocol.CallHierarchyPrepareParams) ([]protocol.CallHierarchyItem, error) {
	if params == nil {
		return nil, nil
	}

	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	target, ok := h.symbolRenameTargetWithTimeout(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
	if !ok || indexableIsNil(target.declaration) {
		return nil, nil
	}

	fn, ok := target.declaration.(*symbols.Function)
	if !ok || fn == nil {
		return nil, nil
	}

	return []protocol.CallHierarchyItem{callHierarchyItemFromFunction(fn)}, nil
}

func (h *Server) CallHierarchyIncomingCalls(_ *glsp.Context, params *protocol.CallHierarchyIncomingCallsParams) ([]protocol.CallHierarchyIncomingCall, error) {
	if params == nil {
		return nil, nil
	}

	targetFn := h.resolveCallHierarchyFunction(params.Item)
	if targetFn == nil {
		return nil, nil
	}

	refs := h.search.FindReferencesInWorkspace(
		targetFn.GetDocumentURI(),
		targetFn.GetIdRange().Start,
		h.state,
		false,
	)
	if len(refs) == 0 {
		return nil, nil
	}

	functionsByDoc := h.functionsByDocument()
	type incomingAccum struct {
		fn     *symbols.Function
		ranges []protocol.Range
	}
	accum := map[string]*incomingAccum{}

	for _, ref := range refs {
		caller := smallestFunctionContaining(functionsByDoc[utils.NormalizePath(ref.URI)], symbols.NewPositionFromLSPPosition(ref.Range.Start))
		if caller == nil {
			continue
		}

		key := callHierarchyFunctionKey(caller)
		entry := accum[key]
		if entry == nil {
			entry = &incomingAccum{fn: caller, ranges: []protocol.Range{}}
			accum[key] = entry
		}
		entry.ranges = append(entry.ranges, ref.Range)
	}

	if len(accum) == 0 {
		return nil, nil
	}

	out := make([]protocol.CallHierarchyIncomingCall, 0, len(accum))
	for _, entry := range accum {
		ranges := dedupeAndSortRanges(entry.ranges)
		out = append(out, protocol.CallHierarchyIncomingCall{
			From:       callHierarchyItemFromFunction(entry.fn),
			FromRanges: ranges,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].From.URI != out[j].From.URI {
			return out[i].From.URI < out[j].From.URI
		}
		return out[i].From.Name < out[j].From.Name
	})

	return out, nil
}

func (h *Server) CallHierarchyOutgoingCalls(_ *glsp.Context, params *protocol.CallHierarchyOutgoingCallsParams) ([]protocol.CallHierarchyOutgoingCall, error) {
	if params == nil {
		return nil, nil
	}

	caller := h.resolveCallHierarchyFunction(params.Item)
	if caller == nil {
		return nil, nil
	}

	callerRange := caller.GetDocumentRange()
	callerDocID := utils.NormalizePath(caller.GetDocumentURI())
	functions := h.allWorkspaceFunctions()
	type outgoingAccum struct {
		fn     *symbols.Function
		ranges []protocol.Range
	}
	accum := map[string]*outgoingAccum{}

	for _, callee := range functions {
		if callee == nil {
			continue
		}

		refs := h.search.FindReferencesInWorkspace(
			callee.GetDocumentURI(),
			callee.GetIdRange().Start,
			h.state,
			false,
		)
		if len(refs) == 0 {
			continue
		}

		for _, ref := range refs {
			if utils.NormalizePath(ref.URI) != callerDocID {
				continue
			}
			if !callerRange.HasPosition(symbols.NewPositionFromLSPPosition(ref.Range.Start)) {
				continue
			}

			key := callHierarchyFunctionKey(callee)
			entry := accum[key]
			if entry == nil {
				entry = &outgoingAccum{fn: callee, ranges: []protocol.Range{}}
				accum[key] = entry
			}
			entry.ranges = append(entry.ranges, ref.Range)
		}
	}

	if len(accum) == 0 {
		return nil, nil
	}

	out := make([]protocol.CallHierarchyOutgoingCall, 0, len(accum))
	for _, entry := range accum {
		ranges := dedupeAndSortRanges(entry.ranges)
		out = append(out, protocol.CallHierarchyOutgoingCall{
			To:         callHierarchyItemFromFunction(entry.fn),
			FromRanges: ranges,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].To.URI != out[j].To.URI {
			return out[i].To.URI < out[j].To.URI
		}
		return out[i].To.Name < out[j].To.Name
	})

	return out, nil
}

func (h *Server) resolveCallHierarchyFunction(item protocol.CallHierarchyItem) *symbols.Function {
	functions := h.functionsByDocument()[utils.NormalizePath(item.URI)]
	if len(functions) == 0 {
		return nil
	}

	selection := symbols.NewPositionFromLSPPosition(item.SelectionRange.Start)
	if fn := smallestFunctionContaining(functions, selection); fn != nil {
		return fn
	}

	for _, fn := range functions {
		if fn.GetName() == item.Name {
			return fn
		}
	}

	return nil
}

func (h *Server) allWorkspaceFunctions() []*symbols.Function {
	var all []*symbols.Function
	h.state.ForEachModule(func(module *symbols.Module) {
		all = append(all, module.ChildrenFunctions...)
	})
	return all
}

func (h *Server) functionsByDocument() map[string][]*symbols.Function {
	byDoc := map[string][]*symbols.Function{}
	for _, fn := range h.allWorkspaceFunctions() {
		if fn == nil {
			continue
		}
		docID := utils.NormalizePath(fn.GetDocumentURI())
		byDoc[docID] = append(byDoc[docID], fn)
	}

	return byDoc
}

func smallestFunctionContaining(functions []*symbols.Function, pos symbols.Position) *symbols.Function {
	var best *symbols.Function
	bestScore := uint(^uint(0))
	for _, fn := range functions {
		if fn == nil {
			continue
		}
		rng := fn.GetDocumentRange()
		if !rng.HasPosition(pos) {
			continue
		}
		score := (rng.End.Line-rng.Start.Line)*1000 + (rng.End.Character - rng.Start.Character)
		if best == nil || score < bestScore {
			best = fn
			bestScore = score
		}
	}

	return best
}

func callHierarchyItemFromFunction(fn *symbols.Function) protocol.CallHierarchyItem {
	detail := fn.GetModuleString()
	return protocol.CallHierarchyItem{
		Name:           fn.GetName(),
		Kind:           completionKindToSymbolKind(fn.GetKind()),
		Detail:         &detail,
		URI:            protocol.DocumentUri(fn.GetDocumentURI()),
		Range:          fn.GetDocumentRange().ToLSP(),
		SelectionRange: fn.GetIdRange().ToLSP(),
		Data: map[string]any{
			"fqn": fn.GetFQN(),
		},
	}
}

func callHierarchyFunctionKey(fn *symbols.Function) string {
	if fn == nil {
		return ""
	}
	start := fn.GetIdRange().Start
	return fmt.Sprintf("%s|%s|%d:%d", fn.GetDocumentURI(), fn.GetName(), start.Line, start.Character)
}
