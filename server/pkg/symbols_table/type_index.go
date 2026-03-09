package symbols_table

import (
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TypeIndex provides O(1) lookup of types by name across all documents.
// This replaces the O(n³) nested loop search in the old implementation.
type TypeIndex struct {
	// typeName -> list of modules that define this type
	byName map[string][]TypeLocation
}

// TypeLocation identifies where a type is defined.
type TypeLocation struct {
	TypeName   string               // Name of the type (e.g., "File")
	ModuleName string               // Full module name (e.g., "std::io")
	DocID      protocol.DocumentUri // Document containing this type
}

// NewTypeIndex creates a new empty type index.
func NewTypeIndex() *TypeIndex {
	return &TypeIndex{
		byName: make(map[string][]TypeLocation),
	}
}

// Index adds all types from the given unit modules to the index.
// This is called when a document is parsed or updated.
func (ti *TypeIndex) Index(unitModules UnitModules) {
	docID := unitModules.DocId()

	for _, module := range unitModules.Modules() {
		moduleName := module.GetName()

		// Index all type-defining symbols (structs, enums, interfaces, etc.)
		for _, typeName := range module.ChildrenNames() {
			ti.addType(typeName, moduleName, docID)
		}
	}
}

// addType registers a type location in the index.
func (ti *TypeIndex) addType(typeName, moduleName string, docID protocol.DocumentUri) {
	location := TypeLocation{
		TypeName:   typeName,
		ModuleName: moduleName,
		DocID:      docID,
	}

	ti.byName[typeName] = append(ti.byName[typeName], location)
}

// Find returns all modules that define the given type name.
// If constrainToModule is not empty, only returns locations in that module.
//
// This is the core optimization: O(1) hash lookup instead of O(n³) nested loops.
func (ti *TypeIndex) Find(typeName string, constrainToModule string) []TypeLocation {
	locations, ok := ti.byName[typeName]
	if !ok {
		return nil
	}

	// If no constraint, return all locations
	if constrainToModule == "" {
		// Return a copy to prevent external modification
		result := make([]TypeLocation, len(locations))
		copy(result, locations)
		return result
	}

	// Filter to only the constrained module
	filtered := []TypeLocation{}
	for _, loc := range locations {
		if loc.ModuleName == constrainToModule {
			filtered = append(filtered, loc)
		}
	}
	return filtered
}

// Clear removes all types from the given document.
// This is called when a document is deleted or before re-indexing it.
func (ti *TypeIndex) Clear(docID protocol.DocumentUri) {
	// Remove all entries from this document
	for typeName, locations := range ti.byName {
		filtered := []TypeLocation{}
		for _, loc := range locations {
			if loc.DocID != docID {
				filtered = append(filtered, loc)
			}
		}

		if len(filtered) > 0 {
			ti.byName[typeName] = filtered
		} else {
			// No more locations for this type name - remove the key entirely
			delete(ti.byName, typeName)
		}
	}
}

// Stats returns index statistics for metrics/debugging.
type IndexStats struct {
	TotalTypes     int                          // Unique type names
	TotalLocations int                          // Total type definitions (one type can be in multiple modules)
	TypesPerDoc    map[protocol.DocumentUri]int // Type count per document
}

// Stats computes statistics about the index.
// Useful for debugging and performance monitoring.
func (ti *TypeIndex) Stats() IndexStats {
	stats := IndexStats{
		TotalTypes:     len(ti.byName),
		TotalLocations: 0,
		TypesPerDoc:    make(map[protocol.DocumentUri]int),
	}

	for _, locations := range ti.byName {
		stats.TotalLocations += len(locations)
		for _, loc := range locations {
			stats.TypesPerDoc[loc.DocID]++
		}
	}

	return stats
}
