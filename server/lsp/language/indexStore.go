package language

import (
	trie "github.com/pherrymason/c3-lsp/lsp/symbol_trie"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
)

type IndexStore struct {
	store *trie.Trie
}

func NewIndexStore() IndexStore {
	return IndexStore{
		store: trie.NewTrie(),
	}
}

func (i *IndexStore) RegisterSymbol(symbol idx.Indexable) {
	i.store.Insert(symbol)
}

func (i *IndexStore) SearchByFQN(query string) []idx.Indexable {
	return i.store.Search(query)
}
