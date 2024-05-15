package parser

import idx "github.com/pherrymason/c3-lsp/lsp/symbols"

type ParsedModulesInterface interface {
	FindModuleInCursorPosition(cursorPosition idx.Position) string
}

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

func (ps ParsedModules) GetLoadableModules(modulePath idx.ModulePath) []*idx.Function {
	var mods []*idx.Function
	for _, scope := range ps.SymbolsByModule() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			mods = append(mods, scope)
		}
	}

	return mods
}

func (ps ParsedModules) HasImplicitLoadableModules(modulePath idx.ModulePath) bool {
	for _, scope := range ps.SymbolsByModule() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			return true
		}
	}

	return false
}

func (ps ParsedModules) SymbolsByModule() map[string]*idx.Function {
	return ps.fnByModules
}

func (ps ParsedModules) FindModuleInCursorPosition(position idx.Position) string {
	closerPreviousRange := idx.NewRange(0, 0, 0, 0)
	priorModule := ""
	for moduleName, module := range ps.fnByModules {
		if module.GetDocumentRange().HasPosition(position) {
			return module.GetModule().GetName()
		}

		if module.GetDocumentRange().IsBeforePosition(position) {
			if closerPreviousRange.IsAfter(module.GetDocumentRange()) {
				closerPreviousRange = module.GetDocumentRange()
				priorModule = moduleName
			}
		}
	}

	if priorModule != "" {
		return ps.fnByModules[priorModule].GetModule().GetName()
	}

	return ""
}
