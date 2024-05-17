package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
)

func assertVariableFound(t *testing.T, name string, symbols idx.Function) {
	_, ok := symbols.Variables[name]
	assert.True(t, ok)
}

func TestExtractSymbols_find_variables(t *testing.T) {
	source := `
	int value = 1;
	char* character;
	int foo, foo2;
	fn void test() { int value = 1; }
	fn void test2() { int value, value2; }
	`
	docId := "x"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds global variable declarations", func(t *testing.T) {
		symbols := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["value"]
		assert.Equal(t, "value", found.GetName(), "Variable name")
		assert.Equal(t, "int", found.GetType().String(), "Variable type")
		assert.Equal(t, idx.NewRange(1, 1, 1, 15), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(1, 5, 1, 10), found.GetIdRange())
	})

	t.Run("finds global pointer variable declarations", func(t *testing.T) {
		line := uint(2)
		symbols := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["character"]
		assert.Equal(t, "character", found.GetName(), "Variable name")
		assert.Equal(t, "char*", found.GetType().String(), "Variable type")
		assert.Equal(t, idx.NewRange(line, 1, line, 17), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(line, 7, line, 16), found.GetIdRange())
	})

	t.Run("finds multiple global variables declared in single sentence", func(t *testing.T) {
		line := uint(3)
		symbols := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["foo"]
		assert.Equal(t, "foo", found.GetName(), "First Variable name")
		assert.Equal(t, "int", found.GetType().String(), "First Variable type")
		assert.Equal(t, idx.NewRange(line, 5, line, 8), found.GetIdRange(), "First variable identifier range")
		assert.Equal(t, idx.NewRange(line, 1, line, 15), found.GetDocumentRange(), "First variable declaration range")

		found = symbols.Get("x").Variables["foo2"]
		assert.Equal(t, "foo2", found.GetName(), "Second variable name")
		assert.Equal(t, "int", found.GetType().String(), "Second variable type")
		assert.Equal(t, idx.NewRange(line, 10, line, 14), found.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, idx.NewRange(line, 1, line, 15), found.GetDocumentRange(), "Second variable declaration range")
	})

	t.Run("finds variables declared inside function", func(t *testing.T) {
		line := uint(4)
		symbols := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["value"]
		assert.Equal(t, "value", variable.GetName(), "variable name")
		assert.Equal(t, "int", variable.GetType().String(), "variable type")
		assert.Equal(t, idx.NewRange(line, 22, line, 27), variable.GetIdRange(), "variable identifier range")
		assert.Equal(t, idx.NewRange(line, 18, line, 32), variable.GetDocumentRange(), "variable declaration range")
	})

	t.Run("finds multiple local variables declared in single sentence", func(t *testing.T) {
		line := uint(5)
		symbols := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test2")
		assert.True(t, function.IsSome())
		variable := function.Get().Variables["value"]
		assert.Equal(t, "value", variable.GetName(), "First Variable name")
		assert.Equal(t, "int", variable.GetType().String(), "First Variable type")
		assert.Equal(t, idx.NewRange(line, 23, line, 28), variable.GetIdRange(), "First variable identifier range")
		assert.Equal(t, idx.NewRange(line, 19, line, 37), variable.GetDocumentRange(), "First variable declaration range")

		variable = function.Get().Variables["value2"]
		assert.Equal(t, "value2", variable.GetName(), "Second variable name")
		assert.Equal(t, "int", variable.GetType().String(), "Second variable type")
		assert.Equal(t, idx.NewRange(line, 30, line, 36), variable.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, idx.NewRange(line, 19, line, 37), variable.GetDocumentRange(), "Second variable declaration range")
	})
}

func TestExtractSymbols_find_constants(t *testing.T) {

	source := `const int A_VALUE = 12;`

	doc := document.NewDocument("docId", source)
	parser := createParser()

	symbols := parser.ParseSymbols(&doc)

	found := symbols.Get("docid").Variables["A_VALUE"]
	assert.Equal(t, "A_VALUE", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType().String(), "Variable type")
	assert.True(t, found.IsConstant())
	assert.Equal(t, idx.NewRange(0, 0, 0, 23), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 10, 0, 17), found.GetIdRange())
}
