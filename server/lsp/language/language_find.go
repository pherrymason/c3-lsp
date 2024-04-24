package language

import (
	"errors"
	"fmt"

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

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams, position protocol.Position) indexables.Indexable {

	var parentIdentifier indexables.Indexable
	// Check if there's parent contextual information in searchParams
	if searchParams.HasParentSymbol() {
		subSearchParams := NewSearchParams(searchParams.parentSymbol, searchParams.docId)
		parentIdentifier = l.findClosestSymbolDeclaration(subSearchParams, position)
	}

	if parentIdentifier != nil {
		//fmt.Printf("Parent found")
		// Current symbol is child of something.
		// Examples:
		// - Property of a struct variable:
		// 		cat.name = "kowalsky";
		// - Method call:
		//		point.add(10);

		// If parent is Variable -> Look for variable.Type
		switch parentIdentifier.(type) {
		case indexables.Variable:
			variable, ok := parentIdentifier.(indexables.Variable)
			if !ok {
				panic("Error")
			}
			sp := NewSearchParams(variable.Type, variable.GetDocumentURI())
			parentTypeSymbol := l.findClosestSymbolDeclaration(sp, position)
			fmt.Sprint(parentTypeSymbol.GetName() + "." + searchParams.selectedSymbol)
			switch parentTypeSymbol.(type) {
			case indexables.Struct:
				// Search searchParams.selectedSymbol in members
				_struct := parentTypeSymbol.(indexables.Struct)
				members := _struct.GetMembers()
				for i := 0; i < len(members); i++ {
					if members[i].GetName() == searchParams.selectedSymbol {
						searchParams.selectedSymbol = members[i].GetType()
						return members[i]
					}
				}
			}

			//searchParams.selectedSymbol = variableTypeSymbol.GetName() + "." + searchParams.selectedSymbol
		}
	}

	identifier, _ := l.findSymbolDeclarationInDocPositionScope(searchParams, position)

	if identifier != nil {
		return identifier
	}

	// TODO search in imported files in docId
	// -----------

	// Not found yet, let's try search the selectedSymbol defined as global in other files
	// Note: Iterating a map is not guaranteed to be done always in the same order.
	// TODO -> recheck if this is correct
	/*
		for _, scope := range l.functionTreeByDocument {
			found, foundDepth := findDeepFirst(searchParams.selectedSymbol, position, &scope, 0, AnyPosition)

			if found != nil && (foundDepth <= 1) {
				return found
			}
		}*/

	// Not found...
	return nil
}

// SearchParams for selectedSymbol in docId
func (l *Language) findSymbolDeclarationInDocPositionScope(searchParams SearchParams, position protocol.Position) (indexables.Indexable, error) {
	scopedTree, found := l.functionTreeByDocument[searchParams.docId]
	if !found {
		return nil, errors.New(fmt.Sprint("Skipping as no symbols for ", searchParams.docId, " were indexed."))
	}

	// Go through every element defined in scopedTree
	symbol, _ := findDeepFirst(searchParams.selectedSymbol, position, &scopedTree, 0, InPosition)
	return symbol, nil
}

func findDeepFirst(identifier string, position protocol.Position, function *indexables.Function, depth uint, mode FindMode) (indexables.Indexable, uint) {
	if identifier == function.GetFullName() {
		return function, depth
	}

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

	// TODO search in Faults
	// TODO Search in ¿Interfaces?

	return nil, depth
}
