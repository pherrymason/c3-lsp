package lsp

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
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

	var1 := indexables.NewVariable(
		"var0",
		"int",
		"x",
		NewRange(0, 4, 0, 7),
		NewRange(0, 4, 0, 7), protocol.CompletionItemKindVariable)

	assert.Equal(t, []indexables.Indexable{
		var1,
		indexables.NewVariable("var1", "int", "x", NewRange(0, 18, 0, 22),
			NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
	}, identifiers)
}

func TestFindIdentifiers_finds_unique_used_identifiers(t *testing.T) {
	source := "int var0 = 3; int var1 = 4; var1 = 2+3;"
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []indexables.Indexable{
		indexables.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		indexables.NewVariable("var1", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
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

	assert.Equal(t, []indexables.Indexable{
		indexables.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		indexables.NewFunction("test", "x", NewRange(0, 18, 0, 22),
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

	assert.Equal(t, []indexables.Indexable{
		indexables.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		indexables.NewVariable("var0", "int", "x", NewRange(0, 18, 0, 22), NewRange(0, 18, 0, 22), protocol.CompletionItemKindVariable),
		indexables.NewFunction("test", "x",
			NewRange(0, 18, 0, 22),
			NewRange(2, 9, 4, 30),
			protocol.CompletionItemKindFunction),
	}, identifiers)
}*/

func TestFindSymbols_finds_function_root_and_global_variables_declarations(t *testing.T) {
	source := `int value = 1;`
	doc := NewDocumentFromString("x", source)

	symbols := FindSymbols(&doc)

	expectedRoot := indexables.NewAnonymousScopeFunction(
		"main",
		"x",
		NewRange(0, 0, 0, 14),
		protocol.CompletionItemKindModule,
	)
	expectedRoot.AddVariables([]indexables.Variable{
		indexables.NewVariable(
			"value",
			"int",
			"x",
			NewRange(0, 4, 0, 9),
			NewRange(0, 4, 0, 9), protocol.CompletionItemKindVariable),
	})

	assert.Equal(t, expectedRoot, symbols)
}

func TestFindSymbols_finds_function_root_and_global_enum_declarations(t *testing.T) {
	source := `enum Colors = { RED, BLUE, GREEN };`
	doc := NewDocumentFromString("x", source)

	symbols := FindSymbols(&doc)

	expectedRoot := indexables.NewAnonymousScopeFunction(
		"main",
		"x",
		NewRange(0, 0, 0, 35),
		protocol.CompletionItemKindModule,
	)
	enum := indexables.NewEnum(
		"Colors",
		"",
		[]indexables.Enumerator{
			indexables.NewEnumerator("RED", "", NewRange(0, 16, 0, 19)),
			indexables.NewEnumerator("BLUE", "", NewRange(0, 21, 0, 25)),
			indexables.NewEnumerator("GREEN", "", NewRange(0, 27, 0, 32)),
		},
		NewRange(0, 0, 0, 0),
		NewRange(0, 5, 0, 11),
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

	function1 := indexables.NewFunction("test", "x",
		NewRange(0, 8, 0, 12),
		NewRange(0, 8, 2, 2),
		protocol.CompletionItemKindFunction)
	function2 := indexables.NewFunction("test2", "x",
		NewRange(3, 9, 3, 14),
		NewRange(3, 9, 5, 2),
		protocol.CompletionItemKindFunction)

	root := indexables.NewAnonymousScopeFunction(
		"main",
		"x",
		protocol.Range{
			Start: protocol.Position{0, 0},
			End:   protocol.Position{0, 14},
		},
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
