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

func TestExtractSymbols_finds_global_variables_declared_in_single_sentence(t *testing.T) {
	source := `int value, value2;`
	module := "file_3"
	docId := "x"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	found := symbols.Variables["value"]
	assert.Equal(t, "value", found.GetName(), "First Variable name")
	assert.Equal(t, "int", found.GetType(), "First Variable type")
	assert.Equal(t, idx.NewRange(0, 4, 0, 9), found.GetIdRange(), "First variable identifier range")
	assert.Equal(t, idx.NewRange(0, 0, 0, 18), found.GetDocumentRange(), "First variable declaration range")

	found = symbols.Variables["value2"]
	assert.Equal(t, "value2", found.GetName(), "Second variable name")
	assert.Equal(t, "int", found.GetType(), "Second variable type")
	assert.Equal(t, idx.NewRange(0, 11, 0, 17), found.GetIdRange(), "Second variable identifier range")
	assert.Equal(t, idx.NewRange(0, 0, 0, 18), found.GetDocumentRange(), "Second variable declaration range")
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
	source := `fn void test() { int value, value2; }`
	module := "x"
	docId := "file_3"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	function, found := symbols.GetChildrenFunctionByName("test")
	assert.True(t, found)

	variable := function.Variables["value"]
	assert.Equal(t, "value", variable.GetName(), "First Variable name")
	assert.Equal(t, "int", variable.GetType(), "First Variable type")
	assert.Equal(t, idx.NewRange(0, 21, 0, 26), variable.GetIdRange(), "First variable identifier range")
	assert.Equal(t, idx.NewRange(0, 17, 0, 35), variable.GetDocumentRange(), "First variable declaration range")

	variable = function.Variables["value2"]
	assert.Equal(t, "value2", variable.GetName(), "Second Variable name")
	assert.Equal(t, "int", variable.GetType(), "Second Variable type")
	assert.Equal(t, idx.NewRange(0, 28, 0, 34), variable.GetIdRange(), "Second variable identifier range")
	assert.Equal(t, idx.NewRange(0, 17, 0, 35), variable.GetDocumentRange(), "Second variable declaration range")

}

func TestExtractSymbols_find_constants(t *testing.T) {

	source := `const int A_VALUE = 12;`

	doc := document.NewDocument("docId", "mod", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	found := symbols.Variables["A_VALUE"]
	assert.Equal(t, "A_VALUE", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType(), "Variable type")
	assert.True(t, found.IsConstant())
	assert.Equal(t, idx.NewRange(0, 0, 0, 23), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 10, 0, 17), found.GetIdRange())
}
