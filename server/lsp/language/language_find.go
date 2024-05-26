package language

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
)

func (l *Language) findModuleInPosition(docId string, position symbols.Position) string {
	for id, modulesByDoc := range l.parsedModulesByDocument {
		if id == docId {
			continue
		}

		for _, scope := range modulesByDoc.Modules() {
			if scope.GetDocumentRange().HasPosition(position) {
				return scope.GetModule().GetName()
			}
		}
	}

	panic("Module not found in position")
}

func (l *Language) implicitImportedParsedModules(acceptedModulePaths []symbols.ModulePath, excludeDocId option.Option[string]) []*symbols.Module {
	//var collectionParsedModules []*parser.ParsedModules
	var collectionModules []*symbols.Module
	for docId, parsedModules := range l.parsedModulesByDocument {
		if excludeDocId.IsSome() && excludeDocId.Get() == docId {
			continue
		}

		for _, scope := range parsedModules.Modules() {
			for _, acceptedModule := range acceptedModulePaths {
				if scope.GetModule().IsImplicitlyImported(acceptedModule) {
					collectionModules = append(collectionModules, scope)
					break
				}
			}
		}
	}

	return collectionModules
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams search_params.SearchParams, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())
	if IsLanguageKeyword(searchParams.Symbol()) {
		l.debug("Ignore because C3 keyword", debugger)
		return NewSearchResultEmpty(searchParams.TrackTraversedModules())
	}

	l.debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s: %s: %s", searchParams.DocId(), searchParams.ContextModule(), searchParams.Symbol()), debugger)

	// Check if there's parent contextual information in searchParams
	if searchParams.HasAccessPath() {
		// Going from here, search should not limit to root search
		symbolResult := l.findInParentSymbols(searchParams, debugger)
		if symbolResult.IsSome() {
			return symbolResult
		}
	}

	/* NO LONGER NEEDED?? Was this an optimization?
	if searchParams.HasModuleSpecified() {
		symbol := l._findSymbolDeclarationInModule(searchParams, debugger.goIn())
		if symbol != nil {
			return symbol
		}

		return nil

	}*/

	// Important, when depth ==0 we really need to transmit to only search into root from here, even if we go multiple levels deep.

	docIdOption := searchParams.DocId()
	var collectionParsedModules []parser.ParsedModules
	if docIdOption.IsSome() {
		docId := docIdOption.Get()
		parsedModules, found := l.parsedModulesByDocument[docId]
		if !found {
			return searchResult
		}

		collectionParsedModules = append(collectionParsedModules, parsedModules)
	} else {
		// Doc id not specified, search by module. Collect scope belonging to same module as searchParams.module
		for docId, parsedModules := range l.parsedModulesByDocument {
			if searchParams.ShouldExcludeDocId(docId) {
				continue
			}

			for _, scope := range parsedModules.Modules() {
				if scope.GetModule().IsImplicitlyImported(searchParams.ContextModulePath()) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
					break
				}
			}
		}
	}

	trackedModules := searchParams.TrackTraversedModules()
	var imports []string
	importsAdded := make(map[string]bool)
	for _, parsedModules := range collectionParsedModules {
		for _, scopedTree := range parsedModules.GetLoadableModules(searchParams.ContextModulePath()) {
			l.debug(fmt.Sprintf("Checking module \"%s\"", scopedTree.GetModuleString()), debugger)
			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scopedTree,
				0,
				searchParams.IsLimitSearchInScope(),
				searchParams.ScopeMode(),
			)

			if identifier != nil {
				searchResult.Set(identifier)
				return searchResult
			}

			// Not found, store imports traversed to avoid checking them again
			for _, imp := range scopedTree.Imports {
				if !importsAdded[imp] {
					imports = append(imports, imp)
				}
			}
		}
	}

	if searchParams.ContinueOnModules() {
		sb := search_params.NewSearchParamsBuilder().
			//WithSymbol(searchParams.Symbol()).
			WithSymbolWord(searchParams.SymbolW()).
			WithContextModule(searchParams.ContextModulePath()).
			WithExcludedDocs(searchParams.DocId()).
			WithScopeMode(search_params.InModuleRoot) // Document this
		searchInSameModule := sb.Build()

		found := l.findClosestSymbolDeclaration(searchInSameModule, debugger.goIn())
		if found.IsSome() {
			return found
		}
	}

	// SEARCH IN IMPORTED MODULES
	if docIdOption.IsSome() {
		for _, parsedModules := range collectionParsedModules {
			for _, mod := range parsedModules.Modules() {
				for i := 0; i < len(mod.Imports); i++ {
					searchResult.TrackTraversedModule(mod.Imports[i])
					if !searchParams.TrackTraversedModule(mod.Imports[i]) {
						continue
					}

					module := mod.Imports[i]
					sp := search_params.NewSearchParamsBuilder().
						//WithSymbol(searchParams.Symbol()).
						WithSymbolWord(searchParams.SymbolW()).
						WithContextModule(symbols.NewModulePathFromString(module)).
						WithTrackedModules(trackedModules).
						WithScopeMode(search_params.InModuleRoot). // Document this
						Build()

					l.debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module \"%s\": %s", module, searchParams.Symbol()), debugger)
					symbol := l.findSymbolDeclarationInModule(sp, debugger.goIn())
					if symbol.IsSome() {
						return symbol
					}
				}
			}
		}
	}

	// Last resort, check if any loadable module is compatible with the string being searched
	if debugger.depth == 0 && searchResult.IsNone() {
		moduleMatches := l.findModuleNameInTraversedModules(searchParams, searchResult.traversedModules)

		if len(moduleMatches) > 0 {
			searchResult.Set(moduleMatches[0])
		}
	}

	// Not found...
	return searchResult
}

// Search symbols inside a given module
func (l *Language) findSymbolDeclarationInModule(searchParams search_params.SearchParams, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())

	for docId, modulesByDoc := range l.parsedModulesByDocument {
		for _, scope := range modulesByDoc.GetLoadableModules(searchParams.ContextModulePath()) {
			searchResult.TrackTraversedModule(scope.GetModuleString())

			//if scope.GetModuleString() != expectedModule { // TODO Ignore current doc we are comming from
			//	continue
			//}

			if !searchParams.TrackTraversedModule(scope.GetModuleString()) {
				continue
			}
			l.debug(fmt.Sprintf("findSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)

			sp := search_params.NewSearchParamsBuilder().
				WithSymbolWord(searchParams.SymbolW()).
				WithDocId(docId).
				WithScopeMode(search_params.InModuleRoot).
				WithTrackedModules(searchParams.TrackedModules()).
				Build()

			symbolResult := l.findClosestSymbolDeclaration(
				sp, FindDebugger{depth: debugger.depth + 1})
			l.debug(fmt.Sprintf("end searching symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)
			if symbolResult.IsSome() {
				return symbolResult
			}
		}
	}

	return searchResult
}

func (l Language) findModuleNameInTraversedModules(searchParams search_params.SearchParams, traversedModules map[string]bool) []*symbols.Module {
	matches := []*symbols.Module{}

	// full module name
	if searchParams.HasAccessPath() {
		return matches
	}

	moduleName := searchParams.GetFullQualifiedName()

	for _, parsedModulesByDoc := range l.parsedModulesByDocument {
		for _, module := range parsedModulesByDoc.Modules() {

			if module.GetName() == moduleName {
				_, ok := traversedModules[moduleName]
				if ok {
					matches = append(matches, module)
				}
			}
		}
	}

	return matches
}

func findDeepFirst(identifier string, position symbols.Position, node symbols.Indexable, depth uint, limitSearchInScope bool, scopeMode search_params.ScopeMode) (symbols.Indexable, uint) {
	// Iterate first children with more children
	// when in InModuleRoot mode, ignore content of functions
	if scopeMode != search_params.InModuleRoot {
		for _, child := range node.NestedScopes() {
			// Check the fn itself! Maybe we are searching for it!
			if child.GetName() == identifier {
				return child, depth
			}

			if limitSearchInScope &&
				!child.GetDocumentRange().HasPosition(position) {
				continue
			}

			if result, resultDepth := findDeepFirst(identifier, position, child, depth+1, limitSearchInScope, scopeMode); result != nil {
				return result, resultDepth
			}
		}
	}

	if depth == 0 || (scopeMode == search_params.InScope) {
		for _, child := range node.ChildrenWithoutScopes() {
			if result, resultDepth := findDeepFirst(identifier, position, child, depth+1, limitSearchInScope, scopeMode); result != nil {
				return result, resultDepth
			}
		}
	}

	// All elements found in nestable symbols checked, check node itself
	if node.GetName() == identifier {
		_, ok := node.(*symbols.Module) // Modules will be searched later explicitly.
		if !ok {
			return node, depth
		}
	}

	return nil, depth
}
