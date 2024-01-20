package language

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}

func TestLanguage_findSymbolDeclarationInDocPositionScope_cursor_on_declaration_resolves_to_same_declaration(t *testing.T) {
	language := NewLanguage()

	// Doc A
	docA := "a"
	moduleA := "modA"
	fileA := idx.NewFunctionBuilder("main", "void", moduleA, docA).
		WithDocumentRange(0, 0, 0, 20).
		Build()
	fileA.AddVariable(idx.NewVariableBuilder("out", "Out", moduleA, docA).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docA] = fileA

	resolvedSymbol, err := language.findSymbolDeclarationInDocPositionScope("out", docA, protocol.Position{0, 5})

	assert.Equal(t, nil, err)
	assert.Equal(t,
		idx.NewVariableBuilder("out", "Out", moduleA, docA).
			WithIdentifierRange(0, 0, 0, 10).
			Build(),
		resolvedSymbol,
	)
}

func TestLanguage_findClosestSymbolDeclaration_cursor_on_declaration_resolves_to_same_declaration(t *testing.T) {
	language := NewLanguage()

	// Doc A
	docA := "a"
	moduleA := "modA"
	fileA := idx.NewFunctionBuilder("main", "void", moduleA, docA).
		Build()
	fileA.AddVariable(idx.NewVariableBuilder("out", "Out", moduleA, docA).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docA] = fileA

	// Doc B
	docB := "b"
	moduleB := "modB"
	fileB := idx.NewFunctionBuilder("main", "void", moduleB, docB).
		Build()
	fileB.AddVariable(idx.NewVariableBuilder("out", "int", moduleB, docB).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docB] = fileB
	// Add more docs to the map to increase possibility of iterating in random ways
	language.functionTreeByDocument["3"] = idx.NewFunctionBuilder("aaa", "void", "aaa", "aaa").Build()
	language.functionTreeByDocument["4"] = idx.NewFunctionBuilder("bbb", "void", "bbb", "bbb").Build()

	resolvedSymbol := language.findClosestSymbolDeclaration("out", docA, protocol.Position{0, 5})

	assert.Equal(t,
		idx.NewVariableBuilder("out", "Out", moduleA, docA).
			WithIdentifierRange(0, 0, 0, 10).
			Build(),
		resolvedSymbol,
	)
}

func TestLanguage_FindHoverInformation(t *testing.T) {
	language := NewLanguage()
	parser := createParser()

	doc := document.NewDocument("x", "", `
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

	doc := document.NewDocument("x", "x", `
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc2 := document.NewDocument("y", "x", `
	fn void importedMethod() {}
	`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{Line: 2, Character: 8},
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
			idx.NewStructBuilder("MyStructure", module, "x").
				WithStructMember("enabled", "bool", idx.NewRange(0, 20, 0, 33)).
				WithStructMember("key", "char", idx.NewRange(0, 34, 0, 43)).
				WithIdentifierRange(0, 7, 0, 18).
				Build(),
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
			doc := document.NewDocument("x", module, tt.sourceCode)
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
	doc := document.NewDocument("x", "mod", `
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
	doc := document.NewDocument("x", "mod", `
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
	doc := document.NewDocument("x", "mod", `
		fn void main() {
			value = 3;
		}
	`)
	language.RefreshDocumentIdentifiers(&doc, &parser)
	doc2 := document.NewDocument("y", "mod", `int value = 1;`)
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
