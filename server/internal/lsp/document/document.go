package document

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
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
	Ast             ast.File
	Version         uint
	Imports         []Module
}

type Storage struct {
	documents map[string]*Document
}

func NewStore() *Storage {
	return &Storage{
		documents: make(map[string]*Document),
	}
}

func (pd *Storage) OpenDocument(uri string, text string, version uint) {
	pd.documents[uri] = &Document{
		Uri:      uri,
		FullPath: utils.NormalizePath(uri),
		Text:     text,
		Owned:    true,
		Version:  version,
		Ast: ast.ConvertToAST(
			ast.GetCST(text),
			text,
			uri,
		),
	}
}

func (pd *Storage) CloseDocument(uri string) {
	delete(pd.documents, uri)
}

func (pd *Storage) GetDocument(uri string) (*Document, error) {
	return pd.documents[uri], nil
}
