package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
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
	char[] message;
	char[4] message2;
	fn void test() { int value = 1; }
	fn void test2() { int value, value2; }
	`
	docId := "x"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds global basic type variable declarations", func(t *testing.T) {
		symbols, pendingToResolve := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["value"]
		assert.Equal(t, "value", found.GetName(), "Variable name")
		assert.Equal(t, "int", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(1, 1, 1, 15), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(1, 5, 1, 10), found.GetIdRange())
		assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(docId)), "Basic types should not be registered as pending to resolve.")
	})

	t.Run("finds global pointer variable declarations", func(t *testing.T) {
		line := uint(2)
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["character"]
		assert.Equal(t, "character", found.GetName(), "Variable name")
		assert.Equal(t, "char*", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 1, line, 17), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(line, 7, line, 16), found.GetIdRange())
	})

	t.Run("finds global variable collection declarations", func(t *testing.T) {
		line := uint(4)
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["message"]
		assert.Equal(t, "message", found.GetName(), "Variable name")
		assert.Equal(t, "char[]", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 1, line, 16), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(line, 8, line, 15), found.GetIdRange())
	})

	t.Run("finds global variable static collection declarations", func(t *testing.T) {
		line := uint(5)
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["message2"]
		assert.Equal(t, "message2", found.GetName(), "Variable name")
		assert.Equal(t, "char[4]", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 1, line, 18), found.GetDocumentRange())
		assert.Equal(t, idx.NewRange(line, 9, line, 17), found.GetIdRange())
	})

	t.Run("finds multiple global variables declared in single sentence", func(t *testing.T) {
		line := uint(3)
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["foo"]
		assert.Equal(t, "foo", found.GetName(), "First Variable name")
		assert.Equal(t, "int", found.GetType().String(), "First Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 5, line, 8), found.GetIdRange(), "First variable identifier range")
		assert.Equal(t, idx.NewRange(line, 1, line, 15), found.GetDocumentRange(), "First variable declaration range")

		found = symbols.Get("x").Variables["foo2"]
		assert.Equal(t, "foo2", found.GetName(), "Second variable name")
		assert.Equal(t, "int", found.GetType().String(), "Second variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 10, line, 14), found.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, idx.NewRange(line, 1, line, 15), found.GetDocumentRange(), "Second variable declaration range")
	})

	t.Run("finds variables declared inside function", func(t *testing.T) {
		line := uint(6)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["value"]
		assert.Equal(t, "value", variable.GetName(), "variable name")
		assert.Equal(t, "int", variable.GetType().String(), "variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 22, line, 27), variable.GetIdRange(), "variable identifier range")
		assert.Equal(t, idx.NewRange(line, 18, line, 32), variable.GetDocumentRange(), "variable declaration range")
	})

	t.Run("finds multiple local variables declared in single sentence", func(t *testing.T) {
		line := uint(7)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test2")
		assert.True(t, function.IsSome())
		variable := function.Get().Variables["value"]
		assert.Equal(t, "value", variable.GetName(), "First Variable name")
		assert.Equal(t, "int", variable.GetType().String(), "First Variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 23, line, 28), variable.GetIdRange(), "First variable identifier range")
		assert.Equal(t, idx.NewRange(line, 19, line, 37), variable.GetDocumentRange(), "First variable declaration range")

		variable = function.Get().Variables["value2"]
		assert.Equal(t, "value2", variable.GetName(), "Second variable name")
		assert.Equal(t, "int", variable.GetType().String(), "Second variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(line, 30, line, 36), variable.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, idx.NewRange(line, 19, line, 37), variable.GetDocumentRange(), "Second variable declaration range")
	})
}

func TestExtractSymbols_find_constants(t *testing.T) {

	source := `const int A_VALUE = 12;`

	doc := document.NewDocument("docId", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	found := symbols.Get("docid").Variables["A_VALUE"]
	assert.Equal(t, "A_VALUE", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType().String(), "Variable type")
	assert.True(t, found.IsConstant())
	assert.Equal(t, idx.NewRange(0, 0, 0, 23), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(0, 10, 0, 17), found.GetIdRange())
}

func TestExtractSymbols_find_variables_flag_pending_to_resolve(t *testing.T) {
	t.Run("resolves basic type declaration should not flag type as pending to be resolved", func(t *testing.T) {
		source := `int value = 1;`
		docId := "x"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, pendingToResolve := parser.ParseSymbols(&doc)

		found := symbols.Get(docId).Variables["value"]
		assert.NotNil(t, found)

		assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(docId)), "Basic types should not be registered as pending to resolve.")
	})
}
