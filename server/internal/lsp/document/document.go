package document

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
)

type Module struct {
}

type Document struct {
	Uri             string
	FullPath        string
	Text            string
	Owned           bool
	DiagnosedErrors bool
	Ast             *ast.File
	Version         uint
	Imports         []Module
}

type Storage struct {
	Documents map[string]*Document
}

func NewStore() *Storage {
	return &Storage{
		Documents: make(map[string]*Document),
	}
}

func (pd *Storage) OpenDocument(uri string, text string, version uint) *Document {
	document := &Document{
		Uri:      uri,
		FullPath: utils.NormalizePath(uri),
		Text:     text,
		Owned:    true,
		Version:  version,
	}

	pd.Documents[uri] = document

	return document
}

func (pd *Storage) OpenDocumentFromPath(path string, text string, version uint) {
	converter := factory.NewASTConverter()
	uri := fs.ConvertPathToURI(path, option.None[string]())
	document := &Document{
		Uri:      uri,
		FullPath: path,
		Text:     text,
		Owned:    false,
		Version:  version,
		Ast:      converter.ConvertToAST(factory.GetCST(text), text, path),
	}

	pd.Documents[uri] = document
}

func (pd *Storage) CloseDocument(uri string) {
	delete(pd.Documents, uri)
}

func (pd *Storage) GetDocument(uri string) (*Document, error) {
	return pd.Documents[uri], nil
}
