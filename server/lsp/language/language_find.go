package language

import (
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (l *Language) findAllScopeSymbols(scopeFunction *indexables.Function, position protocol.Position) []indexables.Indexable {
	var symbols []indexables.Indexable

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

	for _, function := range scopeFunction.ChildrenFunctions {
		if function.GetDocumentRange().HasPosition(position) {
			symbols = append(symbols, function)
		}
	}

	return symbols
}

type FindOptions struct {
	continueOnModule bool
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams) indexables.Indexable {
	return l._findClosestSymbolDeclarationInDoc(searchParams, FindOptions{continueOnModule: true})
}

func (l *Language) _findClosestSymbolDeclarationInDoc(searchParams SearchParams, options FindOptions) indexables.Indexable {

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
		scopedTree, found := l.functionTreeByDocument[searchParams.docId]
		if !found {
			return nil
		}

		// Go through every element defined in scopedTree
		identifier, _ := findDeepFirst(searchParams.selectedSymbol.token, position, &scopedTree, 0, searchParams.findMode)

		if identifier != nil {
			return identifier
		}

		if options.continueOnModule {
			// Try to find in same module, different files
			currentModule := l.functionTreeByDocument[searchParams.docId].GetModule()
			for _, scope := range l.functionTreeByDocument {
				if scope.GetDocumentURI() == searchParams.docId || scope.GetModule() != currentModule {
					continue
				}

				// Can we just call findDeepFirst() directly instead?
				found := l._findClosestSymbolDeclarationInDoc(
					SearchParams{
						selectedSymbol: searchParams.selectedSymbol,
						docId:          scope.GetDocumentURI(),
						findMode:       AnyPosition,
					},
					FindOptions{continueOnModule: false},
				)
				if found != nil {
					return found
				}
			}
		}

		if len(scopedTree.Imports) > 0 {
			for i := 0; i < len(scopedTree.Imports); i++ {
				symbol := l._findSymbolDeclarationInModule(
					SearchParams{
						selectedSymbol: searchParams.selectedSymbol,
						modulePath:     indexables.NewModulePath([]string{scopedTree.Imports[i]}),
						findMode:       AnyPosition,
					},
				)
				if symbol != nil {
					return symbol
				}
			}
		}
	}

	// Not found yet, let's try search the selectedSymbol defined as global in other files
	// Note: Iterating a map is not guaranteed to be done always in the same order.

	// Not found...
	return nil
}

func (l *Language) _findSymbolDeclarationInModule(searchParams SearchParams) indexables.Indexable {
	for docId, scope := range l.functionTreeByDocument {
		if scope.GetModule() != searchParams.modulePath.GetName() { // TODO Ignore current doc we are comming from
			continue
		}

		symbol := l._findClosestSymbolDeclarationInDoc(SearchParams{
			selectedSymbol: searchParams.selectedSymbol,
			docId:          docId,
			findMode:       searchParams.findMode,
		}, FindOptions{continueOnModule: true})

		if symbol != nil {
			return symbol
		}
	}

	return nil
}

func (l *Language) findInParentSymbols(searchParams SearchParams) indexables.Indexable {
	var levelIdentifier indexables.Indexable

	parentSymbolsCount := len(searchParams.parentSymbols)
	rootSymbol := searchParams.parentSymbols[parentSymbolsCount-1]
	levelSearchParams := SearchParams{
		selectedSymbol: rootSymbol,
		docId:          searchParams.docId,
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
	if mode == InPosition &&
		!function.GetDocumentRange().HasPosition(position) {
		return nil, depth
	}

	for _, child := range function.ChildrenFunctions {
		if result, resultDepth := findDeepFirst(identifier, position, &child, depth+1, mode); result != nil {
			return result, resultDepth
		}
	}

	variable, foundVariableInThisScope := function.Variables[identifier]
	if foundVariableInThisScope {
		return variable, depth
	}

	enum, foundEnumInThisScope := function.Enums[identifier]
	if foundEnumInThisScope {
		return enum, depth
	}

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
