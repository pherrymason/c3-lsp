package lsp

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func TestLanguage_FindHoverInformation(t *testing.T) {
	language := NewLanguage()

	doc := NewDocumentFromString("x", `
	int value = 1;
	fn void main() {
		char value = 3;
	}
`)
	language.RefreshDocumentIdentifiers(&doc)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{
				Line:      3,
				Character: 8,
			},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover, _ := language.FindHoverInformation(&doc, &params)

	expectedHover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("char value"),
		},
	}
	assert.Equal(t, expectedHover, hover)
}

func TestLanguage_FindHoverInformationFromDifferentFile(t *testing.T) {
	language := NewLanguage()

	doc := NewDocumentFromString("x", `
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc)

	doc2 := NewDocumentFromString("y", `
	fn void importedMethod() {}
	`)
	language.RefreshDocumentIdentifiers(&doc2)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			NewPosition(2, 8),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover, _ := language.FindHoverInformation(&doc, &params)

	expectedHover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("?? importedMethod()"),
		},
	}
	assert.Equal(t, expectedHover, hover)
}

func TestLanguage_FindIdentifierDeclaration_same_scope(t *testing.T) {
	language := NewLanguage()

	doc := NewDocumentFromString("x", `
		int value = 1;
		value = 3;
	`)
	language.RefreshDocumentIdentifiers(&doc)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			NewPosition(3, 4),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"x",
		NewRange(1, 6, 1, 11),
		NewRange(1, 6, 1, 11),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindIdentifierDeclaration_outside_current_function(t *testing.T) {
	language := NewLanguage()

	doc := NewDocumentFromString("x", `
		int value = 1;
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			NewPosition(3, 4),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"x",
		NewRange(1, 6, 1, 11),
		NewRange(1, 6, 1, 11),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindIdentifierDeclaration_outside_current_file(t *testing.T) {
	language := NewLanguage()

	doc := NewDocumentFromString("x", `
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc)
	doc2 := NewDocumentFromString("y", `int value = 1;`)
	language.RefreshDocumentIdentifiers(&doc2)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			NewPosition(2, 4),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"y",
		NewRange(0, 4, 0, 9),
		NewRange(0, 4, 0, 9),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}
