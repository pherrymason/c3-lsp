package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
)

func TestExtractSymbols_Functions(t *testing.T) {
	source := `fn void test() {
		return 1;
	}`
	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("test")
		assert.True(t, found, "Function was not found")
		assert.Equal(t, "test", fn.GetName(), "Function name")
		assert.Equal(t, "void", fn.GetReturnType(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 12), fn.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.GetDocumentRange())
	})
}

func TestExtractSymbols_FunctionsWithArguments(t *testing.T) {
	source := `fn void test(int number, char ch, int* pointer) {
		return 1;
	}`
	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("test")
		assert.True(t, found, "Function was not found")
		assert.Equal(t, "test", fn.GetName(), "Function name")
		assert.Equal(t, "void", fn.GetReturnType(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 12), fn.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.GetDocumentRange())
	})

	t.Run("Finds function arguments", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("test")
		assert.True(t, found, "Function was not found")

		variable := fn.Variables["number"]
		assert.Equal(t, "number", variable.GetName())
		assert.Equal(t, "int", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 17, 0, 23), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 13, 0, 23), variable.GetDocumentRange())

		variable = fn.Variables["ch"]
		assert.Equal(t, "ch", variable.GetName())
		assert.Equal(t, "char", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 30, 0, 32), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 25, 0, 32), variable.GetDocumentRange())

		variable = fn.Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 39, 0, 46), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 34, 0, 46), variable.GetDocumentRange())
	})
}

func TestExtractSymbols_StructMemberFunctionWithArguments(t *testing.T) {
	source := `fn Object* UserStruct.method(self, int* pointer) {
		return 1;
	}`
	module := "x"
	docId := "docId"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("Finds method", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, found, "Method was not found")
		assert.Equal(t, "Object*", fn.GetReturnType(), "Return type")
		assert.Equal(t, "method", fn.GetName())
		assert.Equal(t, idx.NewRange(0, 22, 0, 28), fn.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.GetDocumentRange())
	})

	t.Run("Finds method arguments", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, found, "Method was not found")

		variable := fn.Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 29, 0, 33), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 33), variable.GetDocumentRange())

		variable = fn.Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 40, 0, 47), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 35, 0, 47), variable.GetDocumentRange())

	})

	t.Run("Finds method arguments, where member reference is a pointer", func(t *testing.T) {
		t.Skip("Incomplete until detecting & in self argument")
		source := `fn Object* UserStruct.method(&self, int* pointer) {
			return 1;
		}`
		module := "x"
		docId := "docId"
		doc := document.NewDocument(docId, module, source)
		parser := createParser()
		symbols := parser.ExtractSymbols(&doc)

		fn, found := symbols.GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, found, "Method was not found")

		variable := fn.Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetDocumentRange())

		variable = fn.Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 41, 0, 48), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 36, 0, 48), variable.GetDocumentRange())

	})
}
