package language

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}

// Asking the selectedSymbol information in the very same declaration, should resolve to the correct selectedSymbol
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

	resolvedSymbol, err := language.findSymbolDeclarationInDocPositionScope(NewSearchParams("out", docA), protocol.Position{0, 5})

	assert.Equal(t, nil, err)
	assert.Equal(t,
		idx.NewVariableBuilder("out", "Out", moduleA, docA).
			WithIdentifierRange(0, 0, 0, 10).
			Build(),
		resolvedSymbol,
	)
}

// Asking the selectedSymbol information in the very same declaration, should resolve to the correct selectedSymbol
// Even if there is another selectedSymbol with same name in a different file.
func TestLanguage_findClosestSymbolDeclaration_cursor_on_declaration_resolves_to_same_declaration(t *testing.T) {
	language := NewLanguage()

	// Doc A
	docA := "a"
	moduleA := "modA"
	fileA := idx.NewFunctionBuilderARoot(moduleA, docA).
		Build()
	fileA.AddVariable(idx.NewVariableBuilder("out", "Out", moduleA, docA).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docA] = fileA

	// Doc B
	docB := "b"
	moduleB := "modB"
	fileB := idx.NewFunctionBuilderARoot(moduleB, docB).
		Build()
	fileB.AddVariable(idx.NewVariableBuilder("out", "int", moduleB, docB).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docB] = fileB
	// Add more docs to the map to increase possibility of iterating in random ways
	language.functionTreeByDocument["3"] = idx.NewFunctionBuilder("aaa", "void", "aaa", "aaa").Build()
	language.functionTreeByDocument["4"] = idx.NewFunctionBuilder("bbb", "void", "bbb", "bbb").Build()

	resolvedSymbol := language.findClosestSymbolDeclaration(NewSearchParams("out", docA), protocol.Position{0, 5})

	assert.Equal(t,
		idx.NewVariableBuilder("out", "Out", moduleA, docA).
			WithIdentifierRange(0, 0, 0, 10).
			Build(),
		resolvedSymbol,
	)
}

// struct MyStruct{}
// fn void MyStruct.search(&self) {}
// fn void search() {}
//
// MyStruct object;
// object.search();
func TestLanguage_findClosestSymbolDeclaration_should_find_right_symbol_when_asking_for_type_function(t *testing.T) {
	language := NewLanguage()

	docId := "a"
	moduleId := "modA"
	fileA := idx.NewFunctionBuilderARoot(moduleId, docId).
		Build()
	fileA.AddVariable(idx.NewVariableBuilder("object", "MyStruct", moduleId, docId).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	fileA.AddStruct(idx.NewStructBuilder("MyStruct", moduleId, docId).
		WithIdentifierRange(0, 0, 0, 10).
		Build(),
	)
	fileA.AddFunction(
		idx.NewFunctionBuilder("search", "void", moduleId, docId).
			WithTypeIdentifier("MyStruct").
			WithDocumentRange(0, 0, 0, 100).
			Build(),
	)
	fileA.AddFunction(
		idx.NewFunctionBuilder("search", "int", moduleId, docId).
			WithDocumentRange(0, 0, 0, 100).
			Build(),
	)
	language.functionTreeByDocument[docId] = fileA

	params := NewSearchParams("search", docId)
	params.parentSymbol = "object"
	resolvedSymbol := language.findClosestSymbolDeclaration(params, protocol.Position{0, 5})

	expectedSymbol := idx.NewFunctionBuilder("search", "void", moduleId, docId).
		WithTypeIdentifier("MyStruct").
		WithDocumentRange(0, 0, 0, 100).
		Build()

	assert.Equal(t,
		&expectedSymbol,
		resolvedSymbol,
	)
}

// Doc A
// -------
// struct MyStruct{}
// fn void MyStruct.search(&self) {}
// fn void search() {}
//
// Doc B
// ------
// MyStruct object;
// object.search();
func TestLanguage_findClosestSymbolDeclaration_should_find_right_symbol_when_asking_for_type_function_parent_symbol_defined_in_different_file(t *testing.T) {
	language := NewLanguage()

	docA := "a"
	moduleA := "modA"
	fileA := idx.NewFunctionBuilderARoot(moduleA, docA).Build()
	fileA.AddVariable(idx.NewVariableBuilder("object", "MyStruct", moduleA, docA).
		WithIdentifierRange(0, 0, 0, 10).
		WithDocumentRange(0, 0, 0, 10).
		Build(),
	)
	language.functionTreeByDocument[docA] = fileA

	// Doc B has struct definition + struct function `search` + function named `search`
	docB := "b"
	moduleB := "modB"
	fileB := idx.NewFunctionBuilderARoot(moduleB, docB).Build()
	fileB.AddStruct(idx.NewStructBuilder("MyStruct", moduleB, docB).
		WithIdentifierRange(0, 0, 0, 10).
		WithDocumentRange(0, 0, 0, 10).
		Build(),
	)
	fileB.AddFunction(
		idx.NewFunctionBuilder("search", "void", moduleB, docB).
			WithTypeIdentifier("MyStruct").
			WithDocumentRange(0, 0, 0, 100).
			Build(),
	)
	fileB.AddFunction(
		idx.NewFunctionBuilder("search", "int", moduleB, docB).
			WithDocumentRange(0, 0, 0, 100).
			Build(),
	)
	language.functionTreeByDocument[docB] = fileB

	params := NewSearchParams("search", docA)
	params.parentSymbol = "object"
	resolvedSymbol := language.findClosestSymbolDeclaration(params, protocol.Position{0, 5})

	expectedSymbol := idx.NewFunctionBuilder("search", "void", moduleB, docB).
		WithTypeIdentifier("MyStruct").
		WithDocumentRange(0, 0, 0, 100).
		Build()
	assert.Equal(t,
		&expectedSymbol,
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

func initLanguage(docId string, module string, source string) (Language, *document.Document) {
	doc := document.NewDocument(docId, module, source)
	language := NewLanguage()
	parser := createParser()
	language.RefreshDocumentIdentifiers(&doc, &parser)

	return language, &doc
}

func TestLanguage_ShouldFindVariablesInSameScope(t *testing.T) {
	module := "mod"
	docId := "docId"

	t.Run("should find variable in same scope of cursor", func(t *testing.T) {
		source := `fn void test() {int value=1; value = 3;}`
		language, doc := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 32)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewVariableBuilder("value", "int", module, docId).
			WithIdentifierRange(0, 20, 0, 25).
			WithDocumentRange(0, 20, 0, 27).
			Build()
		assert.Equal(t, expected, symbol)
	})

	t.Run("should find enum declaration in same scope of cursor", func(t *testing.T) {
		source := `enum Colors { RED, BLUE, GREEN };Colors foo = RED;`
		language, doc := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 36)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewEnumBuilder("Colors", "", module, docId).
			WithIdentifierRange(0, 5, 0, 11).
			WithDocumentRange(0, 0, 0, 32).
			WithEnumerator(
				idx.NewEnumeratorBuilder("RED", docId).
					WithIdentifierRange(0, 14, 0, 17).
					Build(),
			).
			WithEnumerator(
				idx.NewEnumeratorBuilder("BLUE", docId).
					WithIdentifierRange(0, 19, 0, 23).
					Build(),
			).
			WithEnumerator(
				idx.NewEnumeratorBuilder("GREEN", docId).
					WithIdentifierRange(0, 25, 0, 30).
					Build(),
			).
			Build()
		assert.Equal(t, expected, symbol)
	})

	t.Run("should find enumerator declaration in same scope of cursor", func(t *testing.T) {
		source := `enum Colors { RED, BLUE, GREEN };Colors foo = RED;`
		language, doc := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 48)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewEnumeratorBuilder("RED", docId).
			WithIdentifierRange(0, 14, 0, 17).
			Build()
		assert.Equal(t, expected, symbol)
	})

	t.Run("should find struct declaration in same scope of cursor", func(t *testing.T) {
		source := `struct MyStructure {bool enabled; char key;} MyStructure value;`
		language, doc := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 47)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewStructBuilder("MyStructure", module, docId).
			WithStructMember("enabled", "bool", idx.NewRange(0, 25, 0, 32)).
			WithStructMember("key", "char", idx.NewRange(0, 39, 0, 42)).
			WithIdentifierRange(0, 7, 0, 18).
			WithDocumentRange(0, 0, 0, 44).
			Build()
		assert.Equal(t, expected, symbol)
	})

	t.Run("should find definition declaration in same scope of cursor", func(t *testing.T) {
		source := `def Kilo = int;Kilo value = 3;`
		language, doc := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 17)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewDefBuilder("Kilo", docId).
			WithResolvesTo("int").
			WithIdentifierRange(0, 4, 0, 8).
			WithDocumentRange(0, 0, 0, 15).
			Build()
		assert.Equal(t, expected, symbol)
	})

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
			protocol.Position{2, 4},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	symbol, err := language.FindSymbolDeclarationInWorkspace(&doc, params.Position)

	assert.Nil(t, err)

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

	symbol, _ := language.FindSymbolDeclarationInWorkspace(&doc, params.Position)

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

	symbol, _ := language.FindSymbolDeclarationInWorkspace(&doc, params.Position)

	expectedSymbol := idx.NewVariableBuilder("value", "int", "mod", "y").
		WithIdentifierRange(0, 4, 0, 9).
		WithDocumentRange(0, 0, 0, 14).
		Build()

	assert.Equal(t, expectedSymbol, symbol)
}
