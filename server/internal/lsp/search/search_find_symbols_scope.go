package search

import (
	"fmt"
	"strings"

	p "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
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
func (s *Search) findSymbolsInScope(params FindSymbolsParams, state *p.ProjectState) []symbols.Indexable {
	var symbolsCollection []symbols.Indexable

	var currentContextModules []symbols.ModulePath
	var currentModule *symbols.Module
	if params.position.IsSome() {
		// Find current module
		for _, module := range state.GetUnitModulesByDoc(params.docId).Modules() {
			if module.GetDocumentRange().HasPosition(params.position.Get()) {
				// Only include current module in the search if there is no scopedToModule
				if params.scopedToModulePath.IsNone() {
					currentContextModules = append(currentContextModules, module.GetModule())

					// Also include explicitly imported modules so their symbols
					for _, importedModule := range module.Imports {
						currentContextModules = append(currentContextModules, symbols.NewModulePathFromString(importedModule))
					}
				}
				currentModule = module
				break
			}
		}
	}

	if params.scopedToModulePath.IsSome() && currentModule != nil {
		// We must take into account that scopedModule path might be a partial path module
		for _, importedModule := range currentModule.Imports {
			if strings.HasSuffix(importedModule, params.scopedToModulePath.Get().GetName()) {
				currentContextModules = append(currentContextModules, symbols.NewModulePathFromString(importedModule))
			}
		}

		currentContextModules = append(currentContextModules, params.scopedToModulePath.Get())
	}

	// -------------------------------------
	// Modules where we can extract symbols
	// -------------------------------------
	modulesToLook := s.implicitImportedParsedModules(
		state,
		currentContextModules,
		option.None[string](),
	)

	for _, module := range modulesToLook {
		// Determine if this module is foreign (not the module the cursor is in).
		// @private symbols should only be visible within their own module.
		isForeignModule := currentModule == nil || module.GetModuleString() != currentModule.GetModuleString()

		// Only include Module itself, when text is not already prepended with same module name
		isAlreadyPrepended := params.scopedToModulePath.IsNone() ||
			(params.scopedToModulePath.IsSome() && module.GetName() != params.scopedToModulePath.Get().GetName() && !strings.HasSuffix(module.GetName(), params.scopedToModulePath.Get().GetName()))

		if isAlreadyPrepended {
			symbolsCollection = append(symbolsCollection, module)
		}

		for _, variable := range module.Variables {
			if isForeignModule && variable.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, variable)
		}
		for _, enum := range module.Enums {
			if isForeignModule && enum.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, enum)
			for _, enumerable := range enum.GetEnumerators() {
				symbolsCollection = append(symbolsCollection, enumerable)
			}
		}
		for _, strukt := range module.Structs {
			if isForeignModule && strukt.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, strukt)
		}
		for _, def := range module.Defs {
			if isForeignModule && def.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, def)
		}
		for _, distinct := range module.Distincts {
			if isForeignModule && distinct.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, distinct)
		}
		for _, fault := range module.Faults {
			if isForeignModule && fault.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, fault)
			for _, constant := range fault.GetConstants() {
				symbolsCollection = append(symbolsCollection, constant)
			}
		}
		for _, interfaces := range module.Interfaces {
			if isForeignModule && interfaces.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, interfaces)
		}

		for _, function := range module.ChildrenFunctions {
			if isForeignModule && function.IsPrivate() {
				continue
			}
			symbolsCollection = append(symbolsCollection, function)

			// Only inspect function bodies (local variables) for the current module,
			// not for imported/foreign modules.
			if !isForeignModule && params.position.IsSome() && function.GetDocumentRange().HasPosition(params.position.Get()) {
				symbolsCollection = append(symbolsCollection, function)

				for _, variable := range function.Variables {
					s.logger.Debug(fmt.Sprintf("Checking %s variable:", variable.GetName()))
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
