package lsp

import (
	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pkg/errors"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"unicode"
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

func (s *documentStore) DidOpen(params protocol.DidOpenTextDocumentParams, notify glsp.NotifyFunc) (*Document, error) {
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
		s.logger.Errorf("ERRROR normaliznado path: %s", err)
		//s.logger.Err(err)
		return nil, false
	}

	d, ok := s.documents[path]
	return d, ok
}

func (s *documentStore) FindDeclaration(language *Language, doc *Document, params *protocol.DeclarationParams) (protocol.Location, bool) {

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

func wordInPosition(text string, position int) string {
	wordStart := 0
	for i := position; i >= 0; i-- {
		if !unicode.IsLetter(rune(text[i])) {
			wordStart = i + 1
			break
		}
	}

	wordEnd := len(text) - 1
	for i := position; i < len(text); i++ {
		if !unicode.IsLetter(rune(text[i])) {
			wordEnd = i - 1
			break
		}
	}

	word := text[wordStart : wordEnd+1]
	return word
}
