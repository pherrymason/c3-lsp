package lsp

import (
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
	"testing"
)

func TestFindIdentifiers_finds_used_identifiers(t *testing.T) {
	source := "int var0 = 3; int var1 = 4;"
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []Identifier{
		{
			name:                "var0",
			kind:                protocol.CompletionItemKindVariable,
			declarationPosition: protocol.Position{Character: 4},
		},
		{
			name:                "var1",
			kind:                protocol.CompletionItemKindVariable,
			declarationPosition: protocol.Position{Character: 18},
		},
	}, identifiers)
}

func TestFindIdentifiers_finds_unique_used_identifiers(t *testing.T) {
	source := "int var0 = 3; int var1 = 4; var1 = 2+3;"
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []Identifier{
		{
			name:                "var0",
			kind:                protocol.CompletionItemKindVariable,
			declarationPosition: protocol.Position{Character: 4},
		},
		{
			name:                "var1",
			kind:                protocol.CompletionItemKindVariable,
			declarationPosition: protocol.Position{Character: 18},
		},
	}, identifiers)
}

func TestFindIdentifiers_finds_function_declaration_identifiers(t *testing.T) {
	source := `fn void test() {
		return 1;
	}
	`
	doc := NewDocumentFromString("x", source)

	identifiers := FindIdentifiers(&doc)

	assert.Equal(t, []Identifier{
		{
			name:                "test",
			kind:                protocol.CompletionItemKindFunction,
			declarationPosition: protocol.Position{Character: 8},
		},
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

	assert.Equal(t, []Identifier{
		{
			name:                "var0",
			kind:                protocol.CompletionItemKindVariable,
			declarationPosition: protocol.Position{Line: 1, Character: 5},
		},
		{
			name:                "test",
			kind:                protocol.CompletionItemKindFunction,
			declarationPosition: protocol.Position{Line: 2, Character: 9},
		},
	}, identifiers)
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
