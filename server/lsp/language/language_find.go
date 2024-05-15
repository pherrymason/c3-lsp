package language

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
)

func (l *Language) findModuleInPosition(docId string, position symbols.Position) string {
	for id, modulesByDoc := range l.functionTreeByDocument {
		if id == docId {
			continue
		}

		for _, scope := range modulesByDoc.SymbolsByModule() {
			if scope.GetDocumentRange().HasPosition(position) {
				return scope.GetModule().GetName()
			}
		}
	}

	panic("Module not found in position")
}

// Returns all symbols in scope.
// Detail: StructMembers and Enumerables are inlined
func (l *Language) findAllScopeSymbols(parsedModules *parser.ParsedModules, position symbols.Position) []symbols.Indexable {
	var symbolsCollection []symbols.Indexable
	for _, scopeFunction := range parsedModules.SymbolsByModule() {
		// Allow also when position is +1 line, in case we are writing new contentn at the end of the file
		has := scopeFunction.GetDocumentRange().HasPosition(symbols.NewPosition(position.Line-1, 0))
		if !scopeFunction.GetDocumentRange().HasPosition(position) && !has {
			continue // We are in a different module
		}

		for _, variable := range scopeFunction.Variables {
			symbolsCollection = append(symbolsCollection, variable)
		}
		for _, enum := range scopeFunction.Enums {
			symbolsCollection = append(symbolsCollection, enum)
			for _, enumerable := range enum.GetEnumerators() {
				symbolsCollection = append(symbolsCollection, enumerable)
			}
		}
		for _, strukt := range scopeFunction.Structs {
			symbolsCollection = append(symbolsCollection, strukt)
		}
		for _, def := range scopeFunction.Defs {
			symbolsCollection = append(symbolsCollection, def)
		}
		for _, faults := range scopeFunction.Faults {
			symbolsCollection = append(symbolsCollection, faults)
		}
		for _, interfaces := range scopeFunction.Interfaces {
			symbolsCollection = append(symbolsCollection, interfaces)
		}

		for _, function := range scopeFunction.ChildrenFunctions {
			symbolsCollection = append(symbolsCollection, function)
			if function.GetDocumentRange().HasPosition(position) {
				symbolsCollection = append(symbolsCollection, function)

				for _, variable := range function.Variables {
					l.logger.Debug(fmt.Sprintf("Checking %s variable:", variable.GetName()))
					declarationPosition := variable.GetIdRange().End
					if declarationPosition.Line > uint(position.Line) ||
						(declarationPosition.Line == uint(position.Line) && declarationPosition.Character > uint(position.Character)) {
						continue
					}

					symbolsCollection = append(symbolsCollection, variable)
				}
			}
		}
	}

	return symbolsCollection
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams search_params.SearchParams, debugger FindDebugger) option.Option[symbols.Indexable] {
	if IsLanguageKeyword(searchParams.Symbol()) {
		l.debug("Ignore because C3 keyword", debugger)
		return option.None[symbols.Indexable]()
	}

	l.debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s: %s: %s", searchParams.DocId(), searchParams.Module(), searchParams.Symbol()), debugger)

	/*position := searchParams.symbolRange.Start*/
	// Check if there's parent contextual information in searchParams
	if searchParams.HasAccessPath() {
		identifier := l.findInParentSymbols(searchParams, debugger)
		if identifier.IsSome() {
			return identifier
		}
	}

	if searchParams.HasModuleSpecified() {
		/*symbol := l._findSymbolDeclarationInModule(searchParams, debugger.goIn())
		if symbol != nil {
			return symbol
		}

		return nil
		*/
	}

	docIdOption := searchParams.DocId()
	var collectionParsedModules []parser.ParsedModules
	if docIdOption.IsSome() {
		docId := docIdOption.Get()
		parsedModules, found := l.functionTreeByDocument[docId]
		if !found {
			return option.None[symbols.Indexable]()
		}

		collectionParsedModules = append(collectionParsedModules, parsedModules)
	} else {
		// Doc id not specified, search by module. Collect scope belonging to same module as searchParams.module
		for docId, parsedModules := range l.functionTreeByDocument {
			if searchParams.ShouldExcludeDocId(docId) {
				continue
			}

			for _, scope := range parsedModules.SymbolsByModule() {
				if scope.GetModule().IsImplicitlyImported(searchParams.ModulePath()) {
					collectionParsedModules = append(collectionParsedModules, parsedModules)
					break
				}
				/*
					scopeModuleName := scope.GetModuleString()
					var isSameModule, isSubModule, isParentModule bool
					isSameModule = scopeModuleName == searchParams.Module()
					if !searchParams.ModulePath().IsEmpty() {
						isSubModule = scope.IsSubModuleOf(searchParams.ModulePath())
						isParentModule = searchParams.ModulePath().IsSubModuleOf(scope.GetModule())
					}

					if !isSameModule && !isSubModule && !isParentModule {
						continue
					}

					collectionParsedModules = append(collectionParsedModules, parsedModules)
					break*/
			}
		}

		//return option.None[symbols.Indexable]()
	}

	trackedModules := searchParams.TrackTraversedModules()
	var imports []string
	importsAdded := make(map[string]bool)
	for _, parsedModules := range collectionParsedModules {
		for _, scopedTree := range parsedModules.GetLoadableModules(searchParams.ModulePath()) {
			l.debug(fmt.Sprintf("Checking module \"%s\"", scopedTree.GetModuleString()), debugger)
			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(
				searchParams.Symbol(),
				searchParams.SymbolPosition(),
				scopedTree,
				0,
				searchParams.IsLimitSearchInScope(),
			)

			if identifier != nil {
				return option.Some(identifier)
			}

			// Not found, store its imports
			for _, imp := range scopedTree.Imports {
				if !importsAdded[imp] {
					imports = append(imports, imp)
				}
			}
		}
	}

	if searchParams.ContinueOnModules() {
		sb := search_params.NewSearchParamsBuilder().
			WithSymbol(searchParams.Symbol()).
			WithSymbolModule(searchParams.ModulePath()).
			WithExcludedDocs(searchParams.DocId())
		searchInSameModule := sb.Build()

		found := l.findClosestSymbolDeclaration(searchInSameModule, debugger.goIn())
		if found.IsSome() {
			return found
		}
	}

	// Try to find element in one of the imported modules
	if docIdOption.IsSome() && len(imports) > 0 {
		for i := 0; i < len(imports); i++ {
			if !searchParams.TrackTraversedModule(imports[i]) {
				continue
			}

			module := imports[i]
			sp := search_params.NewSearchParamsBuilder().
				WithSymbol(searchParams.Symbol()).
				WithSymbolModule(symbols.NewModulePathFromString(module)).
				WithTrackedModules(trackedModules).
				Build()

			l.debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module \"%s\": %s", module, searchParams.Symbol()), debugger)
			symbol := l.findSymbolDeclarationInModule(sp, debugger.goIn())
			if symbol.IsSome() {
				return symbol
			}
		}
	}

	// Not found...
	return option.None[symbols.Indexable]()
}

// Search symbols inside a given module
func (l *Language) findSymbolDeclarationInModule(searchParams search_params.SearchParams, debugger FindDebugger) option.Option[symbols.Indexable] {
	//expectedModule := searchParams.ModulePath().GetName()

	for docId, modulesByDoc := range l.functionTreeByDocument {
		for _, scope := range modulesByDoc.GetLoadableModules(searchParams.ModulePath()) {
			//if scope.GetModuleString() != expectedModule { // TODO Ignore current doc we are comming from
			//	continue
			//}

			if !searchParams.TrackTraversedModule(scope.GetModuleString()) {
				continue
			}
			l.debug(fmt.Sprintf("findSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)

			sp := search_params.NewSearchParamsBuilder().
				WithSymbol(searchParams.Symbol()).
				WithDocId(docId).
				WithTrackedModules(searchParams.TrackedModules()).
				Build()

			symbol := l.findClosestSymbolDeclaration(
				sp,
				/*SearchParams{
					selectedToken:     searchParams.selectedToken,
					docId:             docId,
					scopeMode:         searchParams.scopeMode,
					continueOnModules: true,
					trackedModules:    searchParams.trackedModules,
				}*/FindDebugger{depth: debugger.depth + 1})
			l.debug(fmt.Sprintf("end searching symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)
			if symbol.IsSome() {
				return symbol
			}
		}
	}

	return option.None[symbols.Indexable]()
}

func findDeepFirst(identifier string, position symbols.Position, function *symbols.Function, depth uint, limitSearchInScope bool) (symbols.Indexable, uint) {
	if function.FunctionType() == symbols.UserDefined && identifier == function.GetFullName() {
		return function, depth
	}

	// When mode is InPosition, we are specifying it is important for us that
	// the function being searched contains the position specified.
	// We use mode = AnyPosition when search for a symbol definition outside the document where the user has its cursor. For example, we are looking in imported files or files of the same module.
	if limitSearchInScope &&
		!function.GetDocumentRange().HasPosition(position) {
		return nil, depth
	}

	// Try this
	// First, loop through symbols that can nest other definitions:
	// - structs, unions [v]
	// - enum, faults	 [v]
	// - interface
	// - functions       [v]
	// For each of them, check if search.symbol.position is inside that element
	// If true, priorize searching there first.
	// If not found, search in the `function`
	for _, structs := range function.Structs {
		if structs.GetDocumentRange().HasPosition(position) {
			// What we are looking is inside this, look for struct member
			for _, member := range structs.GetMembers() {
				if member.GetName() == identifier {
					return member, depth
				}
			}
		}
	}

	for _, scopedEnums := range function.Enums {
		if scopedEnums.GetDocumentRange().HasPosition(position) {
			if scopedEnums.HasEnumerator(identifier) {
				enumerator := scopedEnums.GetEnumerator(identifier)
				return enumerator, depth
			}
		}
	}

	for _, child := range function.ChildrenFunctions {
		if result, resultDepth := findDeepFirst(identifier, position, &child, depth+1, limitSearchInScope); result != nil {
			return result, resultDepth
		}
	}

	// All elements found in nestable symbols checked, check rest of function

	variable, foundVariableInThisScope := function.Variables[identifier]
	if foundVariableInThisScope {
		return variable, depth
	}

	enum, foundEnumInThisScope := function.Enums[identifier]
	if foundEnumInThisScope {
		return enum, depth
	}

	// Apparently, removing this makes enumerator tests in language fail.
	var enumerator symbols.Enumerator
	foundEnumeratorInThisScope := false
	for _, scopedEnums := range function.Enums {
		if scopedEnums.HasEnumerator(identifier) {
			enumerator = scopedEnums.GetEnumerator(identifier)
			foundEnumeratorInThisScope = true
		}
	}
	if foundEnumeratorInThisScope {
		return enumerator, depth
	}

	_struct, foundStructInThisScope := function.Structs[identifier]
	if foundStructInThisScope {
		return _struct, depth
	}

	def, foundDefInScope := function.Defs[identifier]
	if foundDefInScope {
		return def, depth
	}

	fault, foundFaultInScope := function.Faults[identifier]
	if foundFaultInScope {
		return fault, depth
	}
	foundEnumeratorInThisScope = false
	var faultConstant symbols.FaultConstant
	for _, scopedEnum := range function.Faults {
		if scopedEnum.HasConstant(identifier) {
			faultConstant = scopedEnum.GetConstant(identifier)
			foundEnumeratorInThisScope = true
		}
	}
	if foundEnumeratorInThisScope {
		return faultConstant, depth
	}

	_interface, foundInInterface := function.Interfaces[identifier]
	if foundInInterface {
		return _interface, depth
	}

	return nil, depth
}

func inSlice(element string, slice []string) bool {
	for _, value := range slice {
		if value == element {
			return true
		}
	}
	return false
}
