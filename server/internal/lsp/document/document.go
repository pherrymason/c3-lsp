package document

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
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
	Cst             *sitter.Tree
	Version         uint

	LastGoodCst *sitter.Tree
	LastGoodAst *sitter.Tree
	Imports     []Module
}

func (d *Document) Close() {
	d.Cst.Close()
}

type Storage struct {
	Documents map[string]*Document
}

func NewStore() *Storage {
	return &Storage{
		Documents: make(map[string]*Document),
	}
}

func (d *Storage) OpenDocument(uri string, text string, version uint) *Document {
	document := &Document{
		Uri:      uri,
		FullPath: utils.NormalizePath(uri),
		Text:     text,
		Owned:    true,
		Version:  version,
	}

	d.Documents[uri] = document

	return document
}

func (d *Storage) OpenDocumentFromPath(path string, text string, version uint) {
	converter := factory.NewASTConverter()
	uri := fs.ConvertPathToURI(path, option.None[string]())
	document := &Document{
		Uri:      uri,
		FullPath: path,
		Text:     text,
		Owned:    false,
		Version:  version,
		Ast:      converter.ConvertToAST(factory.GetCST(text).RootNode(), text, path),
	}

	d.Documents[uri] = document
}

func (d *Storage) CloseDocument(uri string) {
	d.Documents[uri].Close()
	delete(d.Documents, uri)
}

func (d *Storage) GetDocument(uri string) (*Document, error) {
	return d.Documents[uri], nil
}
