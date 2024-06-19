package document

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DocumentStore struct {
	RootURI   string
	documents map[string]*Document
	fs        fs.FileStorage
	logger    commonlog.Logger
}

func NewDocumentStore(fs fs.FileStorage, logger *commonlog.Logger) *DocumentStore {
	return &DocumentStore{
		documents: map[string]*Document{},
		fs:        fs,
		logger:    *logger,
	}
}

func (s *DocumentStore) normalizePath(pathOrUri string) (string, error) {
	path, err := fs.UriToPath(pathOrUri)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse URI: %s", pathOrUri)
	}
	return fs.GetCanonicalPath(path), nil
}

func (s *DocumentStore) Open(params protocol.DidOpenTextDocumentParams, notify glsp.NotifyFunc) (*Document, error) {
	langID := params.TextDocument.LanguageID
	if langID != "c3" {
		return nil, nil
	}

	uri := params.TextDocument.URI
	path, err := s.normalizePath(uri)
	s.logger.Debug(fmt.Sprintf("Opening %s :: %s", uri, path))

	if err != nil {
		return nil, err
	}
	// TODO test that document is created with the normalized path and not params.TextDocument.URI
	doc := NewDocumentFromString(path, params.TextDocument.Text)

	s.documents[path] = &doc
	return &doc, nil
}

func (s *DocumentStore) Close(uri protocol.DocumentUri) {
	delete(s.documents, uri)
}

func (s *DocumentStore) Get(pathOrURI string) (*Document, bool) {
	path, err := s.normalizePath(pathOrURI)
	s.logger.Debugf("normalized path:%s", path)

	if err != nil {
		s.logger.Errorf("Could not normalize path: %s", err)
		return nil, false
	}

	d, ok := s.documents[path]
	return d, ok
}

func (s *DocumentStore) Delete(pathOrURI string) {
	path, err := s.normalizePath(pathOrURI)
	s.logger.Debugf("normalized path:%s", path)

	if err != nil {
		s.logger.Errorf("Could not normalize path: %s", err)
		return
	}

	delete(s.documents, path)
}
