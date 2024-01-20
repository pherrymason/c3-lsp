package parser

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
	"testing"
)

func createParser() Parser {
	logger := &commonlog.MockLogger{}
	return NewParser(logger)
}

func TestExtractSymbols_finds_function_root_and_global_variables_declarations(t *testing.T) {
	source := `int value = 1;`
	doc := document.NewDocument("x", "file_3", source)
	parser := createParser()
	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction("main", "file_3", "x", idx.NewRange(0, 0, 0, 14), protocol.CompletionItemKindModule)
	expectedRoot.AddVariables([]idx.Variable{
		idx.NewVariable("value", "int", "file_3", "x", idx.NewRange(0, 4, 0, 9), idx.NewRange(0, 0, 0, 14)),
	})

	assert.Equal(t, expectedRoot, symbols)
}

func TestExtractSymbols_finds_function_root_and_global_enum_declarations(t *testing.T) {
	source := `enum Colors { RED, BLUE, GREEN };`
	doc := document.NewDocument("x", "x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction("main", "x", "x", idx.NewRange(0, 0, 0, 35), protocol.CompletionItemKindModule)
	enum := idx.NewEnum("Colors", "", []idx.Enumerator{}, "x", "x", idx.NewRange(0, 5, 0, 11), idx.NewRange(0, 0, 0, 32))
	enum.RegisterEnumerator("RED", "", idx.NewRange(0, 14, 0, 17))
	enum.RegisterEnumerator("BLUE", "", idx.NewRange(0, 19, 0, 23))
	enum.RegisterEnumerator("GREEN", "", idx.NewRange(0, 25, 0, 30))
	expectedRoot.AddEnum(enum)
	assert.Equal(t, enum, symbols.Enums["Colors"])
}

func TestExtractSymbols_finds_function_root_and_global_enum_with_base_type_declarations(t *testing.T) {
	source := `enum Colors:int { RED, BLUE, GREEN };`
	doc := document.NewDocument("x", "x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction("main", "x", "x", idx.NewRange(0, 0, 0, 35), protocol.CompletionItemKindModule)
	enum := idx.NewEnum("Colors", "", []idx.Enumerator{}, "x", "x", idx.NewRange(0, 5, 0, 11), idx.NewRange(0, 0, 0, 36))
	enum.RegisterEnumerator("RED", "", idx.NewRange(0, 18, 0, 21))
	enum.RegisterEnumerator("BLUE", "", idx.NewRange(0, 23, 0, 27))
	enum.RegisterEnumerator("GREEN", "", idx.NewRange(0, 29, 0, 34))

	expectedRoot.AddEnum(enum)
	assert.Equal(t, enum, symbols.Enums["Colors"])
}

func TestExtractSymbols_finds_function_root_and_global_struct_declarations(t *testing.T) {
	source := `struct MyStructure {
		bool enabled;
		char key;
	}`
	doc := document.NewDocument("x", "x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedStruct := idx.NewStruct("MyStructure", []idx.StructMember{
		idx.NewStructMember("enabled", "bool", idx.NewRange(1, 2, 1, 15)),
		idx.NewStructMember("key", "char", idx.NewRange(2, 2, 2, 11)),
	}, "x", "x", idx.NewRange(0, 7, 0, 18))

	assert.Equal(t, expectedStruct, symbols.Structs["MyStructure"])
}

func TestExtractSymbols_extracts_struct_declaration_with_member_functions(t *testing.T) {
	source := `struct MyStruct{
	int data;
}

fn void MyStruct.init(&self)
{
	*self = {
		.data = 4,
	};
}`

	module := "x"
	docId := "x"
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedStruct := idx.NewStruct("MyStruct", []idx.StructMember{
		idx.NewStructMember("data", "int", idx.NewRange(1, 1, 1, 10)),
	}, module, docId, idx.NewRange(0, 7, 0, 15))

	expectedMethod := idx.NewFunctionBuilder("init", "void", module, docId).
		WithTypeIdentifier("MyStruct").
		WithArgument(
			idx.NewVariableBuilder("self", "", module, docId).
				WithIdentifierRange(4, 23, 4, 27).
				WithDocumentRange(4, 23, 4, 27).
				Build(),
		).
		WithIdentifierRange(4, 17, 4, 21).
		WithDocumentRange(4, 0, 9, 1).
		Build()

	assert.Equal(t, expectedStruct, symbols.Structs["MyStruct"])
	assert.Equal(t, expectedMethod, symbols.ChildrenFunctions["MyStruct.init"])
}

func TestExtractSymbols_finds_function_declaration_identifiers(t *testing.T) {
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
	doc := document.NewDocument(docId, "mod", source)
	parser := createParser()
	tree := parser.ExtractSymbols(&doc)

	function1 := idx.NewFunctionBuilder("test", "void", "mod", "x").
		WithIdentifierRange(0, 8, 0, 12).
		WithDocumentRange(0, 0, 2, 2).
		Build()

	function2 := idx.NewFunctionBuilder("test2", "int", "mod", "x").
		WithArgument(
			idx.NewVariableBuilder("number", "int", "mod", "x").
				WithIdentifierRange(3, 18, 3, 24).
				WithDocumentRange(3, 14, 3, 24).
				Build(),
		).
		WithArgument(
			idx.NewVariableBuilder("ch", "char", "mod", "x").
				WithIdentifierRange(3, 31, 3, 33).
				WithDocumentRange(3, 26, 3, 33).
				Build(),
		).
		WithIdentifierRange(3, 8, 3, 34).
		WithDocumentRange(3, 1, 5, 2).
		Build()

	functionMethod := idx.NewFunctionBuilder("method", "Object*", "mod", "x").
		WithArgument(
			idx.NewVariableBuilder("self", "", "mod", "x").
				WithIdentifierRange(6, 31, 6, 35).
				WithDocumentRange(6, 31, 6, 35).
				Build(),
		).
		WithArgument(
			idx.NewVariableBuilder("pointer", "int*", "mod", "x").
				WithIdentifierRange(6, 42, 6, 49).
				WithDocumentRange(6, 37, 6, 49).
				Build(),
		).
		WithIdentifierRange(6, 23, 6, 29).
		WithDocumentRange(6, 1, 8, 2).
		Build()

	root := idx.NewAnonymousScopeFunction("main", "mod", docId, idx.NewRange(0, 0, 0, 14), protocol.CompletionItemKindModule)
	root.AddFunction(function1)
	root.AddFunction(function2)

	assertSameFunction(t, function1, tree.ChildrenFunctions["test"])
	assertSameFunction(t, function2, tree.ChildrenFunctions["test2"])
	assertSameFunction(t, functionMethod, tree.ChildrenFunctions["UserStruct.method"])
}

func TestExtractSymbols_finds_definition(t *testing.T) {
	source := `
	def Kilo = int;
	def KiloPtr = Kilo*;
	def MyFunction = fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*);
	def MyMap = HashMap(<String, Feature>);
	`
	doc := document.NewDocument("x", "x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedDefKilo := idx.NewDefBuilder("Kilo", "x").
		WithResolvesTo("int").
		WithIdentifierRange(1, 5, 1, 9).
		WithDocumentRange(1, 1, 1, 16).
		Build()

	expectedDefKiloPtr := idx.NewDefBuilder("KiloPtr", "x").
		WithResolvesTo("Kilo*").
		WithIdentifierRange(2, 5, 2, 12).
		WithDocumentRange(2, 1, 2, 21).
		Build()

	expectedDefFunction := idx.NewDefBuilder("MyFunction", "x").
		WithResolvesTo("fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*)").
		WithIdentifierRange(3, 5, 3, 15).
		WithDocumentRange(3, 1, 3, 74).
		Build()

	expectedDefTypeWithGenerics := idx.NewDefBuilder("MyMap", "x").
		WithResolvesTo("HashMap(<String, Feature>)").
		WithIdentifierRange(4, 5, 4, 10).
		WithDocumentRange(4, 1, 4, 40).
		Build()

	assert.Equal(t, expectedDefKilo, symbols.Defs["Kilo"])
	assert.Equal(t, expectedDefKiloPtr, symbols.Defs["KiloPtr"])
	assert.Equal(t, expectedDefFunction, symbols.Defs["MyFunction"])

	assert.Equal(t, expectedDefTypeWithGenerics, symbols.Defs["MyMap"])
}

func TestExtractSymbols_find_macro(t *testing.T) {
	if true {
		t.Skip("Incomplete until defining macros in grammar.js")
	}

	sourceCode := `
	macro void log(LogLevel $level, String format, args...) {
		if (log_level != OFF && $level <= log_level) {
			io::fprintf(&log_file, "[%s] ", $level)!!;
			io::fprintfn(&log_file, format, ...args)!!;
		}
	}`

	_ = document.NewDocument("x", "x", sourceCode)
	//	parser := createParser()
	//	tree := parser.ExtractSymbols(&doc)

	assert.Equal(t, true, true)
}

func dfs(n *sitter.Node, level int) {
	if n == nil {
		return
	}

	// Procesa el nodo actual (puedes imprimir, almacenar en un slice, etc.)
	tabs := strings.Repeat("\t", level)
	fmt.Printf("%sNode", tabs)
	//fmt.Printf("%sPos: %d - %d -> ", tabs, n.StartPoint().Row, n.StartPoint().Column)
	//fmt.Printf("%d - %d\n", n.EndPoint().Row, n.EndPoint().Column)
	fmt.Printf("%sType: %s", tabs, n.Type())
	//fmt.Printf("\tContent: %s", n.C)
	fmt.Printf("\n")

	// Llama recursivamente a DFS para los nodos hijos
	//fmt.Printf("~inside~")
	for i := uint32(0); i < n.ChildCount(); i++ {
		child := n.Child(int(i))
		dfs(child, level+1)
	}
}
