package document

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
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
	documents map[string]Document
}

func (pd *Storage) OpenDocument(uri string, text string, version uint) {
	pd.documents[uri] = Document{
		Uri:      uri,
		FullPath: "????",
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
