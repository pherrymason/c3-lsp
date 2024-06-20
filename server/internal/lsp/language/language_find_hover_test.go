package language

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestLanguage_FindHoverInformation(t *testing.T) {

	language := NewLanguage(commonlog.MockLogger{}, option.Some("dummy"))
	parser := createParser()

	doc := document.NewDocument("x", `
	int value = 1;
	fn void main() {
		char value = 3;
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "x"},
			Position: protocol.Position{
				Line:      3,
				Character: 8,
			},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover := language.FindHoverInformation(&doc, &params)

	expectedHover := option.Some(protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "char value",
		},
	})
	assert.Equal(t, expectedHover, hover)
}

func TestLanguage_FindHoverInformationFromDifferentFile(t *testing.T) {
	t.Skip()
	language := NewLanguage(commonlog.MockLogger{}, option.Some("dummy"))
	parser := createParser()
	docId := "x"
	doc := document.NewDocument(docId, `
	module a;
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc2Id := "y"
	doc2 := document.NewDocument(doc2Id, `
	module a;
	fn void importedMethod() {}
	`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docId},
			Position:     protocol.Position{Line: 3, Character: 8},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover := language.FindHoverInformation(&doc, &params)

	expectedHover := option.Some(protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "void importedMethod()",
		},
	})
	assert.Equal(t, expectedHover, hover)
}
