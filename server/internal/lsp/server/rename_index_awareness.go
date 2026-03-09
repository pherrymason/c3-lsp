package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) maybeWarnPartialWorkspaceIndexForRename(context *glsp.Context, uri protocol.DocumentUri) {
	root := fs.GetCanonicalPath(s.resolveProjectRootForURI(&uri))
	if root == "" {
		return
	}

	if !s.shouldWarnPartialWorkspaceIndexForRename(root) {
		return
	}

	message := "C3-LSP rename may be incomplete: workspace has not been fully indexed yet"
	if s.isRootIndexedOrIndexing(root) {
		message = "C3-LSP rename may be incomplete: workspace indexing is still in progress"
	}

	if s.server != nil {
		s.server.Log.Warning(message, "root", root)
	}
	s.notifyWindowLogMessage(context, protocol.MessageTypeWarning, fmt.Sprintf("%s (%s)", message, root))
}

func (s *Server) shouldWarnPartialWorkspaceIndexForRename(root string) bool {
	if root == "" {
		return false
	}

	s.idx.mu.Lock()
	defer s.idx.mu.Unlock()
	s.ensureIndexingStateMapsLocked()
	if s.renameWarningRoots == nil {
		s.renameWarningRoots = make(map[string]bool)
	}

	if s.idx.indexed[root] {
		delete(s.renameWarningRoots, root)
		return false
	}

	if s.renameWarningRoots[root] {
		return false
	}

	s.renameWarningRoots[root] = true
	return true
}
