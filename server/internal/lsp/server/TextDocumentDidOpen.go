package server

import (
	"path/filepath"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	/*
		doc, err := h.documents.Open(*params, context.Notify)
		if err != nil {
			//glspServer.Log.Debug("Could not open file document.")
			return err
		}

		if doc != nil {
			h.state.RefreshDocumentIdentifiers(doc, h.parser)
		}
	*/

	langID := params.TextDocument.LanguageID
	if !isC3Document(langID, params.TextDocument.URI) {
		return nil
	}

	doc := document.NewDocumentFromDocURI(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)
	h.state.RefreshDocumentIdentifiers(doc, h.parser)

	return nil
}

func isC3LanguageID(languageID string) bool {
	return strings.EqualFold(languageID, "c3")
}

func isC3Document(languageID string, uri protocol.DocumentUri) bool {
	if isC3LanguageID(languageID) {
		return true
	}

	path, err := fs.UriToPath(string(uri))
	if err != nil {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".c3" || ext == ".c3i" || ext == ".c3t"
}
