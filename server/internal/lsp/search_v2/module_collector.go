package search_v2

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

// SearchContext holds the context for symbol search
type SearchContext struct {
	CurrentDoc    string
	CurrentModule symbols.ModulePath
	Position      symbols.Position
}

// ModuleScope represents a module to search with priority
type ModuleScope struct {
	Module   *symbols.Module
	DocId    string
	Priority int
}

const (
	HighPriority   = 100
	MediumPriority = 50
	LowPriority    = 10
)

// ModuleCollector determines which modules to search in priority order
type ModuleCollector struct {
	state *project_state.ProjectState
}

func NewModuleCollector(state *project_state.ProjectState) *ModuleCollector {
	return &ModuleCollector{state: state}
}

// CollectRelevantModules returns modules ordered by search priority
func (c *ModuleCollector) CollectRelevantModules(ctx SearchContext) []ModuleScope {
	result := []ModuleScope{}
	seen := make(map[string]bool) // Track to avoid duplicates

	// 1. Current document's modules (highest priority)
	currentDocModules := c.getModulesFromDoc(ctx.CurrentDoc)
	for _, mod := range currentDocModules {
		key := mod.DocId + "::" + mod.Module.GetModuleString()
		if !seen[key] {
			seen[key] = true
			result = append(result, mod)
		}
	}

	// 2. Same module in other files (high priority)
	sameModuleOtherDocs := c.getSameModuleInOtherDocs(ctx.CurrentModule, ctx.CurrentDoc)
	for _, mod := range sameModuleOtherDocs {
		key := mod.DocId + "::" + mod.Module.GetModuleString()
		if !seen[key] {
			seen[key] = true
			result = append(result, mod)
		}
	}

	// 3. Imported modules (medium priority)
	imports := c.getImportsForModule(ctx.CurrentModule, ctx.CurrentDoc)
	for _, importPath := range imports {
		importedModules := c.getModulesFromImport(importPath)
		for _, mod := range importedModules {
			key := mod.DocId + "::" + mod.Module.GetModuleString()
			if !seen[key] {
				seen[key] = true
				result = append(result, mod)
			}
		}
	}

	return result
}

// getModulesFromDoc gets all modules from a specific document
func (c *ModuleCollector) getModulesFromDoc(docId string) []ModuleScope {
	unitModules := c.state.GetUnitModulesByDoc(docId)
	if unitModules == nil {
		return nil
	}

	scopes := []ModuleScope{}
	for _, modId := range unitModules.ModuleIds() {
		mod := unitModules.Get(modId)
		scopes = append(scopes, ModuleScope{
			Module:   mod,
			DocId:    docId,
			Priority: HighPriority,
		})
	}
	return scopes
}

// getSameModuleInOtherDocs finds the same module in other documents
func (c *ModuleCollector) getSameModuleInOtherDocs(modulePath symbols.ModulePath, excludeDocId string) []ModuleScope {
	scopes := []ModuleScope{}

	for docId, unitModules := range c.state.GetAllUnitModules() {
		if docId == excludeDocId {
			continue
		}

		for _, modId := range unitModules.ModuleIds() {
			mod := unitModules.Get(modId)
			if mod.GetModule().GetName() == modulePath.GetName() {
				scopes = append(scopes, ModuleScope{
					Module:   mod,
					DocId:    docId,
					Priority: HighPriority,
				})
			}
		}
	}

	return scopes
}

// getImportsForModule gets the imports for a module
func (c *ModuleCollector) getImportsForModule(modulePath symbols.ModulePath, docId string) []string {
	unitModules := c.state.GetUnitModulesByDoc(docId)
	if unitModules == nil {
		return nil
	}

	imports := []string{}
	for _, modId := range unitModules.ModuleIds() {
		mod := unitModules.Get(modId)
		if mod.GetModule().GetName() == modulePath.GetName() {
			imports = append(imports, mod.Imports...)
		}
	}

	return imports
}

// getModulesFromImport gets modules matching an import path
func (c *ModuleCollector) getModulesFromImport(importPath string) []ModuleScope {
	scopes := []ModuleScope{}
	importModulePath := symbols.NewModulePathFromString(importPath)

	for docId, unitModules := range c.state.GetAllUnitModules() {
		for _, modId := range unitModules.ModuleIds() {
			mod := unitModules.Get(modId)
			if mod.GetModule().IsImplicitlyImported(importModulePath) {
				scopes = append(scopes, ModuleScope{
					Module:   mod,
					DocId:    docId,
					Priority: MediumPriority,
				})
			}
		}
	}

	return scopes
}
