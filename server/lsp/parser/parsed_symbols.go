package parser

import idx "github.com/pherrymason/c3-lsp/lsp/indexables"

type ParsedSymbols struct {
	symbolsTable   []*idx.Indexable
	scopedFunction idx.Function
}

func NewParsedSymbols() ParsedSymbols {
	return ParsedSymbols{
		symbolsTable: make([]*idx.Indexable, 0),
	}
}

func (ps *ParsedSymbols) RegisterSymbol(symbol *idx.Indexable) {
	ps.symbolsTable = append(ps.symbolsTable, symbol)
}
