package search

import (
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func symbolNameMatches(actual string, expected string) bool {
	if actual == expected {
		return true
	}

	if expected == "" || strings.HasPrefix(expected, "@") {
		return false
	}

	if actual == "@"+expected {
		return true
	}

	if dot := strings.LastIndex(expected, "."); dot >= 0 {
		prefix := expected[:dot+1]
		suffix := expected[dot+1:]
		if suffix != "" && !strings.HasPrefix(suffix, "@") {
			return actual == prefix+"@"+suffix
		}
	}

	return false
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
			if symbolNameMatches(child.GetName(), identifier) {
				return child, depth
			}

			if limitSearchInScope &&
				!child.GetDocumentRange().HasPosition(position) {
				continue
			}

			result, resultDepth := findDeepFirst(identifier, position, child, depth+1, limitSearchInScope, scopeMode)
			if result != nil {
				return result, resultDepth
			}
		}
	}

	if depth == 0 || (scopeMode == search_params.InScope) {
		for _, child := range node.ChildrenWithoutScopes() {
			result, resultDepth := findDeepFirst(identifier, position, child, depth+1, limitSearchInScope, scopeMode)
			if result != nil {
				return result, resultDepth
			}
		}
	}

	if depth == 0 && scopeMode == search_params.InModuleRoot {
		for _, child := range node.Children() {
			if symbolNameMatches(child.GetName(), identifier) {
				return child, depth
			}
		}
		for _, child := range node.NestedScopes() {
			if symbolNameMatches(child.GetName(), identifier) {
				return child, depth
			}
		}
	}

	// All elements found in nestable symbols checked, check node itself
	if symbolNameMatches(node.GetName(), identifier) {
		_, ok := node.(*symbols.Module) // Modules will be searched later explicitly.
		if !ok {
			return node, depth
		}
	}

	return nil, depth
}
