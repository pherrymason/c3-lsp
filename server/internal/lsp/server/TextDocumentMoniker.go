package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const monikerSchemeC3 = "c3"

func (h *Server) TextDocumentMoniker(_ *glsp.Context, params *protocol.MonikerParams) ([]protocol.Moniker, error) {
	if params == nil {
		return nil, nil
	}

	doc, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	if moduleTarget, ok := moduleRenameTarget(doc.SourceCode.Text, params.Position, unitModules); ok {
		identifier := moduleTarget.moduleFullName
		if identifier == "" {
			identifier = moduleTarget.name
		}
		kind := protocol.MonikerKindExport
		return []protocol.Moniker{{
			Scheme:     monikerSchemeC3,
			Identifier: fmt.Sprintf("module:%s", identifier),
			Unique:     protocol.UniquenessLevelProject,
			Kind:       &kind,
		}}, nil
	}

	target, ok := h.symbolRenameTargetWithTimeout(doc.URI, doc.SourceCode.Text, params.Position, unitModules)
	if !ok || indexableIsNil(target.declaration) {
		return nil, nil
	}

	kind, unique := monikerKindAndUniqueness(target.declaration)
	return []protocol.Moniker{{
		Scheme:     monikerSchemeC3,
		Identifier: monikerIdentifier(target.declaration),
		Unique:     unique,
		Kind:       &kind,
	}}, nil
}

func monikerKindAndUniqueness(declaration symbols.Indexable) (protocol.MonikerKind, protocol.UniquenessLevel) {
	if indexableIsNil(declaration) {
		return protocol.MonikerKindLocal, protocol.UniquenessLevelDocument
	}
	if declaration.IsLocal() {
		return protocol.MonikerKindLocal, protocol.UniquenessLevelDocument
	}
	if !declaration.HasSourceCode() {
		return protocol.MonikerKindImport, protocol.UniquenessLevelScheme
	}

	return protocol.MonikerKindExport, protocol.UniquenessLevelProject
}

func monikerIdentifier(declaration symbols.Indexable) string {
	if indexableIsNil(declaration) {
		return ""
	}

	idRange := declaration.GetIdRange()
	return fmt.Sprintf(
		"%s|%s|%d|%d:%d:%d:%d",
		declaration.GetModuleString(),
		declaration.GetName(),
		declaration.GetKind(),
		idRange.Start.Line,
		idRange.Start.Character,
		idRange.End.Line,
		idRange.End.Character,
	)
}
