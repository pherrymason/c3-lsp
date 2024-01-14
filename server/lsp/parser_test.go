package lsp

import (
	"fmt"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
	"testing"
)

/*
func TestFindIdentifiers_finds_used_identifiers(t *testing.T) {
	source := "int var0 = 3; int var1 = 4;"
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	var1 := idx.NewVariable(
		"var0",
		"int",
		"x",
		NewRange(0, 4, 0, 7),
		NewRange(0, 4, 0, 7), protocol.CompletionItemKindVariable)

	assert.Equal(t, []idx.Indexable{
		var1,
		idx.NewVariable("var1", "int", "x", NewRange(0, 18, 0, 22),
			NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
	}, identifiers)
}

func TestFindIdentifiers_finds_unique_used_identifiers(t *testing.T) {
	source := "int var0 = 3; int var1 = 4; var1 = 2+3;"
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []idx.Indexable{
		idx.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		idx.NewVariable("var1", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
	}, identifiers)
}

func TestFindIdentifiers_should_find_different_types(t *testing.T) {
	source := `
	int var0 = 2;
	fn void test() {
		return 1;
	}
	`
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []idx.Indexable{
		idx.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		idx.NewFunction("test", "x", NewRange(0, 18, 0, 22),
			NewRange(2, 9, 4, 40),
			protocol.CompletionItemKindFunction),
	}, identifiers)
}

func TestFindIdentifiers_should_assign_different_scopes_to_same_name_identifiers(t *testing.T) {
	source := `
	int var0 = 2;
	fn void test() {
		int var0 = 3;
		return 1;
	}
	`
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []idx.Indexable{
		idx.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		idx.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		idx.NewFunction("test", "x",
			NewRange(0, 18, 0, 22),
			NewRange(2, 9, 4, 30),
			protocol.CompletionItemKindFunction),
	}, identifiers)
}*/

func TestExtractSymbols_finds_function_root_and_global_variables_declarations(t *testing.T) {
	source := `int value = 1;`
	doc := NewDocumentFromString("x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 14),
		protocol.CompletionItemKindModule,
	)
	expectedRoot.AddVariables([]idx.Variable{
		idx.NewVariable("value", "int", "x", idx.NewRange(0, 4, 0, 9), idx.NewRange(0, 0, 0, 14)),
	})

	assert.Equal(t, expectedRoot, symbols)
}

func TestExtractSymbols_finds_function_root_and_global_enum_declarations(t *testing.T) {
	source := `enum Colors { RED, BLUE, GREEN };`
	doc := NewDocumentFromString("x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 35),
		protocol.CompletionItemKindModule,
	)
	enum := idx.NewEnum(
		"Colors",
		"",
		[]idx.Enumerator{},
		idx.NewRange(0, 5, 0, 11),
		idx.NewRange(0, 0, 0, 32),
		"x",
	)
	enum.RegisterEnumerator("RED", "", idx.NewRange(0, 14, 0, 17))
	enum.RegisterEnumerator("BLUE", "", idx.NewRange(0, 19, 0, 23))
	enum.RegisterEnumerator("GREEN", "", idx.NewRange(0, 25, 0, 30))
	expectedRoot.AddEnum(&enum)
	assert.Equal(t, &enum, symbols.Enums["Colors"])
}

func TestExtractSymbols_finds_function_root_and_global_enum_with_base_type_declarations(t *testing.T) {
	source := `enum Colors:int { RED, BLUE, GREEN };`
	doc := NewDocumentFromString("x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 35),
		protocol.CompletionItemKindModule,
	)
	enum := idx.NewEnum(
		"Colors",
		"",
		[]idx.Enumerator{},
		idx.NewRange(0, 5, 0, 11),
		idx.NewRange(0, 0, 0, 36),
		"x",
	)
	enum.RegisterEnumerator("RED", "", idx.NewRange(0, 18, 0, 21))
	enum.RegisterEnumerator("BLUE", "", idx.NewRange(0, 23, 0, 27))
	enum.RegisterEnumerator("GREEN", "", idx.NewRange(0, 29, 0, 34))

	expectedRoot.AddEnum(&enum)
	assert.Equal(t, &enum, symbols.Enums["Colors"])
}

func TestExtractSymbols_finds_function_root_and_global_struct_declarations(t *testing.T) {
	source := `struct MyStructure {
		bool enabled;
		char key;
	}`
	doc := NewDocumentFromString("x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedStruct := idx.NewStruct(
		"MyStructure",
		[]idx.StructMember{
			idx.NewStructMember("enabled", "bool", idx.NewRange(1, 2, 1, 15)),
			idx.NewStructMember("key", "char", idx.NewRange(2, 2, 2, 11)),
		},
		"x",
		idx.NewRange(0, 7, 0, 18),
	)

	assert.Equal(t, expectedStruct, symbols.Structs["MyStructure"])
}

func TestExtractSymbols_finds_function_declaration_identifiers(t *testing.T) {
	source := `fn void test() {
		return 1;
	}
	fn int test2(int number, char ch){
		return 2;
	}`
	docId := "x"
	doc := NewDocumentFromString(docId, source)
	parser := createParser()
	tree := parser.ExtractSymbols(&doc)

	function1 := idx.NewFunction("test", "void", nil, docId, idx.NewRange(0, 8, 0, 12), idx.NewRange(0, 0, 2, 2), protocol.CompletionItemKindFunction)
	function2 := idx.NewFunction("test2", "int", []string{"number", "ch"}, docId, idx.NewRange(3, 8, 3, 34), idx.NewRange(3, 1, 5, 2), protocol.CompletionItemKindFunction)

	var1 := idx.NewVariable("number", "int", docId,
		idx.NewRange(3, 18, 3, 24),
		idx.NewRange(3, 14, 3, 24))
	var2 := idx.NewVariable("ch", "char", docId,
		idx.NewRange(3, 31, 3, 33),
		idx.NewRange(3, 26, 3, 33))
	function2.AddVariables([]idx.Variable{var1, var2})

	root := idx.NewAnonymousScopeFunction(
		"main",
		docId,
		idx.NewRange(0, 0, 0, 14),
		protocol.CompletionItemKindModule,
	)
	root.AddFunction(&function1)
	root.AddFunction(&function2)

	assertSameFunction(t, &function1, tree.ChildrenFunctions["test"])
	assertSameFunction(t, &function2, tree.ChildrenFunctions["test2"])
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func assertSameVariable(t *testing.T, expected idx.Variable, actual idx.Variable) {
	assert.Equal(t, expected.GetName(), actual.GetName())
	assert.Equal(t, expected.GetType(), actual.GetType(), expected.GetName())
	assert.Equal(t, expected.GetDocumentURI(), actual.GetDocumentURI(), expected.GetName())
	assertSameRange(t, expected.GetDeclarationRange(), actual.GetDeclarationRange(), fmt.Sprint("Variable  declaration range:", expected.GetName()))
	assertSameRange(t, expected.GetDocumentRange(), actual.GetDocumentRange(), fmt.Sprint("Variable document range:", expected.GetName()))
	assert.Equal(t, expected.GetKind(), actual.GetKind(), expected.GetName())
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

	_ = NewDocumentFromString("x", sourceCode)
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
