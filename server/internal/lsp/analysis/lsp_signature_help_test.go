package analysis

import (
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func getSignatureHelp(source string) *protocol.SignatureHelp {
	uri := "file://dummy.c3"
	srv, position := startTestServer(source, uri)

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
