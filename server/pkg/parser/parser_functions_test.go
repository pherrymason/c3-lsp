package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
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

func TestExtractSymbols_Functions_returns_optional_type(t *testing.T) {
	source := `fn usz! test() {
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
		assert.Equal(t, "usz!", fn.Get().GetReturnType().String(), "Return type")
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
		assert.Nil(t, fn.Get().GetDocComment())
		assert.Equal(t, idx.NewRange(0, 8, 0, 19), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 0, 78), fn.Get().GetDocumentRange())
	})

	t.Run("Finds function with doc comment", func(t *testing.T) {
		source := `<*
			abc
		*>
		fn void init_window(int width, int height, char* title) @extern("InitWindow");`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("init_window")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "init_window", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(3, 10, 3, 21), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(3, 2, 3, 80), fn.Get().GetDocumentRange())
		assert.Equal(t, "abc", fn.Get().GetDocComment().GetBody())
		assert.Equal(t, "abc", fn.Get().GetDocComment().DisplayBodyWithContracts())
	})

	t.Run("Resolves function with unnamed parameters correctly", func(t *testing.T) {
		source := `fn void init_window(int, int, char*) @extern("InitWindow");`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("init_window")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "init_window", fn.Get().GetName(), "Function name")

		arg0 := fn.Get().Variables["$arg0"]
		assert.Equal(t, "$arg0", arg0.GetName())
		assert.Equal(t, "int", arg0.GetType().String())

		arg1 := fn.Get().Variables["$arg1"]
		assert.Equal(t, "$arg1", arg1.GetName())
		assert.Equal(t, "int", arg1.GetType().String())

		arg2 := fn.Get().Variables["$arg2"]
		assert.Equal(t, "$arg2", arg2.GetName())
		assert.Equal(t, "char*", arg2.GetType().String())
	})

	t.Run("Resolves function with some unnamed parameters correctly", func(t *testing.T) {
		source := `fn void init_window(int width, int height, char*) @extern("InitWindow");`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("init_window")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "init_window", fn.Get().GetName(), "Function name")

		arg0 := fn.Get().Variables["width"]
		assert.Equal(t, "width", arg0.GetName())
		assert.Equal(t, "int", arg0.GetType().String())

		arg1 := fn.Get().Variables["height"]
		assert.Equal(t, "height", arg1.GetName())
		assert.Equal(t, "int", arg1.GetType().String())

		arg2 := fn.Get().Variables["$arg2"]
		assert.Equal(t, "$arg2", arg2.GetName())
		assert.Equal(t, "char*", arg2.GetType().String())
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
		assert.Nil(t, fn.Get().GetDocComment())
	})

	t.Run("Finds function with simple doc comment", func(t *testing.T) {
		source := `<*
			abc

			def

			ghi
			jkl
		*>
		fn void test(int number, char ch, int* pointer) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		expectedDoc := `abc

def

ghi
jkl`

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(8, 10, 8, 14), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(8, 2, 10, 3), fn.Get().GetDocumentRange())
		assert.Equal(t, expectedDoc, fn.Get().GetDocComment().GetBody())
		assert.Equal(t, expectedDoc, fn.Get().GetDocComment().DisplayBodyWithContracts())
	})

	t.Run("Finds function with doc comment with contracts", func(t *testing.T) {
		source := `<*
			Hello world.
			Hello world.

			@pure
			@param [in] pointer
			@require number > 0, number < 1000 : "invalid number"
			@ensure return == 1
		*>
		fn void test(int number, char ch, int* pointer) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(9, 10, 9, 14), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(9, 2, 11, 3), fn.Get().GetDocumentRange())
		assert.Equal(t, `Hello world.
Hello world.`, fn.Get().GetDocComment().GetBody())
		assert.Equal(t, `Hello world.
Hello world.

**@pure**

**@param** [in] pointer

**@require** number > 0, number < 1000 : "invalid number"

**@ensure** return == 1`, fn.Get().GetDocComment().DisplayBodyWithContracts())
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
