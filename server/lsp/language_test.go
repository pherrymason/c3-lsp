package lsp

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func TestLanguage_FindHoverInformation(t *testing.T) {
	language := NewLanguage()
	parser := createParser()

	doc := NewDocumentFromString("x", `
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

	doc := NewDocumentFromString("x", `
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc2 := NewDocumentFromString("y", `
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
			Value: fmt.Sprintf("?? importedMethod()"),
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
func createVariable(docId string, name string, baseType string, sL uint, sC uint, eL uint, eC uint) indexables.Indexable {
	return indexables.Variable{
		Name: name,
		Type: baseType,
		BaseIndexable: indexables.NewBaseIndexable(
			docId,
			indexables.NewRange(sL, sC, eL, eC),
			indexables.NewRange(sL, sC, eL, eC),
			protocol.CompletionItemKindVariable,
		),
	}
}

func createEnum(docId string, name string, variants []indexables.Enumerator, idRange [4]uint, docRange [4]uint) *indexables.Enum {
	enum := indexables.NewEnum(
		name,
		"",
		variants,
		indexables.NewRange(idRange[0], idRange[1], idRange[2], idRange[3]),
		indexables.NewRange(docRange[0], docRange[1], docRange[2], docRange[3]),
		docId,
	)

	return &enum
}

func createEnumerator(name string, pRange [4]uint) indexables.Enumerator {
	enumerator := indexables.NewEnumerator(name, "", indexables.NewRange(pRange[0], pRange[1], pRange[2], pRange[3]))

	return enumerator
}

func createStruct(docId string, name string, members []indexables.StructMember, idRange indexables.Range) indexables.Indexable {
	return indexables.NewStruct(
		name,
		members,
		docId,
		idRange,
	)
}

func TestLanguage_FindSymbolDeclarationInWorkspace_symbol_same_scope(t *testing.T) {
	cases := []struct {
		name               string
		sourceCode         string
		highlightedWord    string
		cursorPositionLine protocol.UInteger
		cursorPositionChar protocol.UInteger
		expected           indexables.Indexable
	}{
		{"variable",
			`int value=1;value=3;`,
			"value",
			0, 13,
			createVariable("x", "value", "int", 0, 4, 0, 9)},
		{
			"enum declaration",
			`enum Colors = { RED, BLUE, GREEN };Colors foo = RED;`,
			"Colors",
			0, 36,
			createEnum("x", "Colors", []indexables.Enumerator{
				indexables.NewEnumerator("RED", "", indexables.NewRange(0, 16, 0, 19)),
				indexables.NewEnumerator("BLUE", "", indexables.NewRange(0, 21, 0, 25)),
				indexables.NewEnumerator("GREEN", "", indexables.NewRange(0, 27, 0, 32)),
			}, [4]uint{0, 5, 0, 11}, [4]uint{0, 0, 0, 34}),
		},
		{
			"enum enumerator",
			`enum Colors = { RED, BLUE, GREEN };Colors foo = RED;`,
			"RED",
			0, 49,
			createEnumerator("RED", [4]uint{0, 16, 0, 19}),
		},
		{
			"struct",
			`struct MyStructure {bool enabled; char key;} MyStructure value;`,
			"MyStructure",
			0, 47,
			createStruct("x", "MyStructure", []indexables.StructMember{
				indexables.NewStructMember("enabled", "bool"),
				indexables.NewStructMember("key", "char"),
			},
				indexables.NewRange(0, 7, 0, 18)),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			doc := NewDocumentFromString("x", tt.sourceCode)
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
	doc := NewDocumentFromString("x", `
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

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"x",
		indexables.NewRange(1, 6, 1, 11),
		indexables.NewRange(1, 6, 1, 11),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindSymbolDeclarationInWorkspace_variable_outside_current_function(t *testing.T) {
	language := NewLanguage()
	parser := createParser()
	doc := NewDocumentFromString("x", `
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

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"x",
		indexables.NewRange(1, 6, 1, 11),
		indexables.NewRange(1, 6, 1, 11),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}

func TestLanguage_FindSymbolDeclarationInWorkspace_variable_outside_current_file(t *testing.T) {
	language := NewLanguage()
	parser := createParser()
	doc := NewDocumentFromString("x", `
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc, &parser)
	doc2 := NewDocumentFromString("y", `int value = 1;`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{2, 4},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, _ := language.FindSymbolDeclarationInWorkspace(doc.URI, "value", params.Position)

	expectedSymbol := indexables.NewVariable(
		"value",
		"int",
		"y",
		indexables.NewRange(0, 4, 0, 9),
		indexables.NewRange(0, 4, 0, 9),
		protocol.CompletionItemKindVariable,
	)
	assert.Equal(t, expectedSymbol, symbol)
}

func createParser() Parser {
	return Parser{
		logger: commonlog.MockLogger{},
	}
}
