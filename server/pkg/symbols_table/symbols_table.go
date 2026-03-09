package symbols_table

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type SymbolsTable struct {
	parsedModulesByDocument map[protocol.DocumentUri]UnitModules
	pendingToResolve        PendingToResolve
	typeIndex               *TypeIndex // Fast O(1) type lookup index
}

func NewSymbolsTable() SymbolsTable {
	return SymbolsTable{
		parsedModulesByDocument: make(map[protocol.DocumentUri]UnitModules),
		pendingToResolve:        NewPendingToResolve(),
		typeIndex:               NewTypeIndex(),
	}
}

func (st *SymbolsTable) Register(unitModules UnitModules, pendingToResolve PendingToResolve) {
	docID := unitModules.DocId()

	// Store unit modules
	st.parsedModulesByDocument[docID] = unitModules

	// Rebuild type index for this document (Issue #2 fix)
	st.typeIndex.Clear(docID)
	st.typeIndex.Index(unitModules)

	// Clear old pending types for this document (Issue #5 fix)
	st.clearPendingForDocument(docID)

	// Merge new pendingToResolve types
	for moduleName, types := range pendingToResolve.typesByModule {
		st.pendingToResolve.typesByModule[moduleName] = append(
			st.pendingToResolve.typesByModule[moduleName],
			types...,
		)
	}

	st.pendingToResolve.subtyptingToResolve = append(
		st.pendingToResolve.subtyptingToResolve,
		pendingToResolve.subtyptingToResolve...,
	)

	st.resolveTypes()
}

// clearPendingForDocument removes all pending type resolutions for a specific document.
// This prevents memory leaks when documents are re-parsed (Issue #5 fix).
func (st *SymbolsTable) clearPendingForDocument(docID protocol.DocumentUri) {
	// Clear pending type resolutions
	for moduleName := range st.pendingToResolve.typesByModule {
		filtered := []PendingTypeContext{}
		for _, ctx := range st.pendingToResolve.typesByModule[moduleName] {
			// Keep only types NOT from this document
			if ctx.contextModule.GetDocumentURI() != string(docID) {
				filtered = append(filtered, ctx)
			}
		}

		if len(filtered) > 0 {
			st.pendingToResolve.typesByModule[moduleName] = filtered
		} else {
			// No more pending types for this module - remove the key
			delete(st.pendingToResolve.typesByModule, moduleName)
		}
	}

	// Clear struct subtyping resolutions
	filtered := []StructWithSubtyping{}
	for _, strukt := range st.pendingToResolve.subtyptingToResolve {
		if strukt.strukt.GetDocumentURI() != string(docID) {
			filtered = append(filtered, strukt)
		}
	}
	st.pendingToResolve.subtyptingToResolve = filtered
}

func (st *SymbolsTable) DeleteDocument(docId string) {
	delete(st.parsedModulesByDocument, docId)
	st.typeIndex.Clear(docId) // Clear type index for deleted document
}
func (st *SymbolsTable) RenameDocument(oldDocId string, newDocId string) {
	if val, ok := st.parsedModulesByDocument[oldDocId]; ok {
		// Clear old document from index
		st.typeIndex.Clear(oldDocId)

		// Move to new document
		st.parsedModulesByDocument[newDocId] = val
		delete(st.parsedModulesByDocument, oldDocId)

		// Rebuild index with new docID
		st.typeIndex.Index(val)
	}
}

func (st *SymbolsTable) GetByDoc(docId string) *UnitModules {
	value := st.parsedModulesByDocument[docId]
	return &value
}

func (st SymbolsTable) All() map[protocol.DocumentUri]UnitModules {
	return st.parsedModulesByDocument
}

// Processes pending Types to resolve, searching
func (st *SymbolsTable) resolveTypes() {

	// Review all pending types, and see if we can resolve them
	for moduleName, typesContext := range st.pendingToResolve.typesByModule {
		// Reviewing types in `moduleName`
		for x := len(typesContext) - 1; x >= 0; x-- {
			if typesContext[x].IsSolved() {
				continue
			}

			st.tryToSolveType(&typesContext[x], moduleName)
		}
	}

	// Resolve inline sub struct
	st.expendStructSubtypes()
}

func (st *SymbolsTable) tryToSolveType(typeContext *PendingTypeContext, moduleName string) {
	typeName := typeContext.vType.GetName()

	if len(typeContext.contextModule.Imports) > 0 {
		// Check inside imported modules
		// Issue #3 fix: Now we constrain search to the specific imported module!
		for _, imported := range typeContext.contextModule.Imports {
			// Use type index for O(1) lookup constrained to imported module
			locations := st.typeIndex.Find(typeName, imported)

			if len(locations) > 0 {
				// Found! Use first matching location
				typeContext.vType.SetModule(locations[0].ModuleName)
				typeContext.Solve()
				return
			}
		}
		// Not found in imports - keep as pending
	} else {
		// No imports - search globally using index
		locations := st.typeIndex.Find(typeName, "")

		if len(locations) > 0 {
			// Found! Use first matching location
			typeContext.vType.SetModule(locations[0].ModuleName)
			typeContext.Solve()
		}
	}
}

// Resolves inline sub structs
func (st *SymbolsTable) expendStructSubtypes() {

	if len(st.pendingToResolve.subtyptingToResolve) == 0 {
		return
	}

	for _, struktWithSubtyping := range st.pendingToResolve.subtyptingToResolve {
		for _, inlinedMemberName := range struktWithSubtyping.members {

			// Go through all parsed modules searching structs with members to inline
			for _, parsedModules := range st.parsedModulesByDocument {
				for _, module := range parsedModules.Modules() {
					// Search
					for _, strukt := range module.Structs {
						if strukt.GetName() == inlinedMemberName.GetName() {
							struktWithSubtyping.strukt.InheritMembersFrom(inlinedMemberName.GetName(), strukt)
						}
					}
				}
			}
		}
	}

	st.pendingToResolve.subtyptingToResolve = []StructWithSubtyping{}
}

// TypeIndexStats returns statistics about the type index.
// Useful for monitoring performance and debugging.
func (st *SymbolsTable) TypeIndexStats() IndexStats {
	return st.typeIndex.Stats()
}
