package project_state

import (
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ScopeCompletionIndex pre-computes, for every module name A that appears in the
// snapshot, the set of module names S that satisfy:
//
//	S.IsImplicitlyImported(A)
//
// which means: S == A, S is a sub-module of A, or A is a sub-module of S.
//
// The index is built once inside buildSnapshotIndexes (under the state lock) and
// is read-only after construction — no synchronisation is needed at query time.
type ScopeCompletionIndex struct {
	// reachable[acceptedModuleName] = sorted slice of module names S
	// that are implicitly imported by acceptedModuleName.
	reachable map[string][]string
}

// buildScopeCompletionIndex constructs the index from the full unit-modules map.
// Complexity: O(M²) where M is the number of unique module names.
func buildScopeCompletionIndex(all map[protocol.DocumentUri]symbols_table.UnitModules) *ScopeCompletionIndex {
	// Collect all unique (name, ModulePath) pairs.
	// A module name may appear in multiple documents; we only need one
	// ModulePath per name to perform the prefix-ancestry check.
	uniquePaths := make(map[string]symbols.ModulePath)
	for _, unitModules := range all {
		for _, mod := range unitModules.Modules() {
			name := mod.GetName()
			if _, seen := uniquePaths[name]; !seen {
				uniquePaths[name] = mod.GetModule()
			}
		}
	}

	// Build a flat slice for the O(N²) cross product.
	type entry struct {
		name string
		path symbols.ModulePath
	}
	entries := make([]entry, 0, len(uniquePaths))
	for name, path := range uniquePaths {
		entries = append(entries, entry{name, path})
	}

	reachable := make(map[string][]string, len(entries))

	for _, accepted := range entries {
		var hits []string
		for _, scope := range entries {
			if scope.path.IsImplicitlyImported(accepted.path) {
				hits = append(hits, scope.name)
			}
		}
		reachable[accepted.name] = hits
	}

	return &ScopeCompletionIndex{reachable: reachable}
}

// ModuleNames returns the slice of module names that are implicitly reachable
// when acceptedModuleName is in scope.  Returns nil when the module is not
// present in the index (caller should fall back to the linear scan).
func (idx *ScopeCompletionIndex) ModuleNames(acceptedModuleName string) []string {
	if idx == nil {
		return nil
	}
	names, ok := idx.reachable[acceptedModuleName]
	if !ok {
		return nil
	}
	return names
}
