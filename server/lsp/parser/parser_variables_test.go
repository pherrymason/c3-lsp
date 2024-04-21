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
