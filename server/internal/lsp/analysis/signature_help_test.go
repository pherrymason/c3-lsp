package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/factory"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

type server struct {
	documents   *document.Storage
	symbolTable *SymbolTable
}

func env(source string, uri string) (server, lsp.Position) {
	srv := server{
		documents:   document.NewStore(),
		symbolTable: NewSymbolTable(),
	}

	source, position := parseBodyWithCursor(source)

	astConverter := factory.NewASTConverter()

	tree := astConverter.ConvertToAST(factory.GetCST(source).RootNode(), source, uri)
	doc := srv.documents.OpenDocument(uri, source, 1)
	doc.Ast = tree
	UpdateSymbolTable(srv.symbolTable, tree, uri)

	return srv, position
}

func getSignatureHelp(source string) *protocol.SignatureHelp {
	uri := "file://dummy.c3"
	srv, position := env(source, uri)

	getDocument, _ := srv.documents.GetDocument(uri)
	return BuildSignatureHelp(getDocument, position, srv.documents, srv.symbolTable)
}

func TestBuildSignatureHelp(t *testing.T) {
	source := `module app;
	<* Docblock comment *>
	fn void foo(bool a, int b) {}
	fn void main() {
		foo(|||);
	}`

	signature := getSignatureHelp(source)
	expected := &protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{
			{
				Label: "foo(bool a, int b)",
				Parameters: []protocol.ParameterInformation{
					{
						Label: "bool a",
					},
					{
						Label: "int b",
					},
				},
				Documentation:   protocol.MarkupContent{Kind: "markdown", Value: "Docblock comment"},
				ActiveParameter: cast.ToPtr(protocol.UInteger(0)),
			},
		},
	}
	assert.Equal(t, expected, signature)
}

func TestBuildSignatureHelp_with_missing_closing_parenthesis(t *testing.T) {
	t.Skip("Not ready, parser breaks node structure")
	source := `module app;
	<* Docblock comment *>
	fn void foo(bool a, int b) {}
	fn void main() {
		foo(|||
	}`

	signature := getSignatureHelp(source)
	expected := &protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{
			{
				Label: "foo(bool a, int b)",
				Parameters: []protocol.ParameterInformation{
					{
						Label: "bool a",
					},
					{
						Label: "int b",
					},
				},
				Documentation:   protocol.MarkupContent{Kind: "markdown", Value: "Docblock comment"},
				ActiveParameter: cast.ToPtr(protocol.UInteger(0)),
			},
		},
	}
	assert.Equal(t, expected, signature)
}
