package server

import (
	"path/filepath"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// loadAndIndexFile reads path from disk, constructs a Document, and indexes it
// into the server state.  It returns true if the file was successfully indexed.
// Callers that need to skip already-indexed files should check
// s.state.GetDocument(path) before calling this.
func (s *Server) loadAndIndexFile(path string) bool {
	docs, err := loadSourceDocuments(path)
	if err != nil {
		return false
	}
	indexedAny := false
	for _, doc := range docs {
		if doc.readErr != nil {
			continue
		}
		indexedAny = true
		s.indexFileWithContent(doc.path, []byte(doc.content))
	}

	return indexedAny
}

// indexFileWithContent constructs a Document from already-read content and
// indexes it into the server state.  Use this when the content was already
// read for another purpose (e.g. module-name filtering) to avoid a second
// os.ReadFile call.
func (s *Server) indexFileWithContent(path string, content []byte) {
	doc := document.NewDocumentFromString(path, string(content))
	s.state.RefreshDocumentIdentifiers(&doc, s.parser)
}

// loadFilterAndIndex reads path (plain file or .c3l archive), filters entries
// by their declared module name using acceptModule, and indexes matching entries
// that are not already in state.  Returns the number of entries indexed.
func (s *Server) loadFilterAndIndex(path string, acceptModule func(moduleName string) bool) int {
	entries, err := loadSourceDocuments(path)
	if err != nil {
		return 0
	}
	indexed := 0
	for _, entry := range entries {
		if entry.readErr != nil || s.state.GetDocument(entry.path) != nil {
			continue
		}
		moduleName := extractDeclaredModuleName([]byte(entry.content))
		if moduleName == "" || !acceptModule(moduleName) {
			continue
		}
		s.indexFileWithContent(entry.path, []byte(entry.content))
		indexed++
	}
	return indexed
}

func (s *Server) ensureOpenDocumentParsed(doc *document.Document) bool {
	if doc == nil {
		return false
	}
	docID := doc.URI
	if docID == "" {
		return false
	}
	if s.state.GetUnitModulesByDoc(docID) != nil {
		return true
	}

	s.state.RefreshDocumentIdentifiers(doc, s.parser)
	return s.state.GetUnitModulesByDoc(docID) != nil
}

func (s *Server) ensureDocumentIndexed(uri protocol.DocumentUri) bool {
	return s.ensureDocumentIndexedWithProgress(nil, uri)
}

func (s *Server) ensureDocumentIndexedWithProgress(lspContext *glsp.Context, uri protocol.DocumentUri) bool {
	start := time.Now()
	defer func() {
		if s.server != nil {
			perfLogf(s.server.Log, "ensureDocumentIndexed", start, "uri=%s", uri)
		}
	}()

	s.ensureWorkspaceIndexedForURIWithProgress(lspContext, uri)

	docURI := string(uri)
	if doc := s.state.GetDocument(docURI); doc != nil {
		s.ensureOpenDocumentParsed(doc)
		s.preloadImportedRootModulesForURI(uri)
		return true
	}

	path, err := fs.UriToPath(docURI)
	if err != nil {
		return false
	}

	if !s.loadAndIndexFile(path) {
		return false
	}
	s.preloadImportedRootModulesForURI(uri)

	return true
}

func (s *Server) ensureWorkspaceIndexedForURI(uri protocol.DocumentUri) {
	s.ensureWorkspaceIndexedForURIWithProgress(nil, uri)
}

func (s *Server) ensureWorkspaceIndexedForURIWithProgress(lspContext *glsp.Context, uri protocol.DocumentUri) {
	start := time.Now()
	defer func() {
		if s.server != nil {
			perfLogf(s.server.Log, "ensureWorkspaceIndexedForURI", start, "uri=%s", uri)
		}
	}()

	root := s.resolveProjectRootForURI(&uri)
	if root == "" {
		if path, err := fs.UriToPath(string(uri)); err == nil {
			root = filepath.Dir(path)
		}
	}

	root = fs.GetCanonicalPath(root)
	if root == "" || s.isRootIndexedOrIndexing(root) {
		return
	}
	if !isBuildableProjectRoot(root) {
		return
	}

	if s.state.GetProjectRootURI() == "" {
		s.state.SetProjectRootURI(root)
	}

	s.configureProjectForRoot(root)
	s.indexWorkspaceAtAsyncWithProgress(root, lspContext)
}
