package server

import (
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	h.ensureDocumentIndexed(params.TextDocument.URI)

	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	targetDocURI := doc.URI
	var targetRange protocol.Range
	changes := map[protocol.DocumentUri][]protocol.TextEdit{}

	if moduleTarget, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules); ok {
		oldModuleFullName := moduleTarget.moduleFullName
		if oldModuleFullName == "" {
			oldModuleFullName = moduleTarget.name
		}

		targetRange = moduleTarget.renameRange
		for docID := range h.state.GetAllUnitModules() {
			otherDoc := h.state.GetDocument(string(docID))
			if otherDoc == nil {
				continue
			}

			edits := moduleRenameEdits(otherDoc.SourceCode.Text, oldModuleFullName, oldModuleFullName)
			if len(edits) == 0 {
				continue
			}

			changes[toWorkspaceEditURI(otherDoc.URI, h.options.C3.StdlibPath)] = edits
		}
	} else {
		target, ok := h.symbolRenameTarget(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
		if !ok {
			return nil, nil
		}

		targetDocURI = target.sourceDocURI
		targetRange = target.renameRange
		if targetDocURI == "" {
			targetDocURI = doc.URI
		}

		canUseReferences := canUseReferencesBackedRename(target.declaration)
		if member, ok := target.declaration.(*symbols.StructMember); ok && member != nil {
			if h.moduleHasFunctionNamed(member.GetModuleString(), member.GetName()) {
				canUseReferences = false
			}
		}

		if canUseReferences {
			refs := h.search.FindReferencesInWorkspace(
				targetDocURI,
				symbols.NewPositionFromLSPPosition(target.renameRange.Start),
				h.state,
				params.Context.IncludeDeclaration,
			)
			if len(refs) > 0 {
				return refs, nil
			}
		}

		cache := newRenameExecutionCache()
		changes = h.semanticRenameChanges(target, target.name, cache)
	}
	targetEditURI := toWorkspaceEditURI(targetDocURI, h.options.C3.StdlibPath)

	locations := make([]protocol.Location, 0)
	for uri, edits := range changes {
		for _, edit := range edits {
			if !params.Context.IncludeDeclaration && uri == targetEditURI && rangeEquals(edit.Range, targetRange) {
				continue
			}

			locations = append(locations, protocol.Location{
				URI:   uri,
				Range: edit.Range,
			})
		}
	}

	if len(locations) == 0 {
		return nil, nil
	}

	sort.Slice(locations, func(i, j int) bool {
		if locations[i].URI != locations[j].URI {
			return locations[i].URI < locations[j].URI
		}
		if locations[i].Range.Start.Line != locations[j].Range.Start.Line {
			return locations[i].Range.Start.Line < locations[j].Range.Start.Line
		}
		return locations[i].Range.Start.Character < locations[j].Range.Start.Character
	})

	return locations, nil
}
