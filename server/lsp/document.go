package lsp

import (
	"github.com/pherrymason/c3-lsp/utils"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type documentStore struct {
	documents map[string]*document
	fs        utils.FileStorage
	logger    commonlog.Logger
}

func newDocumentStore(fs utils.FileStorage, logger *commonlog.Logger) *documentStore {
	return &documentStore{
		documents: map[string]*document{},
		fs:        fs,
		logger:    *logger,
	}
}

func (s *documentStore) normalizePath(pathOrUri string) (string, error) {
	path, err := uriToPath(pathOrUri)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse URI: %s", pathOrUri)
	}
	return s.fs.Canonical(path), nil
}

func (s *documentStore) DidOpen(params protocol.DidOpenTextDocumentParams, notify glsp.NotifyFunc) (*document, error) {
	langID := params.TextDocument.LanguageID
	if langID != "c3" {
		return nil, nil
	}

	uri := params.TextDocument.URI
	path, err := s.normalizePath(uri)

	if err != nil {

		s.logger.Debug("ERRROR normaliznado path:")
		return nil, err
	}
	doc := &document{
		URI:     uri,
		Path:    path,
		Content: params.TextDocument.Text,
	}
	s.documents[path] = doc
	return doc, nil
}

func (s *documentStore) Close(uri protocol.DocumentUri) {
	delete(s.documents, uri)
}

func (s *documentStore) Get(pathOrURI string) (*document, bool) {
	path, err := s.normalizePath(pathOrURI)
	s.logger.Debugf("normalized path:%s", path)

	if err != nil {
		s.logger.Errorf("ERRROR normaliznado path: %s", err)
		//s.logger.Err(err)
		return nil, false
	}

	d, ok := s.documents[path]
	return d, ok
}

type document struct {
	URI                     protocol.DocumentUri
	Path                    string
	NeedsRefreshDiagnostics bool
	Content                 string
	lines                   []string
}

// ApplyChanges updates the content of the document from LSP textDocument/didChange events.
func (d *document) ApplyChanges(changes []interface{}) {
	for _, change := range changes {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			startIndex, endIndex := c.Range.IndexesIn(d.Content)
			d.Content = d.Content[:startIndex] + c.Text + d.Content[endIndex:]
		case protocol.TextDocumentContentChangeEventWhole:
			d.Content = c.Text
		}
	}

	d.lines = nil
}
