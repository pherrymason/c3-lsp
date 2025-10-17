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
	source := `fn usz? test() {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		docModule := symbols.Get("docid")
		if docModule == nil {
			t.Fatalf("couldnt find docid")
		}
		fn := docModule.GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "usz?", fn.Get().GetReturnType().String(), "Return type")
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

		docModule := symbols.Get("docid")
		if docModule == nil {
			t.Fatalf("couldnt find docid")
		}
		fn := docModule.GetChildrenFunctionByName("init_window")
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
		fnDoc := fn.Get().GetDocComment()
		if fnDoc == nil {
			t.Fatalf("function doc is nil")
		}
		assert.Equal(t, "abc", fnDoc.GetBody())
		assert.Equal(t, "abc", fnDoc.DisplayBodyWithContracts())
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

		arg0, found := fn.Get().Variables["$arg#0"]
		if !found {
			t.Fatalf("couldnt find variable $arg#0")
		}
		assert.Equal(t, "$arg#0", arg0.GetName())
		assert.Equal(t, "int", arg0.GetType().String())

		arg1, found := fn.Get().Variables["$arg#1"]
		if !found {
			t.Fatalf("couldnt find variable $arg#1")
		}
		assert.Equal(t, "$arg#1", arg1.GetName())
		assert.Equal(t, "int", arg1.GetType().String())

		arg2, found := fn.Get().Variables["$arg#2"]
		if !found {
			t.Fatalf("couldnt find variable $arg#1")
		}
		assert.Equal(t, "$arg#2", arg2.GetName())
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

		arg0, found := fn.Get().Variables["width"]
		if !found {
			t.Fatalf("Couldnt find variable width")
		}
		assert.Equal(t, "width", arg0.GetName())
		assert.Equal(t, "int", arg0.GetType().String())

		arg1, found := fn.Get().Variables["height"]
		if !found {
			t.Fatalf("Couldnt find variable height")
		}
		assert.Equal(t, "height", arg1.GetName())
		assert.Equal(t, "int", arg1.GetType().String())

		arg2, found := fn.Get().Variables["$arg#2"]
		if !found {
			t.Fatalf("Couldnt find variable $arg#2")
		}
		assert.Equal(t, "$arg#2", arg2.GetName())
		assert.Equal(t, "char*", arg2.GetType().String())
	})
}

func TestExtractSymbols_FunctionsWithArguments(t *testing.T) {
	source := `fn void test(int number = 10, char ch, int* pointer) {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn void test(int number = 10, char ch, int* pointer)", fn.Get().GetHoverInfo(), "Function signature")
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

	t.Run("Finds function with doc comment with body and contracts", func(t *testing.T) {
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

	t.Run("Finds function with doc comment with only contracts", func(t *testing.T) {
		source := `<*
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
		assert.Equal(t, idx.NewRange(6, 10, 6, 14), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(6, 2, 8, 3), fn.Get().GetDocumentRange())
		assert.Equal(t, "", fn.Get().GetDocComment().GetBody())
		assert.Equal(t, `**@pure**

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
		assert.Equal(t, "10", variable.Arg.Default.Get())
		assert.Equal(t, idx.NewRange(0, 17, 0, 23), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 13, 0, 28), variable.GetDocumentRange())

		variable = fn.Get().Variables["ch"]
		assert.Equal(t, "ch", variable.GetName())
		assert.Equal(t, "char", variable.GetType().String())
		assert.True(t, variable.Arg.Default.IsNone())
		assert.Equal(t, idx.NewRange(0, 35, 0, 37), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 30, 0, 37), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 44, 0, 51), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 39, 0, 51), variable.GetDocumentRange())
	})

	t.Run("Finds function with empty argument names", func(t *testing.T) {
		source := `<* func *>
		fn void test(int, char, int*, ...);`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("test")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn void test(int, char, int*, ...)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "test", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(1, 10, 1, 14), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(1, 2, 1, 37), fn.Get().GetDocumentRange())
		assert.Equal(t, "func", fn.Get().GetDocComment().GetBody())
		assert.Equal(t, "func", fn.Get().GetDocComment().DisplayBodyWithContracts())
	})
}

func TestExtractSymbols_MacrosWithArguments(t *testing.T) {
	source := `macro @test(char name = 'a', $ch, #expr) {
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds macro", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("@test")
		assert.True(t, fn.IsSome(), "Macro was not found")
		assert.Equal(t, "macro @test(char name = 'a', $ch, #expr)", fn.Get().GetHoverInfo(), "Macro signature")
		assert.Equal(t, "@test", fn.Get().GetName(), "Macro name")
		assert.Equal(t, "", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 6, 0, 11), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 2), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())
	})

	t.Run("Finds macro with simple doc comment", func(t *testing.T) {
		source := `<*
			abc

			def

			ghi
			jkl
		*>
		macro int @test(char name = 'a', $ch, #expr) {
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

		fn := symbols.Get("docid").GetChildrenFunctionByName("@test")
		assert.True(t, fn.IsSome(), "Macro was not found")
		assert.Equal(t, "@test", fn.Get().GetName(), "Macro name")
		assert.Equal(t, "int", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(8, 12, 8, 17), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(8, 2, 10, 3), fn.Get().GetDocumentRange())
		assert.Equal(t, expectedDoc, fn.Get().GetDocComment().GetBody())
		assert.Equal(t, expectedDoc, fn.Get().GetDocComment().DisplayBodyWithContracts())
	})

	t.Run("Finds macro arguments", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("@test")
		assert.True(t, fn.IsSome(), "Macro was not found")

		variable := fn.Get().Variables["name"]
		assert.Equal(t, "name", variable.GetName())
		assert.Equal(t, "char", variable.GetType().String())
		assert.Equal(t, "'a'", variable.Arg.Default.Get())
		assert.Equal(t, idx.NewRange(0, 17, 0, 21), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 12, 0, 27), variable.GetDocumentRange())

		variable = fn.Get().Variables["$ch"]
		assert.Equal(t, "$ch", variable.GetName())
		assert.Equal(t, "", variable.GetType().String())
		assert.True(t, variable.Arg.Default.IsNone())
		assert.Equal(t, idx.NewRange(0, 29, 0, 32), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 32), variable.GetDocumentRange())

		variable = fn.Get().Variables["#expr"]
		assert.Equal(t, "#expr", variable.GetName())
		assert.Equal(t, "", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 34, 0, 39), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 34, 0, 39), variable.GetDocumentRange())
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
		assert.Equal(t, "UserStruct*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 34), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 41, 0, 48), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 36, 0, 48), variable.GetDocumentRange())

	})
}

func TestExtractSymbols_StructMemberMacroWithArguments(t *testing.T) {
	source := `macro Object* UserStruct.@method(self, int* pointer; @body) {
		@body();
		return 1;
	}`
	docId := "docId"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("Finds method macro", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.@method")
		assert.True(t, fn.IsSome(), "Method was not found")
		assert.Equal(t, "Object", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, "Object*", fn.Get().GetReturnType().String(), "Return type")
		assert.Equal(t, "UserStruct.@method", fn.Get().GetName())
		assert.Equal(t, idx.NewRange(0, 25, 0, 32), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 3, 2), fn.Get().GetDocumentRange())
	})

	t.Run("Finds method macro arguments", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.@method")
		assert.True(t, fn.IsSome(), "Method was not found")

		assert.Equal(t, "macro Object* UserStruct.@method(UserStruct self, int* pointer; @body)", fn.Get().GetHoverInfo())

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 33, 0, 37), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 33, 0, 37), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 44, 0, 51), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 39, 0, 51), variable.GetDocumentRange())

		variable = fn.Get().Variables["@body"]
		assert.Equal(t, "@body", variable.GetName())
		assert.Equal(t, "fn void()", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 53, 0, 58), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 53, 0, 58), variable.GetDocumentRange())
	})

	t.Run("Finds method macro arguments, where @body has parameters", func(t *testing.T) {
		source := `macro Object* UserStruct.@method(self, int* pointer; @body(&something, int a, float* b)) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.@method")
		assert.True(t, fn.IsSome(), "Method was not found")

		assert.Equal(t, "macro Object* UserStruct.@method(UserStruct self, int* pointer; @body(&something, int a, float* b))", fn.Get().GetHoverInfo())

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 33, 0, 37), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 33, 0, 37), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 44, 0, 51), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 39, 0, 51), variable.GetDocumentRange())

		variable = fn.Get().Variables["@body"]
		assert.Equal(t, "@body", variable.GetName())
		assert.Equal(t, "fn void(&something, int a, float* b)", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 53, 0, 58), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 53, 0, 87), variable.GetDocumentRange())

	})

	t.Run("Finds method macro arguments, where member reference is a pointer", func(t *testing.T) {
		source := `macro Object* UserStruct.@method(&self, int* pointer; @body) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.@method")
		assert.True(t, fn.IsSome(), "Method was not found")

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 34, 0, 38), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 33, 0, 38), variable.GetDocumentRange())

		variable = fn.Get().Variables["pointer"]
		assert.Equal(t, "pointer", variable.GetName())
		assert.Equal(t, "int*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 45, 0, 52), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 40, 0, 52), variable.GetDocumentRange())

		variable = fn.Get().Variables["@body"]
		assert.Equal(t, "@body", variable.GetName())
		assert.Equal(t, "fn void()", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 54, 0, 59), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 54, 0, 59), variable.GetDocumentRange())

	})

	t.Run("Finds method macro-specific arguments", func(t *testing.T) {
		source := `macro Object* UserStruct.@method(&self, $comp, #expr; @body) {
			return 1;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("UserStruct.@method")
		assert.True(t, fn.IsSome(), "Method was not found")

		variable := fn.Get().Variables["self"]
		assert.Equal(t, "self", variable.GetName())
		assert.Equal(t, "UserStruct*", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 34, 0, 38), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 33, 0, 38), variable.GetDocumentRange())

		variable = fn.Get().Variables["$comp"]
		assert.Equal(t, "$comp", variable.GetName())
		assert.Equal(t, "", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 40, 0, 45), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 40, 0, 45), variable.GetDocumentRange())

		variable = fn.Get().Variables["#expr"]
		assert.Equal(t, "#expr", variable.GetName())
		assert.Equal(t, "", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 47, 0, 52), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 47, 0, 52), variable.GetDocumentRange())

		variable = fn.Get().Variables["@body"]
		assert.Equal(t, "@body", variable.GetName())
		assert.Equal(t, "fn void()", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 54, 0, 59), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 54, 0, 59), variable.GetDocumentRange())

	})
}

func TestExtractSymbols_FunctionsWithVariableArguments(t *testing.T) {
	t.Run("Finds arguments with single-typed vararg", func(t *testing.T) {
		source := `fn bool va_singletyped(int... args) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_singletyped")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn bool va_singletyped(int... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_singletyped", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 22), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "int[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 30, 0, 34), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 23, 0, 34), variable.GetDocumentRange())
	})

	t.Run("Finds arguments with any-ref-typed vararg", func(t *testing.T) {
		source := `fn bool va_variants_explicit(any*... args) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_variants_explicit")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn bool va_variants_explicit(any*... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_variants_explicit", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 28), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "any*[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 37, 0, 41), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 41), variable.GetDocumentRange())
	})

	t.Run("Finds arguments with implicit any-ref-typed vararg", func(t *testing.T) {
		source := `fn bool va_variants_implicit(args...) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_variants_implicit")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn bool va_variants_implicit(any*... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_variants_implicit", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 8, 0, 28), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "any*[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 29, 0, 33), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 29, 0, 36), variable.GetDocumentRange())
	})

	t.Run("Finds arguments with C-style vararg", func(t *testing.T) {
		source := `extern fn void va_untyped(...);`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_untyped")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "fn void va_untyped(...)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_untyped", fn.Get().GetName(), "Function name")
		assert.Equal(t, "void", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 15, 0, 25), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 7, 0, 31), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["$arg#0"]
		assert.Equal(t, "$arg#0", variable.GetName())
		assert.Equal(t, "any*[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 0, 0, 0), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 26, 0, 29), variable.GetDocumentRange())
	})
}

func TestExtractSymbols_MacrosWithVariableArguments(t *testing.T) {
	t.Run("Finds arguments with single-typed vararg", func(t *testing.T) {
		source := `macro bool va_singletyped(int... args) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_singletyped")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "macro bool va_singletyped(int... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_singletyped", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 11, 0, 25), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "int[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 33, 0, 37), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 26, 0, 37), variable.GetDocumentRange())
	})

	t.Run("Finds arguments with any-ref-typed vararg", func(t *testing.T) {
		source := `macro bool va_variants_explicit(any*... args) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_variants_explicit")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "macro bool va_variants_explicit(any*... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_variants_explicit", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 11, 0, 31), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "any*[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 40, 0, 44), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 32, 0, 44), variable.GetDocumentRange())
	})

	t.Run("Finds arguments with implicit any-ref-typed vararg", func(t *testing.T) {
		source := `macro bool va_variants_implicit(args...) {
			return true;
		}`
		docId := "docId"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		fn := symbols.Get("docid").GetChildrenFunctionByName("va_variants_implicit")
		assert.True(t, fn.IsSome(), "Function was not found")
		assert.Equal(t, "macro bool va_variants_implicit(any*... args)", fn.Get().GetHoverInfo(), "Function signature")
		assert.Equal(t, "va_variants_implicit", fn.Get().GetName(), "Function name")
		assert.Equal(t, "bool", fn.Get().GetReturnType().GetName(), "Return type")
		assert.Equal(t, idx.NewRange(0, 11, 0, 31), fn.Get().GetIdRange())
		assert.Equal(t, idx.NewRange(0, 0, 2, 3), fn.Get().GetDocumentRange())
		assert.Nil(t, fn.Get().GetDocComment())

		variable := fn.Get().Variables["args"]
		assert.Equal(t, "args", variable.GetName())
		assert.Equal(t, "any*[]", variable.GetType().String())
		assert.Equal(t, idx.NewRange(0, 32, 0, 36), variable.GetIdRange())
		assert.Equal(t, idx.NewRange(0, 32, 0, 39), variable.GetDocumentRange())
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
		assert.NotNil(t, found, "variables: %v", symbols.Get(docId).Variables)

		assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(docId)), "Basic types should not be registered as pending to resolve.")
	})
}
