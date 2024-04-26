package language

import (
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

type FindOptions struct {
	continueOnModule bool
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams) indexables.Indexable {
	return l._findClosestSymbolDeclaration(searchParams, FindOptions{continueOnModule: true})
}

func (l *Language) _findClosestSymbolDeclaration(searchParams SearchParams, options FindOptions) indexables.Indexable {

	var parentIdentifier indexables.Indexable
	position := searchParams.position
	// Check if there's parent contextual information in searchParams
	if searchParams.HasParentSymbol() {
		subSearchParams := NewSearchParams(searchParams.parentSymbol, position, searchParams.docId)
		parentIdentifier = l.findClosestSymbolDeclaration(subSearchParams)
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
			sp := NewSearchParams(variable.Type, position, variable.GetDocumentURI())
			parentTypeSymbol := l.findClosestSymbolDeclaration(sp)
			//fmt.Sprint(parentTypeSymbol.GetName() + "." + searchParams.selectedSymbol)
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

		case indexables.Enum:
			// Search searchParams.selectedSymbol in enumerators
			_enum := parentIdentifier.(indexables.Enum)
			enumerators := _enum.GetEnumerators()
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == searchParams.selectedSymbol {
					//searchParams.selectedSymbol = enumerators[i].GetName()
					return enumerators[i]
				}
			}

		case indexables.Fault:
			// Search searchParams.selectedSymbol in enumerators
			_fault := parentIdentifier.(indexables.Fault)
			enumerators := _fault.GetConstants()
			for i := 0; i < len(enumerators); i++ {
				if enumerators[i].GetName() == searchParams.selectedSymbol {
					//searchParams.selectedSymbol = enumerators[i].GetName()
					return enumerators[i]
				}
			}
		default:
			fmt.Println("Parent Type desconocido!")
		}
	}

	scopedTree, found := l.functionTreeByDocument[searchParams.docId]
	if !found {
		return nil
	}

	// Go through every element defined in scopedTree
	identifier, _ := findDeepFirst(searchParams.selectedSymbol, position, &scopedTree, 0, searchParams.findMode)

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
			found := l._findClosestSymbolDeclaration(
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

	// TODO search in imported files in docId
	// -----------

	// Not found yet, let's try search the selectedSymbol defined as global in other files
	// Note: Iterating a map is not guaranteed to be done always in the same order.

	// Not found...
	return nil
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
