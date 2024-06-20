package symbol_trie

import (
	"cmp"
	"slices"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func sort(items []symbols.Indexable) []symbols.Indexable {
	slices.SortFunc(items, func(a, b symbols.Indexable) int {
		return cmp.Compare(strings.ToLower(a.GetFQN()), strings.ToLower(b.GetFQN()))
	})

	return items
}

func TestTrie(t *testing.T) {
	trie := NewTrie()
	docId := "doc"
	strukt := symbols.NewStructBuilder("structName", "app", &docId).Build()

	fun2 := symbols.NewFunctionBuilder("method1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	fun3 := symbols.NewFunctionBuilder("method2", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	fun4 := symbols.NewFunctionBuilder("tearPot", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	funa := symbols.NewFunctionBuilder("anotherMethod1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("anotherStruct").Build()

	trie.Insert(strukt)
	trie.Insert(fun2)
	trie.Insert(fun3)
	trie.Insert(fun4)
	trie.Insert(funa)

	t.Run("Exact search", func(t *testing.T) {
		exactSearch := trie.Search("app::structName")

		assert.Equal(t, 1, len(exactSearch))
		assert.Equal(t, "app::structName", exactSearch[0].GetFQN())
	})

	t.Run("Find all children of parent", func(t *testing.T) {
		result := trie.Search("app::structName.")
		result = sort(result)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "app::structName.method1", result[0].GetFQN())
		assert.Equal(t, "app::structName.method2", result[1].GetFQN())
		assert.Equal(t, "app::structName.tearPot", result[2].GetFQN())
	})

	t.Run("Find all children starting with t", func(t *testing.T) {
		prefixSearch := trie.Search("app::structName.t*")

		assert.Equal(t, 1, len(prefixSearch))
		assert.Equal(t, "app::structName.tearPot", prefixSearch[0].GetFQN())
	})
}

func TestTrie_with_empty_nodes(t *testing.T) {
	trie := NewTrie()

	docId := "doc"

	fun2 := symbols.NewFunctionBuilder("method1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	fun3 := symbols.NewFunctionBuilder("method2", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	fun4 := symbols.NewFunctionBuilder("tearPot", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	funa := symbols.NewFunctionBuilder("anotherMethod1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("anotherStruct").Build()

	//trie.Insert(&strukt)
	trie.Insert(fun2)
	trie.Insert(fun3)
	trie.Insert(fun4)
	trie.Insert(funa)

	t.Run("Exact search", func(t *testing.T) {
		exactSearch := trie.Search("app::structName")

		assert.Equal(t, 0, len(exactSearch))
		//assert.Equal(t, "app::structName", exactSearch[0].GetFQN())
	})

	t.Run("Find all children of parent", func(t *testing.T) {
		result := trie.Search("app::structName.")
		result = sort(result)

		assert.Equal(t, 3, len(result))
		assert.Equal(t, "app::structName.method1", result[0].GetFQN())
		assert.Equal(t, "app::structName.method2", result[1].GetFQN())
		assert.Equal(t, "app::structName.tearPot", result[2].GetFQN())
	})

	t.Run("Find all children starting with t", func(t *testing.T) {
		prefixSearch := trie.Search("app::structName.t*")

		assert.Equal(t, 1, len(prefixSearch))
		assert.Equal(t, "app::structName.tearPot", prefixSearch[0].GetFQN())
	})
}

func TestTrie_clearing_by_tag(t *testing.T) {
	trie := NewTrie()
	docId := "doc"

	fun := symbols.NewFunctionBuilder("method1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	trie.Insert(fun)

	trie.ClearByTag(docId)
	assert.Equal(t, 0, len(trie.root.children))
}

func TestTrie_clearing_by_tag_should_keep_nodes_with_children(t *testing.T) {
	trie := NewTrie()

	docId := "doc"
	fun := symbols.NewFunctionBuilder("method1", symbols.NewTypeFromString("void", "app"), "app", &docId).WithTypeIdentifier("structName").Build()
	trie.Insert(fun)

	docIdB := "different-doc"
	fun2 := symbols.NewFunctionBuilder("method2", symbols.NewTypeFromString("void", "app"), "app", &docIdB).WithTypeIdentifier("structName").Build()
	trie.Insert(fun2)

	trie.ClearByTag("different-doc")
	assert.Equal(t, 1, len(trie.Search("app::structName::method1")))
	assert.Equal(t, 0, len(trie.Search("app::structName::method2")))
}
