package server

import (
	"path/filepath"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) WorkspaceDidChangeWorkspaceFolders(context *glsp.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	if !h.shouldProcessNotification(protocol.MethodWorkspaceDidChangeWorkspaceFolders) {
		return nil
	}
	if params == nil {
		return nil
	}

	h.invalidateProjectRootCache()

	for _, removed := range params.Event.Removed {
		removedRoot := fs.GetCanonicalPath(utilsURIToPath(removed.URI))
		if removedRoot == "" {
			continue
		}

		h.cancelRootIndexing(removedRoot)
		h.clearRootTracking(removedRoot)
		if h.activeConfigRoot == removedRoot {
			h.activeConfigRoot = ""
		}
		if h.state.GetProjectRootURI() == removedRoot {
			h.state.SetProjectRootURI("")
		}
	}

	for _, added := range params.Event.Added {
		addedRoot := fs.GetCanonicalPath(utilsURIToPath(added.URI))
		if addedRoot == "" {
			continue
		}

		if h.state.GetProjectRootURI() == "" {
			h.state.SetProjectRootURI(addedRoot)
		}

		if !isBuildableProjectRoot(addedRoot) {
			continue
		}

		h.configureProjectForRootWithContext(addedRoot, context)
		h.cancelRootIndexing(addedRoot)
		h.indexWorkspaceAtAsync(addedRoot)
	}

	return nil
}

func utilsURIToPath(uri protocol.DocumentUri) string {
	path, err := fs.UriToPath(string(uri))
	if err == nil {
		return path
	}
	return filepath.Clean(string(uri))
}
