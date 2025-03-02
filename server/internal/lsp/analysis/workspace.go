package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis/symbol"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"log"
)

type Workspace struct {
	documents   *document.Storage
	astParser   *factory.ASTConverter
	symbolTable *symbols.SymbolTable
}

func NewWorkspace() *Workspace {
	return &Workspace{
		documents:   document.NewStore(),
		astParser:   factory.NewASTConverter(),
		symbolTable: symbols.NewSymbolTable(),
	}
}

func (w *Workspace) OpenDocument(uri protocol.DocumentUri, text string, version uint) *document.Document {
	cst := factory.GetCST(text)

	doc := w.documents.OpenDocument(uri, text, version)
	doc.Cst = cst

	ast := w.astParser.ConvertToAST(cst.RootNode(), text, uri)
	doc.Ast = ast
	symbols.UpdateSymbolTable(w.symbolTable, doc.Ast, "")

	return doc
}

func (w *Workspace) UpdateDocument(uri protocol.DocumentUri, protocolChanges []interface{}, version uint) {
	doc, err := w.documents.GetDocument(uri)
	if err != nil {
		log.Fatalf("error getting document: %s", uri)
		return
	}

	for _, change := range protocolChanges {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			startIndex, endIndex := c.Range.IndexesIn(doc.Text)
			doc.Text = doc.Text[:startIndex] + c.Text + doc.Text[endIndex:]

		case protocol.TextDocumentContentChangeEventWhole:
			doc.Text = c.Text
		}
	}

	// TODO optimization opportunity here: use doc.cst.Edit() to only reparse what actually changed. However, astParser.ConvertToAST will reparse the whole tree again.
	editInput := sitter.EditInput{}
	doc.Cst.Edit(editInput)

	ast := w.astParser.ConvertToAST(doc.Cst.RootNode(), doc.Text, uri)
	doc.Ast = ast

	symbols.UpdateSymbolTable(w.symbolTable, doc.Ast, "")
}
