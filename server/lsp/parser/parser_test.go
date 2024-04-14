package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func createParser() Parser {
	logger := &commonlog.MockLogger{}
	return NewParser(logger)
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
	funcMethod, _ := symbols.GetChildrenFunctionByName("MyStruct.init")
	assert.Equal(t, expectedMethod, funcMethod)
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
