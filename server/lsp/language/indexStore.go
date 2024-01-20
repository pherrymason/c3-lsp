package language

import idx "github.com/pherrymason/c3-lsp/lsp/indexables"

type IndexStore struct {
	symbols []idx.Indexable
}

func NewIndexStore() IndexStore {
	return IndexStore{
		symbols: make([]idx.Indexable, 0),
	}
}

func (i *IndexStore) RegisterSymbol(symbol idx.Indexable) {
	i.symbols = append(i.symbols, symbol)
}
