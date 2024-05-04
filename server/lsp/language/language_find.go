package language

import (
	"fmt"
	"strings"

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

type DebugFind struct {
	depth int
}

func (d DebugFind) goIn() DebugFind {
	return DebugFind{depth: d.depth + 1}
}

func min(num1, num2 int) int {
	if num1 < num2 {
		return num1
	}
	return num2
}
func (l *Language) debug(message string, debugger DebugFind) {
	maxo := min(debugger.depth, 8)
	prep := "|" + strings.Repeat("_", maxo)
	if len(prep) > 10 {
		return
	}
	l.logger.Debug(fmt.Sprintf("%s %s", prep, message))
}

// Finds the closest selectedSymbol based on current scope.
// If not present in current Scope:
// - Search in files of same module
// - SearchParams in imported files (TODO)
// - SearchParams in global symbols in workspace
func (l *Language) findClosestSymbolDeclaration(searchParams SearchParams, debugger DebugFind) indexables.Indexable {
	l.debug(fmt.Sprintf("findClosestSymbolDeclaration on doc %s", searchParams.docId), debugger)

	position := searchParams.selectedSymbol.position
	// Check if there's parent contextual information in searchParams
	if searchParams.HasParentSymbol() {
		identifier := l.findInParentSymbols(searchParams, debugger)
		if identifier != nil {
			return identifier
		}
	}

	if searchParams.HasModuleSpecified() {
		symbol := l._findSymbolDeclarationInModule(searchParams, debugger.goIn())
		if symbol != nil {
			return symbol
		}
	} else {
		documentModules, found := l.functionTreeByDocument[searchParams.docId]
		if !found {
			return nil
		}

		for module, scopedTree := range documentModules.SymbolsByModule() {
			l.debug(fmt.Sprintf("Checking module \"%s\"", module), debugger)
			// Go through every element defined in scopedTree
			identifier, _ := findDeepFirst(searchParams.selectedSymbol.token, position, scopedTree, 0, searchParams.scopeMode)

			if identifier != nil {
				return identifier
			}

			if searchParams.continueOnModules {
				found := l.findSymbolsInModuleOtherFiles(scopedTree.GetModule(), searchParams, debugger.goIn())
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
					l.debug(fmt.Sprintf("findClosestSymbolDeclaration: search in imported module \"%s\"", module), debugger)
					symbol := l._findSymbolDeclarationInModule(sp, debugger.goIn())
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

func (l *Language) findSymbolsInModuleOtherFiles(module indexables.ModulePath, searchParams SearchParams, debugger DebugFind) indexables.Indexable {

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
				DebugFind{depth: debugger.depth + 1},
			)
			if found != nil {
				return found
			}
		}
	}

	return nil
}

func (l *Language) _findSymbolDeclarationInModule(searchParams SearchParams, debugger DebugFind) indexables.Indexable {
	expectedModule := searchParams.modulePath.GetName()

	for docId, modulesByDoc := range l.functionTreeByDocument {
		for _, scope := range modulesByDoc.SymbolsByModule() {
			if scope.GetModuleString() != expectedModule { // TODO Ignore current doc we are comming from
				continue
			}
			if inSlice(scope.GetModuleString(), searchParams.traversedModules) {
				continue
			}

			l.debug(fmt.Sprintf("findSymbolDeclarationInModule: search symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)
			symbol := l.findClosestSymbolDeclaration(SearchParams{
				selectedSymbol:    searchParams.selectedSymbol,
				docId:             docId,
				scopeMode:         searchParams.scopeMode,
				continueOnModules: true,
				traversedModules:  append(searchParams.traversedModules, scope.GetModuleString()),
			}, DebugFind{depth: debugger.depth + 1})
			l.debug(fmt.Sprintf("end searching symbols in module \"%s\" file \"%s\"", scope.GetModuleString(), docId), debugger)
			if symbol != nil {
				return symbol
			}
		}
	}

	return nil
}

// TODO Document what this does
func (l *Language) findInParentSymbols(searchParams SearchParams, debugger DebugFind) indexables.Indexable {
	var found indexables.Indexable

	tokens := append(searchParams.parentSymbols, searchParams.selectedSymbol)
	totalTokens := len(tokens)
	var parentSymbol indexables.Indexable

	l.debug("findInParentSymbols", debugger)

	for it, token := range tokens {
		isLastToken := it == (totalTokens - 1)

		if parentSymbol == nil {
			levelToken := token.token
			levelDocId := searchParams.docId
			for {
				symbolFound := false
				levelSearchParams := NewSearchParams(
					levelToken,
					token.position,
					levelDocId,
				)
				found = l.findClosestSymbolDeclaration(levelSearchParams, debugger.goIn())
				switch found.(type) {
				case indexables.Variable:
					variable, _ := found.(indexables.Variable)
					levelToken = variable.GetType().GetName()
					levelDocId = variable.GetDocumentURI()

				case indexables.Struct, indexables.Enum, indexables.Fault: // TODO Is this tested for Enum and Fault?
					parentSymbol = found
					symbolFound = true
				}

				if symbolFound {
					parentSymbol = found
					break
				}
			}
		} else {
			var nextSymbol indexables.Indexable
			// Search inside parentSymbol
			switch parentSymbol.(type) {
			case indexables.Struct:
				strukt, _ := parentSymbol.(indexables.Struct)
				members := strukt.GetMembers()
				for i := 0; i < len(members); i++ {
					if members[i].GetName() == token.token {
						if !isLastToken {
							levelSearchParams := NewSearchParams(
								members[i].GetType(), // Bug: When searching System.init() this evaluated to System.System!!
								token.position,
								strukt.GetDocumentURI(),
							)
							nextSymbol = l.findClosestSymbolDeclaration(levelSearchParams, debugger.goIn())
						} else {
							nextSymbol = members[i]
						}
						break
					}
				}

				if nextSymbol == nil {
					// Not found... should search in functions
					levelSearchParams := NewSearchParams(
						strukt.GetName()+"."+token.token,
						token.position,
						strukt.GetDocumentURI(),
					)
					nextSymbol = l.findClosestSymbolDeclaration(levelSearchParams, debugger.goIn())
				}

			case indexables.Enum: // TODO Is this tested?
				// Search searchParams.selectedSymbol in enumerators
				_enum := parentSymbol.(indexables.Enum)
				enumerators := _enum.GetEnumerators()
				for i := 0; i < len(enumerators); i++ {
					if enumerators[i].GetName() == token.token {
						nextSymbol = enumerators[i]
						break
					}
				}

			case indexables.Fault: // TODO Is this tested?
				_fault := parentSymbol.(indexables.Fault)
				enumerators := _fault.GetConstants()
				for i := 0; i < len(enumerators); i++ {
					if enumerators[i].GetName() == token.token {
						nextSymbol = enumerators[i]
						break
					}
				}
			}

			if nextSymbol == nil {
				return nil
			}
			parentSymbol = nextSymbol
		}
	}

	found = parentSymbol

	l.debug("finish findInParentSymbols", debugger)
	return found
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
