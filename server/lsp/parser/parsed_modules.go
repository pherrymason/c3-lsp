package parser

import (
	"github.com/pherrymason/c3-lsp/data"
	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
)

type ParsedModulesInterface interface {
	FindContextModuleInCursorPosition(cursorPosition idx.Position) string
}

type ParsedModules struct {
	docId   string
	modules data.OrderedMap[*idx.Module]
}

func NewParsedModules(docId string) ParsedModules {
	return ParsedModules{
		docId:   docId,
		modules: *data.NewOrderedMap[*idx.Module](),
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

	module, exists := ps.modules.Get(moduleName)
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

		ps.modules.Set(moduleName, module)
	}

	return module
}

func (ps *ParsedModules) RegisterModule(symbol *idx.Module) {
	ps.modules.Set(symbol.GetModule().GetName(), symbol)
}

func (ps ParsedModules) Get(moduleName string) *idx.Module {
	mod, ok := ps.modules.Get(moduleName)
	if ok {
		return mod
	}

	return nil
}

func (ps ParsedModules) GetLoadableModules(modulePath idx.ModulePath) []*idx.Module {
	var mods []*idx.Module
	for _, scope := range ps.Modules() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			mods = append(mods, scope)
		}
	}

	return mods
}

func (ps ParsedModules) HasImplicitLoadableModules(modulePath idx.ModulePath) bool {
	for _, scope := range ps.Modules() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			return true
		}
	}

	return false
}

// Returns modules sorted by value
func (ps ParsedModules) Modules() []*idx.Module {
	return ps.modules.Values()
}

func (ps ParsedModules) FindContextModuleInCursorPosition(position idx.Position) string {
	closerPreviousRange := idx.NewRange(0, 0, 0, 0)
	priorModule := ""
	for _, module := range ps.modules.Values() {

		if module.GetDocumentRange().HasPosition(position) {
			return module.GetModule().GetName()
		}

		if module.GetDocumentRange().IsBeforePosition(position) {
			if closerPreviousRange.IsAfter(module.GetDocumentRange()) {
				closerPreviousRange = module.GetDocumentRange()
				priorModule = module.GetName()
			}
		}
	}

	if priorModule != "" {
		mod, _ := ps.modules.Get(priorModule)
		return mod.GetModule().GetName()
	}

	return ""
}
