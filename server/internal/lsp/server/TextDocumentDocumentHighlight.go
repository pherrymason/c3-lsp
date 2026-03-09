package server

import (
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentDocumentHighlight(context *glsp.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	if params == nil {
		return nil, nil
	}

	refs, err := h.TextDocumentReferences(context, &protocol.ReferenceParams{
		TextDocumentPositionParams: params.TextDocumentPositionParams,
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})
	if err != nil || len(refs) == 0 {
		return nil, err
	}

	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	targetRange := protocol.Range{}
	if moduleTarget, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules); ok {
		targetRange = moduleTarget.renameRange
	} else if target, ok := h.symbolRenameTarget(doc.URI, doc.SourceCode.Text, params.Position, unitModules); ok {
		targetRange = target.renameRange
	}

	currentDoc := utils.NormalizePath(params.TextDocument.URI)
	highlights := make([]protocol.DocumentHighlight, 0, len(refs))
	for _, loc := range refs {
		if utils.NormalizePath(loc.URI) != currentDoc {
			continue
		}

		kind := protocol.DocumentHighlightKindRead
		if rangeEquals(loc.Range, targetRange) {
			kind = protocol.DocumentHighlightKindWrite
		}

		highlights = append(highlights, protocol.DocumentHighlight{
			Range: loc.Range,
			Kind:  &kind,
		})
	}

	if len(highlights) == 0 {
		return nil, nil
	}

	sort.Slice(highlights, func(i, j int) bool {
		if highlights[i].Range.Start.Line != highlights[j].Range.Start.Line {
			return highlights[i].Range.Start.Line < highlights[j].Range.Start.Line
		}
		return highlights[i].Range.Start.Character < highlights[j].Range.Start.Character
	})

	return highlights, nil
}
