package language

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/parser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (l *Language) findAllScopeSymbols(parsedModules *parser.ParsedModules, position protocol.Position) []indexables.Indexable {
	var symbols []indexables.Indexable
	for _, scopeFunction := range parsedModules.SymbolsByModule() {
		if !scopeFunction.GetDocumentRange().HasPosition(position) {
			continue // We are in a different module
		}

		for _, variable := range scopeFunction.Variables {
			symbols = append(symbols, variable)
		}
		for _, enum := range scopeFunction.Enums {
			symbols = append(symbols, enum)
		}
		for _, strukt := range scopeFunction.Structs {
			symbols = append(symbols, strukt)
		}
		for _, def := range scopeFunction.Defs {
			symbols = append(symbols, def)
		}
		for _, faults := range scopeFunction.Faults {
			symbols = append(symbols, faults)
		}
		for _, interfaces := range scopeFunction.Interfaces {
			symbols = append(symbols, interfaces)
		}

		for _, function := range scopeFunction.ChildrenFunctions {
			if function.GetDocumentRange().HasPosition(position) {
				symbols = append(symbols, function)

				for _, variable := range function.Variables {
					l.logger.Debug(fmt.Sprintf("Checking %s variable:", variable.GetName()))
					declarationPosition := variable.GetIdRange().End
					if declarationPosition.Line > uint(position.Line) ||
						(declarationPosition.Line == uint(position.Line) && declarationPosition.Character > uint(position.Character)) {
						continue
					}

					symbols = append(symbols, variable)
				}
			}
		}
	}

	return symbols
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams) indexables.Indexable {
	l.logger.Debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s", searchParams.docId))
	position := searchParams.selectedSymbol.position
	// Check if there's parent contextual information in searchParams
	if searchParams.HasParentSymbol() {
		identifier := l.findInParentSymbols(searchParams)
		if identifier != nil {
			return identifier
		}
	}

	if searchParams.HasModuleSpecified() {
		symbol := l._findSymbolDeclarationInModule(searchParams)
		if symbol != nil {
			return symbol
		}
	} else {
		documentModules, found := l.functionTreeByDocument[searchParams.docId]
		if !found {
			return nil
		}

		for module, scopedTree := range documentModules.SymbolsByModule() {
			l.logger.Debug(fmt.Sprintf("Checking module %s\n", module))
			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(searchParams.selectedSymbol.token, position, scopedTree, 0, searchParams.scopeMode)

			if identifier != nil {
				return identifier
			}

			if searchParams.continueOnModules {
				found := l.findSymbolsInModuleOtherFiles(scopedTree.GetModule(), searchParams)
				if found != nil {
					return found
				}
			}

			// Try to find element in one of the imported modules
			if len(scopedTree.Imports) > 0 {
				traversedModules := searchParams.traversedModules
				for i := 0; i < len(scopedTree.Imports); i++ {
					if inSlice(scopedTree.Imports[i], traversedModules) {
						continue
					}

					// TODO: Some scenarios cause an infinite loop here
					module := scopedTree.Imports[i]
					sp := SearchParams{
						selectedSymbol:   searchParams.selectedSymbol,
						modulePath:       indexables.NewModulePath([]string{module}),
						scopeMode:        AnyPosition,
						traversedModules: traversedModules,
					}
					l.logger.Debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module %s", module))
					symbol := l._findSymbolDeclarationInModule(sp)
					if symbol != nil {
						return symbol
					}
					// TODO: This next line is an optimization to remember imported submodules in `module`
					traversedModules = append(traversedModules, module)
				}
			}
		}
	}

	// Not found yet, let's try search the selectedSymbol defined as global in other files
	// Note: Iterating a map is not guaranteed to be done always in the same order.

	// Not found...
	return nil
}

func (l *Language) findSymbolsInModuleOtherFiles(module indexables.ModulePath, searchParams SearchParams) indexables.Indexable {

	for docId, modulesByDoc := range l.functionTreeByDocument {
		if docId == searchParams.docId {
			continue
		}

		for _, scope := range modulesByDoc.SymbolsByModule() {
			isSameModule := scope.GetModuleString() == module.GetName()
			isSubModule := scope.IsSubModuleOf(module)
			isParentModule := module.IsSubModuleOf(scope.GetModule())
			if !isSameModule && !isSubModule && !isParentModule {
				continue
			}

			// Can we just call findDeepFirst() directly instead?
			found := l.findClosestSymbolDeclaration(
				SearchParams{
					selectedSymbol:    searchParams.selectedSymbol,
					docId:             scope.GetDocumentURI(),
					scopeMode:         AnyPosition,
					continueOnModules: false,
				},
			)
			if found != nil {
				return found
			}
		}
	}

	return nil
}

func (l *Language) _findSymbolDeclarationInModule(searchParams SearchParams) indexables.Indexable {
	expectedModule := searchParams.modulePath.GetName()

	for docId, modulesByDoc := range l.functionTreeByDocument {
		for _, scope := range modulesByDoc.SymbolsByModule() {
			if scope.GetModuleString() != expectedModule { // TODO Ignore current doc we are comming from
				continue
			}
			if inSlice(scope.GetModuleString(), searchParams.traversedModules) {
				continue
			}

			l.logger.Debug(fmt.Sprintf("_findSymbolDeclarationInModule: search with fdcdd inside %s file %s", scope.GetModuleString(), docId))
			symbol := l.findClosestSymbolDeclaration(SearchParams{
				selectedSymbol:    searchParams.selectedSymbol,
				docId:             docId,
				scopeMode:         searchParams.scopeMode,
				continueOnModules: true,
				traversedModules:  append(searchParams.traversedModules, scope.GetModuleString()),
			})

			if symbol != nil {
				return symbol
			}
		}
	}

	return nil
}

func (l *Language) findInParentSymbols(searchParams SearchParams) indexables.Indexable {
	var levelIdentifier indexables.Indexable

	parentSymbolsCount := len(searchParams.parentSymbols)
	rootSymbol := searchParams.parentSymbols[parentSymbolsCount-1]
	levelSearchParams := SearchParams{
		selectedSymbol:    rootSymbol,
		docId:             searchParams.docId,
		scopeMode:         InScope,
		continueOnModules: true,
	}

	currentToken := rootSymbol
	parentDepth := parentSymbolsCount - 1
	searchingFinalSymbol := false
	finalSymbolFound := false
	for {
		finalSymbolFound = false
		levelIdentifier = l.findClosestSymbolDeclaration(levelSearchParams)
		if levelIdentifier == nil {
			return nil
			//panic(fmt.Sprintf("Could not find symbol at %d level", parentDepth))
		}

		switch levelIdentifier.(type) {
		case indexables.Variable:
			variable, ok := levelIdentifier.(indexables.Variable)
			if !ok {
				panic("Could not convert levelIdentifier to idx.Variable")
			}
			levelSearchParams = NewSearchParams(
				variable.GetType().GetName(),
				currentToken.position,
				variable.GetDocumentURI(),
			)
			levelIdentifier = variable
			finalSymbolFound = true

		case indexables.Struct:
			_struct := levelIdentifier.(indexables.Struct)
			levelIdentifier = _struct
			members := _struct.GetMembers()
			for i := 0; i < len(members); i++ {
				if members[i].GetName() == currentToken.token {
					levelIdentifier = members[i]
					finalSymbolFound = true
					levelSearchParams = NewSearchParams(
						members[i].GetType(),
						currentToken.position,
						_struct.GetDocumentURI(),
					)
					break
				}
			}

			// Member not found, let's do an extra search in functions
			if !finalSymbolFound {
				levelSearchParams = NewSearchParams(
					_struct.GetName()+"."+currentToken.token,
					currentToken.position,
					levelIdentifier.GetDocumentURI(),
				)
			}

		case indexables.Enum:
			// Search searchParams.selectedSymbol in enumerators
			_enum := levelIdentifier.(indexables.Enum)
			levelIdentifier = _enum
			enumerators := _enum.GetEnumerators()
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == currentToken.token {
					levelIdentifier = enumerators[i]
					finalSymbolFound = true
					levelSearchParams = NewSearchParams(
						enumerators[i].GetName(),
						currentToken.position,
						levelIdentifier.GetDocumentURI(),
					)
				}
			}

		case indexables.Fault:
			_fault := levelIdentifier.(indexables.Fault)
			enumerators := _fault.GetConstants()
			levelIdentifier = _fault
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == currentToken.token {
					levelIdentifier = enumerators[i]
					finalSymbolFound = true
					levelSearchParams = NewSearchParams(
						enumerators[i].GetName(),
						currentToken.position,
						levelIdentifier.GetDocumentURI(),
					)
				}
			}

		case *indexables.Function:
			_fun := levelIdentifier.(*indexables.Function)
			levelIdentifier = _fun
			finalSymbolFound = true
		}

		if searchingFinalSymbol && finalSymbolFound {
			break
		}

		parentDepth--
		if parentDepth >= 0 {
			currentToken = searchParams.parentSymbols[parentDepth]
		} else {
			currentToken = searchParams.selectedSymbol
			searchingFinalSymbol = true
		}
	}

	return levelIdentifier
}

func findDeepFirst(identifier string, position protocol.Position, function *indexables.Function, depth uint, mode FindMode) (indexables.Indexable, uint) {
	if function.FunctionType() == indexables.UserDefined && identifier == function.GetFullName() {
		return function, depth
	}

	// When mode is InPosition, we are specifying it is important for us that
	// the function being searched contains the position specified.
	// We use mode = AnyPosition when search for a symbol definition outside the document where the user has its cursor. For example, we are looking in imported files or files of the same module.
	if mode == InScope &&
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
		if result, resultDepth := findDeepFirst(identifier, position, &child, depth+1, mode); result != nil {
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
	var enumerator indexables.Enumerator
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
	var faultConstant indexables.FaultConstant
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
