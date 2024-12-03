package projectDocuments

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
)

type Document struct {
	Uri             string
	FullPath        string
	Text            string
	Owned           bool
	DiagnosedErrors bool
	Ast             ast.File
	Version         uint
}

type ProjectDocuments struct {
	documents map[string]Document
}

func (pd *ProjectDocuments) OpenDocument(uri string, text string, version uint) {
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

func (pd *ProjectDocuments) CloseDocument(uri string) {
	delete(pd.documents, uri)
}
