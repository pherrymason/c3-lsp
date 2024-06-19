package handlers

import (
	"github.com/pherrymason/c3-lsp/lsp/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Handlers) WorkspaceDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return nil
}

func (h *Handlers) WorkspaceDidDeleteFiles(context *glsp.Context, params *protocol.DeleteFilesParams) error {
	for _, file := range params.Files {
		// The file has been removed! update our indices
		docId, _ := utils.NormalizePath(file.URI)
		h.documents.Delete(file.URI)
		h.language.DeleteDocument(docId)
	}

	return nil
}

func (h *Handlers) WorkspaceDidRenameFiles(context *glsp.Context, params *protocol.RenameFilesParams) error {
	for _, file := range params.Files {
		h.documents.Rename(file.OldURI, file.NewURI)

		oldDocId, _ := utils.NormalizePath(file.OldURI)
		newDocId, _ := utils.NormalizePath(file.NewURI)
		h.language.RenameDocument(oldDocId, newDocId)
	}

	return nil
}
