package parser

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
)

type ModulesInDocument map[string]*idx.Module

type ParsedModulesInterface interface {
	FindModuleInCursorPosition(cursorPosition idx.Position) string
}

type ParsedModules struct {
	docId   string
	modules ModulesInDocument
}

func NewParsedModules(docId string) ParsedModules {
	return ParsedModules{
		docId:   docId,
		modules: make(ModulesInDocument),
	}
}

func (ps ParsedModules) DocId() string {
	return ps.docId
}

func (ps *ParsedModules) GetOrInitModule(moduleName string, doc *document.Document, anonymousModuleName bool) *idx.Module {
	if anonymousModuleName {
		// Build module name from filename
		moduleName = idx.NormalizeModuleName(doc.URI)
	}

	module, exists := ps.modules[moduleName]
	if !exists {
		module = idx.NewModule(
			moduleName,
			doc.URI,
			idx.NewRangeFromTreeSitterPositions(
				doc.ContextSyntaxTree.RootNode().StartPoint(),
				doc.ContextSyntaxTree.RootNode().EndPoint(),
			),
			idx.NewRangeFromTreeSitterPositions(
				doc.ContextSyntaxTree.RootNode().StartPoint(),
				doc.ContextSyntaxTree.RootNode().EndPoint(),
			),
		)

		ps.modules[moduleName] = module
	}

	return module
}

func (ps *ParsedModules) RegisterModule(symbol *idx.Module) {
	ps.modules[symbol.GetModule().GetName()] = symbol
}

func (ps ParsedModules) Get(moduleName string) *idx.Module {
	return ps.modules[moduleName]
}

func (ps ParsedModules) GetLoadableModules(modulePath idx.ModulePath) []*idx.Module {
	var mods []*idx.Module
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

func (ps ParsedModules) SymbolsByModule() ModulesInDocument {
	return ps.modules
}

func (ps ParsedModules) FindModuleInCursorPosition(position idx.Position) string {
	closerPreviousRange := idx.NewRange(0, 0, 0, 0)
	priorModule := ""
	for moduleName, module := range ps.modules {
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
		return ps.modules[priorModule].GetModule().GetName()
	}

	return ""
}
