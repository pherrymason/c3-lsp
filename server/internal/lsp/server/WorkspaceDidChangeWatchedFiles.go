package server

import (
	"path/filepath"
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) WorkspaceDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	if !h.shouldProcessNotification(protocol.MethodWorkspaceDidChangeWatchedFiles) {
		return nil
	}
	if params == nil {
		return nil
	}

	shouldReloadConfig := false
	reloadRoots := map[string]struct{}{}
	reindexRoots := map[string]struct{}{}

	for _, change := range params.Changes {
		docID := utils.NormalizePath(change.URI)
		if isWorkspaceConfigFile(docID) {
			shouldReloadConfig = true
			reloadRoots[filepath.Dir(docID)] = struct{}{}
			continue
		}

		if isC3SourceFile(docID) {
			if isC3LibraryArchive(docID) {
				h.dropIndexedArchiveEntries(docID)
			}
			if root := h.resolveProjectRootForPath(docID); root != "" {
				reindexRoots[root] = struct{}{}
			}
		}

		if change.Type == protocol.FileChangeTypeDeleted {
			h.state.DeleteDocument(docID)
			continue
		}

		if (change.Type == protocol.FileChangeTypeCreated || change.Type == protocol.FileChangeTypeChanged) && isPlainC3SourceFile(docID) {
			h.loadAndIndexFile(docID)
		}
	}

	if shouldReloadConfig {
		for _, root := range sortedRootKeys(reloadRoots) {
			h.reloadWorkspaceConfiguration(context, root, "workspace/didChangeWatchedFiles")
		}
	}

	reindexedAny := false
	for _, root := range sortedRootKeys(reindexRoots) {
		if _, reloaded := reloadRoots[root]; reloaded {
			continue
		}
		if !isBuildableProjectRoot(root) {
			continue
		}

		h.cancelRootIndexing(root)
		h.indexWorkspaceAtAsync(root)
		reindexedAny = true
	}

	if reindexedAny && context != nil {
		h.RunDiagnosticsFull(h.state, context.Notify, false)
	}

	return nil
}

func isWorkspaceConfigFile(path string) bool {
	base := filepath.Base(path)
	return base == "c3lsp.json" || base == "project.json"
}

func isC3SourceFile(path string) bool {
	return isC3SourcePath(path)
}

func sortedRootKeys(roots map[string]struct{}) []string {
	out := make([]string, 0, len(roots))
	for root := range roots {
		if root == "" {
			continue
		}
		out = append(out, root)
	}
	sort.Strings(out)
	return out
}

func (h *Server) reloadWorkspaceConfiguration(context *glsp.Context, root string, source string) {
	if root == "" {
		return
	}

	h.server.Log.Info("reloading config", "source", source, "root", root)
	h.invalidateProjectRootCache()
	h.activeConfigRoot = ""
	h.configureProjectForRootWithContext(root, context)

	projectRoot := h.state.GetProjectRootURI()
	if isBuildableProjectRoot(projectRoot) {
		canonicalRoot := fs.GetCanonicalPath(projectRoot)
		h.cancelRootIndexing(canonicalRoot)
		h.indexWorkspaceAtAsync(canonicalRoot)
		if context != nil {
			h.RunDiagnosticsFull(h.state, context.Notify, false)
		}
	} else {
		h.server.Log.Info("skipped indexing: aggregate root", "source", source)
	}
}

func (h *Server) WorkspaceDidDeleteFiles(context *glsp.Context, params *protocol.DeleteFilesParams) error {
	if !h.shouldProcessNotification(protocol.MethodWorkspaceDidDeleteFiles) {
		return nil
	}
	if params == nil {
		return nil
	}

	for _, file := range params.Files {
		// The file has been removed! update our indices
		docId := utils.NormalizePath(file.URI)
		if isC3LibraryArchive(docId) {
			h.dropIndexedArchiveEntries(docId)
		}
		//h.documents.Delete(file.URI)
		h.state.DeleteDocument(docId)
	}
	h.invalidateProjectRootCache()

	return nil
}

func (h *Server) WorkspaceWillCreateFiles(context *glsp.Context, params *protocol.CreateFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (h *Server) WorkspaceDidCreateFiles(context *glsp.Context, params *protocol.CreateFilesParams) error {
	if !h.shouldProcessNotification(protocol.MethodWorkspaceDidCreateFiles) {
		return nil
	}
	if params == nil {
		return nil
	}

	for _, file := range params.Files {
		docId := utils.NormalizePath(file.URI)
		if !isPlainC3SourceFile(docId) {
			continue
		}
		h.loadAndIndexFile(docId)
	}
	h.invalidateProjectRootCache()

	return nil
}

func (h *Server) WorkspaceWillRenameFiles(context *glsp.Context, params *protocol.RenameFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}

func (h *Server) WorkspaceDidRenameFiles(context *glsp.Context, params *protocol.RenameFilesParams) error {
	if !h.shouldProcessNotification(protocol.MethodWorkspaceDidRenameFiles) {
		return nil
	}
	if params == nil {
		return nil
	}

	for _, file := range params.Files {
		//h.documents.Rename(file.OldURI, file.NewURI)

		oldDocId := utils.NormalizePath(file.OldURI)
		newDocId := utils.NormalizePath(file.NewURI)
		if isC3LibraryArchive(oldDocId) {
			h.dropIndexedArchiveEntries(oldDocId)
		}
		h.state.RenameDocument(oldDocId, newDocId)
	}
	h.invalidateProjectRootCache()

	return nil
}

func (h *Server) WorkspaceWillDeleteFiles(context *glsp.Context, params *protocol.DeleteFilesParams) (*protocol.WorkspaceEdit, error) {
	return nil, nil
}
