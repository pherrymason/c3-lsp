package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const linkedEditingWordPatternIdentifier = `[A-Za-z_][A-Za-z0-9_]*`

func (h *Server) TextDocumentLinkedEditingRange(_ *glsp.Context, params *protocol.LinkedEditingRangeParams) (*protocol.LinkedEditingRanges, error) {
	if params == nil {
		return nil, nil
	}

	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	if moduleTarget, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules); ok {
		ranges := linkedEditingRangesForModuleTarget(doc.SourceCode.Text, moduleTarget)
		if len(ranges) == 0 {
			ranges = []protocol.Range{moduleTarget.renameRange}
		}
		return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: linkedEditingWordPatternPtr()}, nil
	}

	target, ok := h.symbolRenameTargetWithTimeout(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
	if !ok {
		return nil, nil
	}

	ranges := h.linkedEditingRangesFromRenameTarget(doc.URI, target)
	if len(ranges) == 0 {
		ranges = []protocol.Range{target.renameRange}
	}

	return &protocol.LinkedEditingRanges{Ranges: ranges, WordPattern: linkedEditingWordPatternPtr()}, nil
}

func linkedEditingWordPatternPtr() *string {
	wordPattern := linkedEditingWordPatternIdentifier
	return &wordPattern
}

func linkedEditingRangesForModuleTarget(source string, target renameTarget) []protocol.Range {
	needle := target.name
	if needle == "" {
		needle = renamePlaceholder(source, target)
	}
	if needle == "" {
		return nil
	}

	return linkedEditingIdentifierRangesInSource(source, needle)
}

func (h *Server) linkedEditingRangesFromRenameTarget(docURI string, target renameTarget) []protocol.Range {
	editsByURI := h.semanticRenameChangesFromReferences(target, target.name)
	if len(editsByURI) == 0 {
		editsByURI = h.semanticRenameChanges(target, target.name, newRenameExecutionCache())
	}

	docID := canonicalPathOrURI(docURI)
	out := make([]protocol.Range, 0, 8)
	for changedURI, edits := range editsByURI {
		if canonicalPathOrURI(string(changedURI)) != docID {
			continue
		}

		for _, edit := range edits {
			out = append(out, edit.Range)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return dedupeAndSortRanges(out)
}

func linkedEditingIdentifierRangesInSource(source string, needle string) []protocol.Range {
	if source == "" || needle == "" {
		return nil
	}

	ranges := make([]protocol.Range, 0, 8)
	posCursor := newLSPPositionCursor(source)
	needleLen := len(needle)

	for i := 0; i+needleLen <= len(source); i++ {
		if source[i:i+needleLen] != needle {
			continue
		}
		if i > 0 && isIdentifierByte(source[i-1]) {
			continue
		}
		end := i + needleLen
		if end < len(source) && isIdentifierByte(source[end]) {
			continue
		}
		if tokenInCommentOrString(source, i) {
			continue
		}

		ranges = append(ranges, protocol.Range{
			Start: posCursor.PositionAt(i),
			End:   posCursor.PositionAt(end),
		})
	}

	return dedupeAndSortRanges(ranges)
}

func dedupeAndSortRanges(in []protocol.Range) []protocol.Range {
	if len(in) == 0 {
		return nil
	}

	result := make([]protocol.Range, 0, len(in))
	seen := map[string]struct{}{}
	for _, r := range in {
		key := rangeKey(r)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Start.Line != result[j].Start.Line {
			return result[i].Start.Line < result[j].Start.Line
		}
		if result[i].Start.Character != result[j].Start.Character {
			return result[i].Start.Character < result[j].Start.Character
		}
		if result[i].End.Line != result[j].End.Line {
			return result[i].End.Line < result[j].End.Line
		}
		return result[i].End.Character < result[j].End.Character
	})

	return result
}

func rangeKey(r protocol.Range) string {
	start := symbols.NewPositionFromLSPPosition(r.Start)
	end := symbols.NewPositionFromLSPPosition(r.End)
	return fmt.Sprintf("%d:%d:%d:%d", start.Line, start.Character, end.Line, end.Character)
}

func canonicalPathOrURI(pathOrURI string) string {
	if strings.HasPrefix(pathOrURI, "file://") {
		path, err := fs.UriToPath(pathOrURI)
		if err == nil {
			return fs.GetCanonicalPath(path)
		}
	}

	return fs.GetCanonicalPath(pathOrURI)
}
