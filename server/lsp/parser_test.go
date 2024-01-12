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

func TestFindSymbols_finds_function_root_and_global_variables_declarations(t *testing.T) {
	source := `int value = 1;`
	doc := NewDocumentFromString("x", source)

	symbols := FindSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 14),
		protocol.CompletionItemKindModule,
	)
	expectedRoot.AddVariables([]idx.Variable{
		idx.NewVariable(
			"value",
			"int",
			"x",
			idx.NewRange(0, 4, 0, 9),
			idx.NewRange(0, 4, 0, 9), protocol.CompletionItemKindVariable),
	})

	assert.Equal(t, expectedRoot, symbols)
}

func TestFindSymbols_finds_function_root_and_global_enum_declarations(t *testing.T) {
	source := `enum Colors = { RED, BLUE, GREEN };`
	doc := NewDocumentFromString("x", source)

	symbols := FindSymbols(&doc)

	expectedRoot := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 35),
		protocol.CompletionItemKindModule,
	)
	enum := idx.NewEnum(
		"Colors",
		"",
		[]idx.Enumerator{
			idx.NewEnumerator("RED", "", idx.NewRange(0, 16, 0, 19)),
			idx.NewEnumerator("BLUE", "", idx.NewRange(0, 21, 0, 25)),
			idx.NewEnumerator("GREEN", "", idx.NewRange(0, 27, 0, 32)),
		},
		idx.NewRange(0, 5, 0, 11),
		idx.NewRange(0, 0, 0, 34),
		"x",
	)
	expectedRoot.AddEnum(&enum)
	assert.Equal(t, &enum, symbols.Enums["Colors"])
}

func TestFindSymbols_finds_function_declaration_identifiers(t *testing.T) {
	source := `fn void test() {
		return 1;
	}
	fn void test2(){
		return 2;
	}
	`
	doc := NewDocumentFromString("x", source)

	tree := FindSymbols(&doc)

	function1 := idx.NewFunction("test", "x",
		idx.NewRange(0, 8, 0, 12),
		idx.NewRange(0, 8, 2, 2),
		protocol.CompletionItemKindFunction)
	function2 := idx.NewFunction("test2", "x",
		idx.NewRange(3, 9, 3, 14),
		idx.NewRange(3, 9, 5, 2),
		protocol.CompletionItemKindFunction)

	root := idx.NewAnonymousScopeFunction(
		"main",
		"x",
		idx.NewRange(0, 0, 0, 14),
		protocol.CompletionItemKindModule,
	)
	root.AddFunction(&function1)
	root.AddFunction(&function2)

	fmt.Println(tree.ChildrenFunctions)
	assert.Equal(t, &function1, tree.ChildrenFunctions["test"], "first function")
	assert.Equal(t, &function2, tree.ChildrenFunctions["test2"], "second function")
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
