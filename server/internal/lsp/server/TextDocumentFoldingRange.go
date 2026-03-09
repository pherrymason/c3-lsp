package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentFoldingRange(_ *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	if params == nil {
		return []protocol.FoldingRange{}, nil
	}

	h.ensureDocumentIndexed(params.TextDocument.URI)
	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil || unitModules == nil {
		return []protocol.FoldingRange{}, nil
	}

	ranges := make([]protocol.FoldingRange, 0)
	for _, module := range unitModules.Modules() {
		if module == nil {
			continue
		}
		ranges = append(ranges, symbolFoldingRanges(module)...)
	}

	ranges = append(ranges, commentFoldingRanges(doc.SourceCode.Text)...)
	if len(ranges) == 0 {
		return []protocol.FoldingRange{}, nil
	}

	ranges = dedupeFoldingRanges(ranges)
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].StartLine != ranges[j].StartLine {
			return ranges[i].StartLine < ranges[j].StartLine
		}
		return ranges[i].EndLine < ranges[j].EndLine
	})

	return ranges, nil
}

func symbolFoldingRanges(item symbols.Indexable) []protocol.FoldingRange {
	if indexableIsNil(item) {
		return nil
	}

	ranges := make([]protocol.FoldingRange, 0)
	if fold, ok := indexableToFoldingRange(item); ok {
		ranges = append(ranges, fold)
	}

	for _, child := range outlineChildren(item) {
		ranges = append(ranges, symbolFoldingRanges(child)...)
	}

	return ranges
}

func indexableToFoldingRange(item symbols.Indexable) (protocol.FoldingRange, bool) {
	r := item.GetDocumentRange()
	if r.End.Line <= r.Start.Line {
		return protocol.FoldingRange{}, false
	}

	kind := string(protocol.FoldingRangeKindRegion)
	return protocol.FoldingRange{
		StartLine: protocol.UInteger(r.Start.Line),
		EndLine:   protocol.UInteger(r.End.Line),
		Kind:      &kind,
	}, true
}

func commentFoldingRanges(source string) []protocol.FoldingRange {
	if source == "" {
		return nil
	}

	ranges := make([]protocol.FoldingRange, 0)
	line := 0
	for i := 0; i < len(source)-1; i++ {
		if source[i] == '\n' {
			line++
			continue
		}

		open := ""
		close := ""
		switch {
		case source[i] == '/' && source[i+1] == '*':
			open = "/*"
			close = "*/"
		case source[i] == '<' && source[i+1] == '*':
			open = "<*"
			close = "*>"
		default:
			continue
		}

		_ = open
		endRel := strings.Index(source[i+2:], close)
		if endRel < 0 {
			continue
		}

		end := i + 2 + endRel + len(close)
		block := source[i:end]
		endLine := line + strings.Count(block, "\n")
		if endLine > line {
			kind := string(protocol.FoldingRangeKindComment)
			ranges = append(ranges, protocol.FoldingRange{
				StartLine: protocol.UInteger(line),
				EndLine:   protocol.UInteger(endLine),
				Kind:      &kind,
			})
		}

		line = endLine
		i = end - 1
	}

	return ranges
}

func dedupeFoldingRanges(ranges []protocol.FoldingRange) []protocol.FoldingRange {
	seen := map[string]struct{}{}
	out := make([]protocol.FoldingRange, 0, len(ranges))
	for _, r := range ranges {
		key := fmt.Sprintf("%d:%d", r.StartLine, r.EndLine)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	return out
}
