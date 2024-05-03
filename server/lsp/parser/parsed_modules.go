package parser

import idx "github.com/pherrymason/c3-lsp/lsp/indexables"

type ParsedModules struct {
	docId       string
	fnByModules map[string]*idx.Function
}

func NewParsedModules(docId string) ParsedModules {
	return ParsedModules{
		docId:       docId,
		fnByModules: make(map[string]*idx.Function),
	}
}

func (ps ParsedModules) DocId() string {
	return ps.docId
}

func (ps *ParsedModules) RegisterModule(symbol *idx.Function) {
	ps.fnByModules[symbol.GetModule().GetName()] = symbol
}

func (ps ParsedModules) Get(moduleName string) *idx.Function {
	return ps.fnByModules[moduleName]
}

func (ps ParsedModules) SymbolsByModule() map[string]*idx.Function {
	return ps.fnByModules
}
