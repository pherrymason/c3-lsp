package server

import (
	"os"
	"path/filepath"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) ensureDocumentIndexed(uri protocol.DocumentUri) bool {
	s.ensureWorkspaceIndexedForURI(uri)

	docURI := string(uri)
	if s.state.GetDocument(docURI) != nil {
		return true
	}

	path, err := fs.UriToPath(docURI)
	if err != nil {
		return false
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	doc := document.NewDocumentFromDocURI(docURI, string(content), 1)
	s.state.RefreshDocumentIdentifiers(doc, s.parser)

	return true
}

func (s *Server) ensureWorkspaceIndexedForURI(uri protocol.DocumentUri) {
	if s.indexedRoots == nil {
		s.indexedRoots = make(map[string]bool)
	}

	root := s.resolveProjectRootForURI(&uri)
	if root == "" {
		if path, err := fs.UriToPath(string(uri)); err == nil {
			root = filepath.Dir(path)
		}
	}

	root = fs.GetCanonicalPath(root)
	if root == "" || s.indexedRoots[root] {
		return
	}

	if s.state.GetProjectRootURI() == "" {
		s.state.SetProjectRootURI(root)
	}

	s.configureProjectForRoot(root)
	s.indexWorkspaceAt(root)
	s.indexedRoots[root] = true
}
