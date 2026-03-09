package server

import (
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentSelectionRange(_ *glsp.Context, params *protocol.SelectionRangeParams) ([]protocol.SelectionRange, error) {
	if params == nil {
		return []protocol.SelectionRange{}, nil
	}

	h.ensureDocumentIndexed(params.TextDocument.URI)
	_, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if unitModules == nil {
		return []protocol.SelectionRange{}, nil
	}

	result := make([]protocol.SelectionRange, 0, len(params.Positions))
	for _, pos := range params.Positions {
		selection, ok := selectionRangeAtPosition(unitModules, symbols.NewPositionFromLSPPosition(pos))
		if !ok {
			result = append(result, protocol.SelectionRange{Range: protocol.Range{Start: pos, End: pos}})
			continue
		}
		result = append(result, selection)
	}

	return result, nil
}

func selectionRangeAtPosition(unitModules *symbols_table.UnitModules, position symbols.Position) (protocol.SelectionRange, bool) {
	if unitModules == nil {
		return protocol.SelectionRange{}, false
	}

	containing := make([]symbols.Range, 0)
	for _, module := range unitModules.Modules() {
		collectContainingRanges(module, position, &containing)
	}
	if len(containing) == 0 {
		return protocol.SelectionRange{}, false
	}

	containing = dedupeSymbolRanges(containing)
	sort.Slice(containing, func(i, j int) bool {
		left, right := containing[i], containing[j]
		if left.Start.Line != right.Start.Line {
			return left.Start.Line > right.Start.Line
		}
		if left.Start.Character != right.Start.Character {
			return left.Start.Character > right.Start.Character
		}
		if left.End.Line != right.End.Line {
			return left.End.Line < right.End.Line
		}
		return left.End.Character < right.End.Character
	})

	var parent *protocol.SelectionRange
	for i := len(containing) - 1; i >= 0; i-- {
		node := protocol.SelectionRange{
			Range:  containing[i].ToLSP(),
			Parent: parent,
		}
		parent = &node
	}

	if parent == nil {
		return protocol.SelectionRange{}, false
	}

	return *parent, true
}

func collectContainingRanges(item symbols.Indexable, position symbols.Position, out *[]symbols.Range) {
	if indexableIsNil(item) {
		return
	}

	r := item.GetDocumentRange()
	if r.HasPosition(position) {
		*out = append(*out, r)
	}

	for _, child := range outlineChildren(item) {
		collectContainingRanges(child, position, out)
	}
}

func dedupeSymbolRanges(ranges []symbols.Range) []symbols.Range {
	seen := map[[4]uint]struct{}{}
	out := make([]symbols.Range, 0, len(ranges))
	for _, r := range ranges {
		key := [4]uint{r.Start.Line, r.Start.Character, r.End.Line, r.End.Character}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	return out
}
