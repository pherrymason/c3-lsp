package server

import (
	"regexp"
	"strings"

	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func moduleRenameTarget(source string, position protocol.Position, unitModules *symbols_table.UnitModules) (renameTarget, bool) {
	if unitModules == nil {
		return renameTarget{}, false
	}

	cursorIndex := symbols.NewPositionFromLSPPosition(position).IndexIn(source)
	if tokenInCommentOrString(source, cursorIndex) {
		return renameTarget{}, false
	}

	sourceCode := code.NewSourceCode(source)
	cursorPosition := symbols.NewPositionFromLSPPosition(position)
	word := sourceCode.SymbolInPosition(cursorPosition, unitModules)
	if word.Text() == "" || word.IsSeparator() {
		return renameTarget{}, false
	}

	startIndex := word.FullTextRange().Start.IndexIn(source)
	if word.HasModulePath() && word.TextRange().HasPosition(cursorPosition) {
		lineStart := strings.LastIndex(source[:startIndex], "\n") + 1
		linePrefix := strings.TrimSpace(source[lineStart:startIndex])
		if linePrefix != "module" && linePrefix != "import" && !strings.HasPrefix(linePrefix, "module ") && !strings.HasPrefix(linePrefix, "import ") {
			return renameTarget{}, false
		}
	}

	if !isLikelyModuleOccurrence(source, startIndex, word) {
		return renameTarget{}, false
	}

	name := word.GetFullQualifiedName()
	if name == "" {
		return renameTarget{}, false
	}

	segmentIndex := len(word.ModulePath())
	if segmentIndex < 0 {
		segmentIndex = 0
	}

	return renameTarget{
		name:               word.Text(),
		renameRange:        word.TextRange().ToLSP(),
		moduleFullName:     name,
		moduleSegmentIndex: segmentIndex,
	}, true
}

func replaceModulePathSegment(moduleFullName string, segmentIndex int, replacement string) (string, bool) {
	if moduleFullName == "" || replacement == "" {
		return "", false
	}
	parts := strings.Split(moduleFullName, "::")
	if segmentIndex < 0 || segmentIndex >= len(parts) {
		return "", false
	}
	parts[segmentIndex] = replacement
	return strings.Join(parts, "::"), true
}

func isLikelyModuleOccurrence(source string, symbolStartIndex int, word code.Word) bool {
	if word.HasModulePath() {
		return true
	}

	lineStart := strings.LastIndex(source[:symbolStartIndex], "\n") + 1
	linePrefix := strings.TrimSpace(source[lineStart:symbolStartIndex])
	return linePrefix == "module" || linePrefix == "import" || strings.HasPrefix(linePrefix, "module ") || strings.HasPrefix(linePrefix, "import ")
}

func moduleRenameEdits(source string, oldModule string, newModule string) []protocol.TextEdit {
	pattern := regexp.MustCompile(`(?m)(^|[^A-Za-z0-9_])(` + regexp.QuoteMeta(oldModule) + `)`)
	matches := pattern.FindAllStringSubmatchIndex(source, -1)
	spans := buildCommentStringSpans(source)
	posCursor := newLSPPositionCursor(source)

	edits := make([]protocol.TextEdit, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		start := match[4]
		end := match[5]
		if byteIndexInSpans(spans, start) {
			continue
		}
		if end < len(source) {
			next := source[end]
			if isIdentifierByte(next) {
				continue
			}
		}

		edits = append(edits, protocol.TextEdit{
			Range: protocol.Range{
				Start: posCursor.PositionAt(start),
				End:   posCursor.PositionAt(end),
			},
			NewText: newModule,
		})
	}

	return edits
}
