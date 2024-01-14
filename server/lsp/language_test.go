package lsp

import (
	"fmt"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func TestLanguage_FindHoverInformation(t *testing.T) {
	language := NewLanguage()
	parser := createParser()

	doc := NewDocumentFromString("x", "", `
	int value = 1;
	fn void main() {
		char value = 3;
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

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
	parser := createParser()

	doc := NewDocumentFromString("x", "x", `
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc2 := NewDocumentFromString("y", "x", `
	fn void importedMethod() {}
	`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			lsp_NewPosition(2, 8),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover, _ := language.FindHoverInformation(&doc, &params)

	expectedHover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("void importedMethod()"),
		},
	}
	assert.Equal(t, expectedHover, hover)
}

func newDeclarationParams(docId string, line protocol.UInteger, char protocol.UInteger) protocol.DeclarationParams {
	return protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: docId},
			protocol.Position{line, char},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}
}
func TestLanguage_FindSymbolDeclarationInWorkspace_symbol_same_scope(t *testing.T) {
	module := "mod"
	cases := []struct {
		name               string
		sourceCode         string
		highlightedWord    string
		cursorPositionLine protocol.UInteger
		cursorPositionChar protocol.UInteger
		expected           idx.Indexable
	}{
		{"variable",
			`int value=1;value=3;`,
			"value",
			0, 13,
			idx.NewVariableBuilder("value", "int", module, "x").
				WithIdentifierRange(0, 4, 0, 9).
				WithDocumentRange(0, 0, 0, 12).
				Build()},
		{
			"enum declaration",
			`enum Colors = { RED, BLUE, GREEN };Colors foo = RED;`,
			"Colors",
			0, 36,
			idx.NewEnumBuilder("Colors", "", module, "x").
				WithIdentifierRange(0, 5, 0, 11).
				WithDocumentRange(0, 0, 0, 34).
				WithEnumerator(
					idx.NewEnumeratorBuilder("RED", "x").
						WithIdentifierRange(0, 16, 0, 19).
						Build(),
				).
				WithEnumerator(
					idx.NewEnumeratorBuilder("BLUE", "x").
						WithIdentifierRange(0, 21, 0, 25).
						Build(),
				).
				WithEnumerator(
					idx.NewEnumeratorBuilder("GREEN", "x").
						WithIdentifierRange(0, 27, 0, 32).
						Build(),
				).
				Build(),
		},
		{
			"enum enumerator",
			`enum Colors = { RED, BLUE, GREEN };Colors foo = RED;`,
			"RED",
			0, 49,
			idx.NewEnumeratorBuilder("RED", "x").
				WithIdentifierRange(0, 16, 0, 19).
				Build(),
		},
		{
			"struct",
			`struct MyStructure {bool enabled; char key;} MyStructure value;`,
			"MyStructure",
			0, 47,
			createStruct("x", module, "MyStructure", []idx.StructMember{
				idx.NewStructMember("enabled", "bool", idx.NewRange(0, 20, 0, 33)),
				idx.NewStructMember("key", "char", idx.NewRange(0, 34, 0, 43)),
			}, idx.NewRange(0, 7, 0, 18)),
		},
		{
			"def",
			"def Kilo = int;Kilo value = 3;",
			"Kilo",
			0, 17,
			idx.NewDefBuilder("Kilo", "x").
				WithResolvesTo("int").
				WithIdentifierRange(0, 4, 0, 8).
				WithDocumentRange(0, 0, 0, 15).
				Build(),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocumentFromString("x", module, tt.sourceCode)
			language := NewLanguage()
			parser := createParser()
			language.RefreshDocumentIdentifiers(&doc, &parser)

			params := newDeclarationParams("x", tt.cursorPositionLine, tt.cursorPositionChar)

			symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, tt.highlightedWord, params.Position)

			assert.Equal(t, tt.expected, symbol)
		})
	}
}

func TestLanguage_FindSymbolDeclarationInWorkspace_variable_same_scope(t *testing.T) {
	language := NewLanguage()
	parser := createParser()
	doc := NewDocumentFromString("x", "mod", `
		int value = 1;
		value = 3;
	`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{3, 4},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := idx.NewVariableBuilder("value", "int", "mod", "x").
		WithIdentifierRange(1, 6, 1, 11).
		WithDocumentRange(1, 2, 1, 16).
		Build()

	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindSymbolDeclarationInWorkspace_variable_outside_current_function(t *testing.T) {
	language := NewLanguage()
	parser := createParser()
	doc := NewDocumentFromString("x", "mod", `
		int value = 1;
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{3, 4},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := idx.NewVariableBuilder("value", "int", "mod", "x").
		WithIdentifierRange(1, 6, 1, 11).
		WithDocumentRange(1, 2, 1, 16).
		Build()

	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindSymbolDeclarationInWorkspace_variable_outside_current_file(t *testing.T) {
	language := NewLanguage()
	parser := createParser()
	doc := NewDocumentFromString("x", "mod", `
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc, &parser)
	doc2 := NewDocumentFromString("y", "mod", `int value = 1;`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{2, 4},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := idx.NewVariableBuilder("value", "int", "mod", "y").
		WithIdentifierRange(0, 4, 0, 9).
		WithDocumentRange(0, 0, 0, 14).
		Build()

	assert.Equal(t, expectedSymbol, symbol)
}
