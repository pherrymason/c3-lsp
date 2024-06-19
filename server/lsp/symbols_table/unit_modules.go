package symbols_table

import (
	"github.com/pherrymason/c3-lsp/data"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	sitter "github.com/smacker/go-tree-sitter"
)

type UnitModules struct {
	docId   *string
	modules data.OrderedMap[*idx.Module]
}

func NewParsedModules(docId *string) UnitModules {
	return UnitModules{
		docId:   docId,
		modules: *data.NewOrderedMap[*idx.Module](),
	}
}

func (ps UnitModules) DocId() string {
	return *ps.docId
}

func (ps *UnitModules) ModuleIds() []string {
	return ps.modules.Keys()
}

func (ps *UnitModules) GetOrInitModule(moduleName string, docId *string, rootNode *sitter.Node, anonymousModuleName bool) *idx.Module {
	if anonymousModuleName {
		// Build module name from filename
		moduleName = idx.NormalizeModuleName(*docId)
	}

	module, exists := ps.modules.Get(moduleName)
	if !exists {
		module = idx.NewModule(
			moduleName,
			docId,
			idx.NewRangeFromTreeSitterPositions(
				rootNode.StartPoint(),
				rootNode.EndPoint(),
			),
			idx.NewRangeFromTreeSitterPositions(
				rootNode.StartPoint(),
				rootNode.EndPoint(),
			),
		)

		ps.modules.Set(moduleName, module)
	}

	return module
}

func (ps *UnitModules) UpdateOrInitModule(module *idx.Module, rootNode *sitter.Node) *idx.Module {
	moduleName := module.GetName()

	existingModule, exists := ps.modules.Get(moduleName)
	if !exists {
		ps.modules.Set(module.GetName(), module)
	} else {
		// Update stored module
		existingModule.SetAttributes(module.GetAttributes())
		existingModule.SetGenericParameters(module.GenericParameters)
	}

	return module
}

func (ps *UnitModules) RegisterModule(symbol *idx.Module) {
	ps.modules.Set(symbol.GetModule().GetName(), symbol)
}

func (ps UnitModules) Get(moduleName string) *idx.Module {
	mod, ok := ps.modules.Get(moduleName)
	if ok {
		return mod
	}

	return nil
}

func (ps UnitModules) GetLoadableModules(modulePath idx.ModulePath) []*idx.Module {
	var mods []*idx.Module
	for _, scope := range ps.Modules() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			mods = append(mods, scope)
		}
	}

	return mods
}

func (ps UnitModules) HasImplicitLoadableModules(modulePath idx.ModulePath) bool {
	for _, scope := range ps.Modules() {
		if scope.GetModule().IsImplicitlyImported(modulePath) {
			return true
		}
	}

	return false
}

func (ps UnitModules) HasExplicitlyImportedModules(modulePath idx.ModulePath) bool {
	for _, scope := range ps.Modules() {
		if modulePath.IsSubModuleOf(scope.GetModule()) {
			return true
		}

		/*
			if scope.GetModule().IsImplicitlyImported(modulePath) {
				return true
			}
		*/
	}

	return false
}

// Returns modules sorted by value
func (ps UnitModules) Modules() []*idx.Module {
	return ps.modules.Values()
}

func (ps UnitModules) FindContextModuleInCursorPosition(position idx.Position) string {
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
