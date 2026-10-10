package symbols_table

import (
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type SymbolsTable struct {
	parsedModulesByDocument map[protocol.DocumentUri]UnitModules
	pendingToResolve        PendingToResolve
}

func NewSymbolsTable() SymbolsTable {
	return SymbolsTable{
		parsedModulesByDocument: make(map[protocol.DocumentUri]UnitModules),
		pendingToResolve:        NewPendingToResolve(),
	}
}

func (st *SymbolsTable) Register(unitModules UnitModules, pendingToResolve PendingToResolve) {
	st.parsedModulesByDocument[unitModules.DocId()] = unitModules

	// Merge pendingToResolve types
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

func (st *SymbolsTable) DeleteDocument(docId string) {
	delete(st.parsedModulesByDocument, docId)
}
func (st *SymbolsTable) RenameDocument(oldDocId string, newDocId string) {
	if val, ok := st.parsedModulesByDocument[oldDocId]; ok {
		// Asignar el valor a la nueva clave
		st.parsedModulesByDocument[newDocId] = val
		// Eliminar la clave antigua
		delete(st.parsedModulesByDocument, oldDocId)
	}
}

func (st *SymbolsTable) GetByDoc(docId string) *UnitModules {
	value, _ := st.parsedModulesByDocument[docId]
	return &value
}

func (st SymbolsTable) All() map[protocol.DocumentUri]UnitModules {
	return st.parsedModulesByDocument
}

// Processes pending Types to resolve, searching
func (st *SymbolsTable) resolveTypes() {
	// Lookups are built lazily and shared by every pending type of this pass.
	// Rebuilding them per pending type made indexing quadratic on the number
	// of registered documents (stdlib alone registers one document per file).
	var lookup *typeResolutionLookup

	// Review all pending types, and see if we can resolve them
	for moduleName, typesContext := range st.pendingToResolve.typesByModule {
		// Reviewing types in `moduleName`
		for x := len(typesContext) - 1; x >= 0; x-- {
			if typesContext[x].IsSolved() {
				continue
			}

			if lookup == nil {
				lookup = st.newTypeResolutionLookup()
			}
			st.tryToSolveType(&typesContext[x], moduleName, lookup)
		}
	}

	// Resolve inline sub struct
	st.expendStructSubtypes()
}

// typeResolutionLookup caches, for a single resolveTypes pass, which module
// owns each symbol name and whether an imported module path exists.
type typeResolutionLookup struct {
	moduleBySymbolName map[string]string
	importFound        map[string]bool
}

func (st *SymbolsTable) newTypeResolutionLookup() *typeResolutionLookup {
	lookup := &typeResolutionLookup{
		moduleBySymbolName: make(map[string]string),
		importFound:        make(map[string]bool),
	}

	for _, parsedModules := range st.parsedModulesByDocument {
		// Use ModuleIds() to cheaply copy the slice of module names
		// Using Modules() appears to lead to expensive clones
		for _, moduleName := range parsedModules.ModuleIds() {
			module := parsedModules.Get(moduleName)

			// Use ChildrenNames() since we are only comparing names
			// Avoid expensive cloning of children
			for _, childName := range module.ChildrenNames() {
				if _, exists := lookup.moduleBySymbolName[childName]; !exists {
					lookup.moduleBySymbolName[childName] = module.GetName()
				}
			}
		}
	}

	return lookup
}

func (st *SymbolsTable) isImportedModuleRegistered(imported string, lookup *typeResolutionLookup) bool {
	if found, cached := lookup.importFound[imported]; cached {
		return found
	}

	mpath := symbols.NewModulePathFromString(imported)
	found := false
	for _, parsedModules := range st.parsedModulesByDocument {
		if parsedModules.HasExplicitlyImportedModules(mpath) {
			found = true
			break
		}
	}
	lookup.importFound[imported] = found

	return found
}

func (st *SymbolsTable) tryToSolveType(typeContext *PendingTypeContext, moduleName string, lookup *typeResolutionLookup) {
	if len(typeContext.contextModule.Imports) > 0 {
		// Only search when at least one imported module is registered.
		importRegistered := false
		for _, imported := range typeContext.contextModule.Imports {
			if st.isImportedModuleRegistered(imported, lookup) {
				importRegistered = true
				break
			}
		}
		if !importRegistered {
			// Not found! Keep it registered as pending
			return
		}
	}

	if owner, ok := lookup.moduleBySymbolName[typeContext.vType.GetName()]; ok {
		typeContext.vType.SetModule(owner)
		typeContext.Solve()
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
