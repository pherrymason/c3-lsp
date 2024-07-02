package project_state

import (
	trie "github.com/pherrymason/c3-lsp/internal/lsp/symbol_trie"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
)

// IndexStore could be removed? is used just as a translation layer
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

func (i *IndexStore) ClearByTag(tag string) {
	i.store.ClearByTag(tag)
}

func (i *IndexStore) SearchByFQN(query string) []idx.Indexable {
	return i.store.Search(query)
}
