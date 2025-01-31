package server

import (
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (srv *Server) WorkspaceDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return nil
}

func (srv *Server) WorkspaceDidDeleteFiles(context *glsp.Context, params *protocol.DeleteFilesParams) error {
	for _, file := range params.Files {
		// The file has been removed! update our indices
		docId := utils.NormalizePath(file.URI)
		//srv.documents.Delete(file.URI)
		srv.state.DeleteDocument(docId)
	}

	return nil
}

func (srv *Server) WorkspaceDidRenameFiles(context *glsp.Context, params *protocol.RenameFilesParams) error {
	for _, file := range params.Files {
		//srv.documents.Rename(file.OldURI, file.NewURI)

		oldDocId := utils.NormalizePath(file.OldURI)
		newDocId := utils.NormalizePath(file.NewURI)
		srv.state.RenameDocument(oldDocId, newDocId)
	}

	return nil
}
