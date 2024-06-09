package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
)

func TestExtractSymbols_Functions_Definitions(t *testing.T) {
	source := `fn void test() {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 12), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.Get().GetDocumentRange())
	})
}

func TestExtractSymbols_Functions_Declaration(t *testing.T) {
	source := `fn void init_window(int width, int height, char* title) @extern("InitWindow");`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("init_window")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "init_window", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 19), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 0, 78), fn.Get().GetDocumentRange())
	})
}

func TestExtractSymbols_FunctionsWithArguments(t *testing.T) {
	source := `fn void test(int number, char ch, int* pointer) {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 12), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.Get().GetDocumentRange())
	})

	t.Run("Finds function arguments", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")

		variable := fn.Get().Variables["number"]
		assert.Equal(t, "number", variable.GetName())
		assert.Equal(t, "int", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 17, 0, 23), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 13, 0, 23), variable.GetDocumentRange())

		variable = fn.Get().Variables["ch"]
		assert.Equal(t, "ch", variable.GetName())
		assert.Equal(t, "char", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 30, 0, 32), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 25, 0, 32), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 39, 0, 46), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 34, 0, 46), variable.GetDocumentRange())
	})
}

func TestExtractSymbols_StructMemberFunctionWithArguments(t *testing.T) {
	source := `fn Object* UserStruct.method(self, int* pointer) {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds method", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, fn.IsSome(), "Method was not found")
		assert.Equal(t, "Object", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, "Object*", fn.Get().GetReturnType().String(), "Return type")
		assert.Equal(t, "UserStruct.method", fn.Get().GetName())
		assert.Equal(t, idx.NewRange(0, 22, 0, 28), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.Get().GetDocumentRange())
	})

	t.Run("Finds method arguments", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, fn.IsSome(), "Method was not found")

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 29, 0, 33), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 33), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 40, 0, 47), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 35, 0, 47), variable.GetDocumentRange())

	})

	t.Run("Finds method arguments, where member reference is a pointer", func(t *testing.T) {
		t.Skip("Incomplete until detecting & in self argument")
		source := `fn Object* UserStruct.method(&self, int* pointer) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.method")
		assert.True(t, fn.IsSome(), "Method was not found")

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType())
		assert.Equal(t, idx.NewRange(0, 41, 0, 48), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 36, 0, 48), variable.GetDocumentRange())

	})
}

func TestExtractSymbols_flags_types_as_pending_to_be_resolved(t *testing.T) {
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
