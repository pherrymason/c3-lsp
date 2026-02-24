package server

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var moduleNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(::[A-Za-z_][A-Za-z0-9_]*)*$`)

type renameTarget struct {
	name        string
	renameRange protocol.Range
}

func (h *Server) TextDocumentPrepareRename(context *glsp.Context, params *protocol.PrepareRenameParams) (any, error) {
	docURI := utils.NormalizePath(params.TextDocument.URI)
	doc := h.state.GetDocument(docURI)
	if doc == nil {
		return nil, nil
	}

	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	target, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules)
	if !ok {
		return nil, nil
	}

	return target.renameRange, nil
}

func (h *Server) TextDocumentRename(context *glsp.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	if !moduleNamePattern.MatchString(params.NewName) {
		return nil, fmt.Errorf("invalid module name: %s", params.NewName)
	}

	docURI := utils.NormalizePath(params.TextDocument.URI)
	doc := h.state.GetDocument(docURI)
	if doc == nil {
		return nil, nil
	}

	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	target, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules)
	if !ok {
		return nil, nil
	}

	if target.name == params.NewName {
		return &protocol.WorkspaceEdit{}, nil
	}

	changes := map[protocol.DocumentUri][]protocol.TextEdit{}
	for docID := range h.state.GetAllUnitModules() {
		otherDoc := h.state.GetDocument(string(docID))
		if otherDoc == nil {
			continue
		}

		edits := moduleRenameEdits(otherDoc.SourceCode.Text, target.name, params.NewName)
		if len(edits) == 0 {
			continue
		}

		changes[otherDoc.URI] = edits
	}

	return &protocol.WorkspaceEdit{Changes: changes}, nil
}

func moduleRenameTarget(source string, position protocol.Position, unitModules *symbols_table.UnitModules) (renameTarget, bool) {
	if unitModules == nil {
		return renameTarget{}, false
	}

	sourceCode := code.NewSourceCode(source)
	cursorPosition := symbols.NewPositionFromLSPPosition(position)
	word := sourceCode.SymbolInPosition(cursorPosition, unitModules)
	if word.Text() == "" || word.IsSeparator() {
		return renameTarget{}, false
	}

	startIndex := word.FullTextRange().Start.IndexIn(source)
	if !isLikelyModuleOccurrence(source, startIndex, word) {
		return renameTarget{}, false
	}

	name := word.GetFullQualifiedName()
	if name == "" {
		return renameTarget{}, false
	}

	return renameTarget{name: name, renameRange: word.FullTextRange().ToLSP()}, true
}

func isLikelyModuleOccurrence(source string, symbolStartIndex int, word code.Word) bool {
	if word.HasModulePath() {
		return true
	}

	lineStart := strings.LastIndex(source[:symbolStartIndex], "\n") + 1
	linePrefix := strings.TrimSpace(source[lineStart:symbolStartIndex])
	return strings.HasPrefix(linePrefix, "module ") || strings.HasPrefix(linePrefix, "import ")
}

func moduleRenameEdits(source string, oldModule string, newModule string) []protocol.TextEdit {
	pattern := regexp.MustCompile(`(?m)(^|[^A-Za-z0-9_])(` + regexp.QuoteMeta(oldModule) + `)`)
	matches := pattern.FindAllStringSubmatchIndex(source, -1)

	edits := make([]protocol.TextEdit, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		start := match[4]
		end := match[5]
		if end < len(source) {
			next := source[end]
			if isIdentifierByte(next) {
				continue
			}
		}

		edits = append(edits, protocol.TextEdit{
			Range: protocol.Range{
				Start: byteIndexToLSPPosition(source, start),
				End:   byteIndexToLSPPosition(source, end),
			},
			NewText: newModule,
		})
	}

	return edits
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func byteIndexToLSPPosition(content string, index int) protocol.Position {
	if index < 0 {
		index = 0
	}
	if index > len(content) {
		index = len(content)
	}

	line := uint32(0)
	character := uint32(0)

	i := 0
	for i < index {
		r, w := utf8.DecodeRuneInString(content[i:])
		if r == '\n' {
			line++
			character = 0
			i += w
			continue
		}

		if r == utf8.RuneError && w == 1 {
			character++
			i += w
			continue
		}

		character += uint32(len(utf16.Encode([]rune{r})))
		i += w
	}

	return protocol.Position{Line: line, Character: character}
}
