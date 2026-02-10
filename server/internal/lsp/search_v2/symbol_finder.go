package search_v2

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type SearchMode int

const (
	LocalScope SearchMode = iota
	ModuleRoot
)

// SymbolFinder performs simple symbol lookups within a single scope
type SymbolFinder struct{}

func NewSymbolFinder() *SymbolFinder {
	return &SymbolFinder{}
}

// FindInScope finds a symbol in a specific scope/module
func (f *SymbolFinder) FindInScope(
	symbolName string,
	scope *symbols.Module,
	searchMode SearchMode,
	position symbols.Position,
) option.Option[symbols.Indexable] {

	if searchMode == LocalScope {
		return f.findInLocalScope(symbolName, scope, position)
	}

	return f.findInModuleRoot(symbolName, scope)
}

// findInModuleRoot performs simple iteration at module root level
func (f *SymbolFinder) findInModuleRoot(
	symbolName string,
	scope *symbols.Module,
) option.Option[symbols.Indexable] {

	// Check direct children at root level
	for _, child := range scope.Children() {
		if child.GetName() == symbolName {
			return option.Some(child)
		}
	}

	// Check nested scopes (functions, but only their names, not their contents)
	for _, nestedScope := range scope.NestedScopes() {
		if nestedScope.GetName() == symbolName {
			return option.Some(nestedScope)
		}
	}

	return option.None[symbols.Indexable]()
}

// findInLocalScope searches within function scope (parameters and local variables)
// Local symbols have priority over module-level symbols (shadowing)
func (f *SymbolFinder) findInLocalScope(
	symbolName string,
	scope *symbols.Module,
	position symbols.Position,
) option.Option[symbols.Indexable] {

	// Find the function containing the position
	function := f.findContainingFunction(scope, position)
	if function != nil {
		// Check function parameters first
		for _, param := range function.GetArguments() {
			if param.GetName() == symbolName {
				return option.Some[symbols.Indexable](param)
			}
		}

		// Check local variables - Variables map contains both params and local vars
		// We need to check all variables that are not in argumentIds
		argumentIds := make(map[string]bool)
		for _, argId := range function.ArgumentIds() {
			argumentIds[argId] = true
		}

		for varId, localVar := range function.Variables {
			if !argumentIds[varId] && localVar.GetName() == symbolName {
				return option.Some[symbols.Indexable](localVar)
			}
		}
	}

	// If not found in local scope, fall back to module root
	return f.findInModuleRoot(symbolName, scope)
}

// findContainingFunction finds the function that contains the given position
func (f *SymbolFinder) findContainingFunction(
	scope *symbols.Module,
	position symbols.Position,
) *symbols.Function {

	for _, nestedScope := range scope.NestedScopes() {
		if function, ok := nestedScope.(*symbols.Function); ok {
			if function.GetDocumentRange().HasPosition(position) {
				return function
			}
		}
	}

	return nil
}
