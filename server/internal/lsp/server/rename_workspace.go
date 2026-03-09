package server

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) workspaceEditFromChanges(changes map[protocol.DocumentUri][]protocol.TextEdit) *protocol.WorkspaceEdit {
	if changes == nil {
		changes = map[protocol.DocumentUri][]protocol.TextEdit{}
	}

	edit := &protocol.WorkspaceEdit{Changes: changes}
	if !h.clientSupportsDocumentChanges() || len(changes) == 0 {
		return edit
	}

	edit.DocumentChanges = workspaceDocumentChanges(changes)
	return edit
}

func (h *Server) clientSupportsDocumentChanges() bool {
	if h == nil || h.clientCapabilities.Workspace == nil || h.clientCapabilities.Workspace.WorkspaceEdit == nil || h.clientCapabilities.Workspace.WorkspaceEdit.DocumentChanges == nil {
		return false
	}

	return *h.clientCapabilities.Workspace.WorkspaceEdit.DocumentChanges
}

func workspaceDocumentChanges(changes map[protocol.DocumentUri][]protocol.TextEdit) []any {
	uris := make([]protocol.DocumentUri, 0, len(changes))
	for uri := range changes {
		uris = append(uris, uri)
	}

	sort.Slice(uris, func(i, j int) bool {
		return uris[i] < uris[j]
	})

	result := make([]any, 0, len(uris))
	for _, uri := range uris {
		edits := changes[uri]
		editPayload := make([]any, 0, len(edits))
		for _, edit := range edits {
			editPayload = append(editPayload, edit)
		}

		result = append(result, protocol.TextDocumentEdit{
			TextDocument: protocol.OptionalVersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
				Version:                nil,
			},
			Edits: editPayload,
		})
	}

	return result
}

func (h *Server) semanticRenameChangesFromReferences(target renameTarget, newName string) map[protocol.DocumentUri][]protocol.TextEdit {
	if !canUseReferencesBackedRename(target.declaration) {
		return map[protocol.DocumentUri][]protocol.TextEdit{}
	}
	if member, ok := target.declaration.(*symbols.StructMember); ok && member != nil {
		if h.moduleHasFunctionNamed(member.GetModuleString(), member.GetName()) {
			return map[protocol.DocumentUri][]protocol.TextEdit{}
		}
	}

	targetDocURI := target.sourceDocURI
	if targetDocURI == "" && !indexableIsNil(target.declaration) {
		targetDocURI = target.declaration.GetDocumentURI()
	}
	if targetDocURI == "" {
		return map[protocol.DocumentUri][]protocol.TextEdit{}
	}

	references := h.search.FindReferencesInWorkspace(
		targetDocURI,
		symbols.NewPositionFromLSPPosition(target.renameRange.Start),
		h.state,
		true,
	)
	if len(references) == 0 {
		return map[protocol.DocumentUri][]protocol.TextEdit{}
	}

	changes := map[protocol.DocumentUri][]protocol.TextEdit{}
	for _, ref := range references {
		changes[ref.URI] = append(changes[ref.URI], protocol.TextEdit{
			Range:   ref.Range,
			NewText: newName,
		})
	}

	return changes
}

func toWorkspaceEditURI(pathOrURI string, stdlibPath option.Option[string]) protocol.DocumentUri {
	if strings.HasPrefix(pathOrURI, "file://") {
		return protocol.DocumentUri(pathOrURI)
	}

	return protocol.DocumentUri(fs.ConvertPathToURI(pathOrURI, stdlibPath))
}

func emptyWorkspaceEdit() *protocol.WorkspaceEdit {
	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentUri][]protocol.TextEdit{},
	}
}

func dedupeTextEdits(edits []protocol.TextEdit) []protocol.TextEdit {
	if len(edits) < 2 {
		return edits
	}

	unique := make(map[string]protocol.TextEdit, len(edits))
	for _, edit := range edits {
		key := fmt.Sprintf("%d:%d-%d:%d", edit.Range.Start.Line, edit.Range.Start.Character, edit.Range.End.Line, edit.Range.End.Character)
		unique[key] = edit
	}

	out := make([]protocol.TextEdit, 0, len(unique))
	for _, edit := range unique {
		out = append(out, edit)
	}

	return out
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
