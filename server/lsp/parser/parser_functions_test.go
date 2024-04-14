package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
)

func TestExtractSymbols_finds_function_declarations(t *testing.T) {
	source := `fn void test() {
		return 1;
	}
	fn int test2(int number, char ch) {
		return 2;
	}
	fn Object* UserStruct.method(&self, int* pointer) {
		return 1;
	}`
	docId := "x"
	module := "mod"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	tree := parser.ExtractSymbols(&doc)

	function1 := idx.NewFunctionBuilder("test", "void", module, docId).
		WithIdentifierRange(0, 8, 0, 12).
		WithDocumentRange(0, 0, 2, 2).
		Build()

	function2 := idx.NewFunctionBuilder("test2", "int", module, docId).
		WithArgument(
			idx.NewVariableBuilder("number", "int", module, docId).
				WithIdentifierRange(3, 18, 3, 24).
				WithDocumentRange(3, 14, 3, 24).
				Build(),
		).
		WithArgument(
			idx.NewVariableBuilder("ch", "char", module, docId).
				WithIdentifierRange(3, 31, 3, 33).
				WithDocumentRange(3, 26, 3, 33).
				Build(),
		).
		WithIdentifierRange(3, 8, 3, 34).
		WithDocumentRange(3, 1, 5, 2).
		Build()

	functionMethod := idx.NewFunctionBuilder("method", "Object*", module, docId).
		WithArgument(
			idx.NewVariableBuilder("self", "", module, docId).
				WithIdentifierRange(6, 31, 6, 35).
				WithDocumentRange(6, 31, 6, 35).
				Build(),
		).
		WithArgument(
			idx.NewVariableBuilder("pointer", "int*", module, docId).
				WithIdentifierRange(6, 42, 6, 49).
				WithDocumentRange(6, 37, 6, 49).
				Build(),
		).
		WithIdentifierRange(6, 23, 6, 29).
		WithDocumentRange(6, 1, 8, 2).
		Build()
		/*
			root := idx.NewAnonymousScopeFunction("main", module, docId, idx.NewRange(0, 0, 0, 14), protocol.CompletionItemKindModule)
			root.AddFunction(function1)
			root.AddFunction(function2)*/

	found, _ := tree.GetChildrenFunctionByName("test")
	assertSameFunction(t, function1, found, "test")

	found, _ = tree.GetChildrenFunctionByName("test2")
	assertSameFunction(t, function2, found, "test2")

	found, _ = tree.GetChildrenFunctionByName("UserStruct.method")
	assertSameFunction(t, functionMethod, found, "UserStruct.method")
}
