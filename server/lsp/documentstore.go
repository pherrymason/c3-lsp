package lsp

import (
	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type documentStore struct {
	rootURI   string
	documents map[string]*Document
	fs        fs.FileStorage
	logger    commonlog.Logger
}

func newDocumentStore(fs fs.FileStorage, logger *commonlog.Logger) *documentStore {
	return &documentStore{
		documents: map[string]*Document{},
		fs:        fs,
		logger:    *logger,
	}
}

func (s *documentStore) normalizePath(pathOrUri string) (string, error) {
	path, err := fs.UriToPath(pathOrUri)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse URI: %s", pathOrUri)
	}
	return fs.GetCanonicalPath(path), nil
}

func (s *documentStore) DidOpen(params protocol.DidOpenTextDocumentParams, notify glsp.NotifyFunc, parser *Parser) (*Document, error) {
	langID := params.TextDocument.LanguageID
	if langID != "c3" {
		return nil, nil
	}

	uri := params.TextDocument.URI
	path, err := s.normalizePath(uri)

	if err != nil {
		return nil, err
	}
	doc := &Document{
		parsedTree: GetParsedTreeFromString(params.TextDocument.Text),
		URI:        uri,
		Path:       path,
		Content:    params.TextDocument.Text,
	}

	moduleName := parser.ExtractModuleName(doc)
	doc.ModuleName = moduleName

	s.documents[path] = doc
	return doc, nil
}

func (s *documentStore) Close(uri protocol.DocumentUri) {
	delete(s.documents, uri)
}

func (s *documentStore) Get(pathOrURI string) (*Document, bool) {
	path, err := s.normalizePath(pathOrURI)
	s.logger.Debugf("normalized path:%s", path)

	if err != nil {
		s.logger.Errorf("Could not normalize path: %s", err)
		return nil, false
	}

	d, ok := s.documents[path]
	return d, ok
}
