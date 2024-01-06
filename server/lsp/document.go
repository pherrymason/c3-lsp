package lsp

import (
	"github.com/pherrymason/c3-lsp/utils"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unicode"
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

func (s *documentStore) FindDeclaration(language *Language, doc *document, params *protocol.DeclarationParams) (protocol.Location, bool) {

	word := wordInPosition(doc.Content, params.Position.IndexIn(doc.Content))
	identifier, err := language.FindIdentifierDeclaration(word)
	found := err == nil

	return protocol.Location{
		URI: doc.URI,
		Range: protocol.Range{
			protocol.Position{identifier.declarationPosition.Line, identifier.declarationPosition.Character},
			protocol.Position{identifier.declarationPosition.Line, identifier.declarationPosition.Character + 1},
		},
	}, found
	/*
		OriginSelectionRange: &protocol.Range{
			protocol.Position{0, 0},
			protocol.Position{0, 0},
		},
		TargetURI: doc.URI,
		TargetSelectionRange:
	}*/
}

func wordInPosition(texto string, posicion int) string {
	// Encontrar el inicio de la palabra
	inicioPalabra := 0
	for i := posicion; i >= 0; i-- {
		if !unicode.IsLetter(rune(texto[i])) {
			inicioPalabra = i + 1
			break
		}
	}

	// Encontrar el final de la palabra
	finalPalabra := len(texto) - 1
	for i := posicion; i < len(texto); i++ {
		if !unicode.IsLetter(rune(texto[i])) {
			finalPalabra = i - 1
			break
		}
	}

	// Extraer la palabra
	palabra := texto[inicioPalabra : finalPalabra+1]
	return palabra
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
