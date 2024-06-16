package language

import (
	"fmt"
	"sort"

	"github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
	"github.com/pherrymason/c3-lsp/option"
)

func (l *Language) findModuleInPosition(docId string, position symbols.Position) string {
	for id, modulesByDoc := range l.symbolsTable.All() {
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
	var collectionModules []*symbols.Module
	for docId, parsedModules := range l.symbolsTable.All() {
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

	l.debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s: %s: %s", searchParams.DocId(), searchParams.ModuleInCursor(), searchParams.Symbol()), debugger)

	// Check if there's parent contextual information in searchParams
	if searchParams.HasAccessPath() {
		// Going from here, search should not limit to root search
		symbolResult := l.findInParentSymbols(searchParams, debugger)
		//if symbolResult.IsSome() {
		return symbolResult
		//}
	}

	if searchParams.LimitsSearchInModule() {
		/*searchInSpecificModule := search_params.NewSearchParamsBuilder().
		WithSymbolWord(searchParams.SymbolW()).
		LimitedToModule(searchParams.LimitToModule()).
		WithoutDoc().
		Build()
		*/
		//searchResult := l.findSymbolDeclarationInModule(searchParams, searchParams.LimitToModule().Get(), debugger.goIn())
		searchResult := l.strictFindSymbolDeclarationInModule(searchParams, searchParams.LimitToModule().Get(), debugger.goIn())
		if searchResult.IsSome() {
			return searchResult
		}

		return searchResult
	}

	// Important, when depth ==0 we really need to transmit to only search into root from here, even if we go multiple levels deep.

	//docIdOption := searchParams.DocId()
	var collectionParsedModules []symbols_table.UnitModules
	if searchParams.LimitSearchToDoc() {
		docIdOption := searchParams.DocId()
		docId := docIdOption.Get()
		parsedModules := l.symbolsTable.GetByDoc(docId)
		if parsedModules == nil {
			return searchResult
		}

		collectionParsedModules = append(collectionParsedModules, *parsedModules)

		for oDocId, parsedModules := range l.symbolsTable.All() {
			if docId == oDocId {
				continue
			}

			for _, scope := range parsedModules.Modules() {
				if scope.GetModule().GetName() == searchParams.ModulePathInCursor().GetName() {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
				} else if scope.GetModule().IsImplicitlyImported(searchParams.ModulePathInCursor()) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
				}
			}
		}
	} else {
		searchInModule := searchParams.ModulePathInCursor()
		if searchParams.LimitsSearchInModule() {
			searchInModule = searchParams.LimitToModule().Get()
		}

		// Doc id not specified, search by module. Collect scope belonging to same module as searchParams.module
		for docId, parsedModules := range l.symbolsTable.All() {
			if searchParams.ShouldExcludeDocId(docId) {
				continue
			}

			for _, scope := range parsedModules.Modules() {
				if scope.GetModule().IsImplicitlyImported(searchInModule) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
					break
				}
			}
		}
	}

	trackedModules := searchParams.TrackTraversedModules()
	var imports []string
	importsAdded := make(map[string]bool)
	contextModulePath := searchParams.ModulePathInCursor()
	for _, parsedModules := range collectionParsedModules {
		for _, scopedTree := range parsedModules.GetLoadableModules(contextModulePath) {
			scopeMode := searchParams.ScopeMode()
			moduleName := scopedTree.GetModuleString()
			if scopeMode == search_params.InScope {
				docOpt := searchParams.DocId()
				if docOpt.IsSome() {
					if docOpt.Get() != scopedTree.GetDocumentURI() {
						scopeMode = search_params.InModuleRoot
					}
				}
			}
			l.debug(
				fmt.Sprintf("Checking module \"%s\": mode %d, symbol: %s",
					moduleName,
					scopeMode,
					searchParams.Symbol(),
				),
				debugger,
			)

			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scopedTree,
				0,
				searchParams.IsLimitSearchInScope(),
				scopeMode,
			)

			if identifier != nil {
				searchResult.Set(identifier)
				return searchResult
			}

			// Not found, store imports traversed to avoid checking them again
			for _, imp := range scopedTree.Imports {
				if !importsAdded[imp] {
					importsAdded[imp] = true
					imports = append(imports, imp)
				}
			}
		}
	}
	/*
		if searchParams.ContinueOnModules() {
			// SEARCH ON SAME ContextModulePath, but other files
			// This search should not go through imported modules in those other files
			sb := search_params.NewSearchParamsBuilder().
				WithText(searchParams.SymbolW().Text(), searchParams.SymbolW().TextRange()).
				WithContextModule(searchParams.ContextModulePath()).
				WithoutDoc().
				WithExcludedDocs(searchParams.DocId()).
				WithoutContinueOnModules().
				WithScopeMode(search_params.InModuleRoot) // Document this
			searchInSameModule := sb.Build()

			found := l.findClosestSymbolDeclaration(searchInSameModule, debugger.goIn())
			if found.IsSome() {
				return found
			}
		}*/

	// SEARCH IN IMPORTED MODULES
	if searchParams.LimitSearchToDoc() {
		for _, parsedModules := range collectionParsedModules {
			for _, mod := range parsedModules.Modules() {
				for i := 0; i < len(mod.Imports); i++ {
					searchResult.TrackTraversedModule(mod.Imports[i])
					if !searchParams.TrackTraversedModule(mod.Imports[i]) {
						continue
					}

					module := mod.Imports[i]
					sp := search_params.NewSearchParamsBuilder().
						WithSymbolWord(searchParams.SymbolW()).
						LimitedToModulePath(symbols.NewModulePathFromString(module)).
						WithTrackedModules(trackedModules).
						WithScopeMode(search_params.InModuleRoot). // Document this
						Build()

					l.debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module \"%s\": %s", module, searchParams.Symbol()), debugger)
					symbol := l.findSymbolDeclarationInModule(sp, symbols.NewModulePathFromString(module), debugger.goIn())
					if symbol.IsSome() {
						return symbol
					}
				}
			}
		}
	}

	// Last resort, check if any loadable module is compatible with the string being searched
	if /*debugger.depth == 0 &&*/ searchResult.IsNone() {
		moduleMatches := l.findModuleNameInTraversedModules(searchParams, searchResult.traversedModules)

		if len(moduleMatches) > 0 {
			searchResult.Set(moduleMatches[0])
		}
	}

	// Not found...
	return searchResult
}

// Search symbols inside a given module
func (l *Language) findSymbolDeclarationInModule(searchParams search_params.SearchParams, moduleToSearch symbols.ModulePath, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())

	for docId, modulesByDoc := range l.symbolsTable.All() {
		//for _, scope := range modulesByDoc.GetLoadableModules(searchParams.ModulePathInCursor()) {
		for _, scope := range modulesByDoc.GetLoadableModules(moduleToSearch) {
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

func (l *Language) strictFindSymbolDeclarationInModule(searchParams search_params.SearchParams, moduleToSearch symbols.ModulePath, debugger FindDebugger) SearchResult {
	searchResult := NewSearchResult(searchParams.TrackTraversedModules())
	results := []struct {
		identifier symbols.Indexable
		score      int
	}{}

	for docId, modulesByDoc := range l.symbolsTable.All() {
		for _, scope := range modulesByDoc.GetLoadableModules(moduleToSearch) {
			searchResult.TrackTraversedModule(scope.GetModuleString())
			if !searchParams.TrackTraversedModule(scope.GetModuleString()) {
				continue
			}
			l.debug(fmt.Sprintf("strictFindSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)

			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scope,
				0,
				searchParams.IsLimitSearchInScope(),
				search_params.InModuleRoot,
			)

			if identifier != nil {
				score := 2
				if moduleToSearch.GetName() == scope.GetModule().GetName() {
					score = 100
				}

				results = append(results, struct {
					identifier symbols.Indexable
					score      int
				}{identifier: identifier, score: score})
				//searchResult.Set(identifier)
				//return searchResult
			}
		}
	}

	if len(results) == 0 {
		if searchResult.IsNone() {
			moduleMatches := l.findModuleNameInTraversedModules(searchParams, searchResult.traversedModules)

			if len(moduleMatches) > 0 {
				searchResult.Set(moduleMatches[0])
			}
		}

		return searchResult
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	searchResult.Set(results[0].identifier)
	return searchResult
}

func (l Language) findModuleNameInTraversedModules(searchParams search_params.SearchParams, traversedModules map[string]bool) []*symbols.Module {
	matches := []*symbols.Module{}

	// full module name
	if searchParams.HasAccessPath() {
		return matches
	}

	moduleName := searchParams.GetFullQualifiedName()

	for _, parsedModulesByDoc := range l.symbolsTable.All() {
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

// There are two modes:
// InScope: Search first symbols defined in same scope as `position`. If not found, will search on root of module.
// InModuleRoot: Will only search symbols defined in the root. Will not go inside functions.
func findDeepFirst(identifier string, position symbols.Position, node symbols.Indexable, depth uint, limitSearchInScope bool, scopeMode search_params.ScopeMode) (symbols.Indexable, uint) {
	// Iterate first children with more children
	// when in InModuleRoot mode, ignore content of functions
	if scopeMode == search_params.InScope {
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

	if depth == 0 && scopeMode == search_params.InModuleRoot {
		for _, child := range node.Children() {
			if child.GetName() == identifier {
				return child, depth
			}
		}
		for _, child := range node.NestedScopes() {
			if child.GetName() == identifier {
				return child, depth
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
