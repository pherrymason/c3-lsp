package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
)

func assertVariableFound(t *testing.T, name string, symbols idx.Function) {
	_, ok := symbols.Variables[name]
	assert.True(t, ok)
}

func TestExtractSymbols_finds_global_variables_declarations(t *testing.T) {
	source := `int value = 1;`
	module := "file_3"
	docId := "x"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	found := symbols.Variables["value"]
	assert.Equal(t, "value", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType(), "Variable type")
	assert.Equal(t, idx.NewRange(0, 0, 0, 14), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 4, 0, 9), found.GetIdRange())
}

func TestExtractSymbols_finds_multiple_global_variables_in_single_declaration(t *testing.T) {
	source := `int value = 1, value2 = 2;`
	module := "file_3"
	docId := "x"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	found := symbols.Variables["value"]
	assert.Equal(t, "value", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType(), "Variable type")
	assert.Equal(t, idx.NewRange(0, 0, 0, 14), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 4, 0, 9), found.GetIdRange())

	found = symbols.Variables["value2"]
	assert.Equal(t, "value2", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType(), "Variable type")
	assert.Equal(t, idx.NewRange(0, 15, 0, 21), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 15, 0, 21), found.GetIdRange())
}

func TestExtractSymbols_variables_declared_in_function(t *testing.T) {
	source := `fn void test() { int value = 1; }`
	module := "x"
	docId := "file_3"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	function, found := symbols.GetChildrenFunctionByName("test")
	assert.True(t, found)

	expectedVariableBldr := idx.NewVariableBuilder("value", "int", module, docId)
	expectedVariableBldr.
		WithDocumentRange(0, 17, 0, 31).
		WithIdentifierRange(0, 21, 0, 26)
	assertVariableFound(t, "value", function)
	assertSameVariable(t, expectedVariableBldr.Build(), function.Variables["value"], "value variable")
}

func TestExtractSymbols_multiple_variables_declared_in_function(t *testing.T) {
	source := `fn void test() { int value = 1, value2 = 2; }`
	module := "x"
	docId := "file_3"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	function, found := symbols.GetChildrenFunctionByName("test")
	assert.True(t, found)

	expectedVariableBldr := idx.NewVariableBuilder("value", "int", module, docId)
	expectedVariableBldr.
		WithDocumentRange(0, 17, 0, 31).
		WithIdentifierRange(0, 21, 0, 26)
	assertVariableFound(t, "value", function)
	assertSameVariable(t, expectedVariableBldr.Build(), function.Variables["value"], "value variable")

	expectedVariable2Bldr := idx.NewVariableBuilder("value2", "int", module, docId)
	expectedVariable2Bldr.
		WithDocumentRange(0, 32, 0, 42).
		WithIdentifierRange(0, 32, 0, 38)
	assertVariableFound(t, "value2", function)
	assertSameVariable(t, expectedVariable2Bldr.Build(), function.Variables["value2"], "value variable")
}
