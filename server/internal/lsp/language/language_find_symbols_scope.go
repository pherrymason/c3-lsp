package language

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type FindSymbolsParams struct {
	docId              string
	scopedToModulePath option.Option[symbols.ModulePath]
	position           option.Option[symbols.Position]
}

// Returns all symbols in scope.
// Detail: StructMembers and Enumerables are inlined
func (l *Language) findSymbolsInScope(params FindSymbolsParams) []symbols.Indexable {
	var symbolsCollection []symbols.Indexable

	var currentContextModules []symbols.ModulePath
	if params.position.IsSome() {
		// Find current module
		for _, module := range l.symbolsTable.GetByDoc(params.docId).Modules() {
			if module.GetDocumentRange().HasPosition(params.position.Get()) {
				currentContextModules = append(currentContextModules, module.GetModule())
				break
			}
		}
	}

	if params.scopedToModulePath.IsSome() {
		currentContextModules = append(currentContextModules, params.scopedToModulePath.Get())
	}

	// -------------------------------------
	// Modules where we can extract symbols
	// -------------------------------------
	modulesToLook := l.implicitImportedParsedModules(
		currentContextModules,
		option.None[string](),
	)

	for _, module := range modulesToLook {
		// Only include Module itself, when text is not already prepended with same module name
		if params.scopedToModulePath.IsNone() || (params.scopedToModulePath.IsSome() && module.GetName() != params.scopedToModulePath.Get().GetName()) {
			symbolsCollection = append(symbolsCollection, module)
		}

		for _, variable := range module.Variables {
			symbolsCollection = append(symbolsCollection, variable)
		}
		for _, enum := range module.Enums {
			symbolsCollection = append(symbolsCollection, enum)
			for _, enumerable := range enum.GetEnumerators() {
				symbolsCollection = append(symbolsCollection, enumerable)
			}
		}
		for _, strukt := range module.Structs {
			symbolsCollection = append(symbolsCollection, strukt)
		}
		for _, def := range module.Defs {
			symbolsCollection = append(symbolsCollection, def)
		}
		for _, fault := range module.Faults {
			symbolsCollection = append(symbolsCollection, fault)
			for _, constant := range fault.GetConstants() {
				symbolsCollection = append(symbolsCollection, constant)
			}
		}
		for _, interfaces := range module.Interfaces {
			symbolsCollection = append(symbolsCollection, interfaces)
		}

		for _, function := range module.ChildrenFunctions {
			symbolsCollection = append(symbolsCollection, function)
			if params.position.IsSome() && function.GetDocumentRange().HasPosition(params.position.Get()) {
				symbolsCollection = append(symbolsCollection, function)

				for _, variable := range function.Variables {
					l.logger.Debug(fmt.Sprintf("Checking %s variable:", variable.GetName()))
					declarationPosition := variable.GetIdRange().End
					if declarationPosition.Line > uint(params.position.Get().Line) ||
						(declarationPosition.Line == uint(params.position.Get().Line) && declarationPosition.Character > uint(params.position.Get().Character)) {
						continue
					}

					symbolsCollection = append(symbolsCollection, variable)
				}
			}
		}
	}

	return symbolsCollection
}
