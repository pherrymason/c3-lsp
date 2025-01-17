package server

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/featureflags"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (srv *Server) TextDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	langID := params.TextDocument.LanguageID
	if langID != "c3" {
		return nil
	}

	if featureflags.IsActive(featureflags.UseGeneratedAST) {
		doc := srv.documents.OpenDocument(params.TextDocument.URI, params.TextDocument.Text, uint(params.TextDocument.Version))

		// Build AST tree node
		doc.Ast = srv.astConverter.ConvertToAST(factory.GetCST(doc.Text), doc.Text, doc.Uri)
		// Extract Symbols
		analysis.UpdateSymbolTable(srv.symbolTable, &doc.Ast, "")

	} else {
		doc := document.NewDocumentFromDocURI(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)
		srv.state.RefreshDocumentIdentifiers(doc, srv.parser)

		return nil
	}

	return nil
}
