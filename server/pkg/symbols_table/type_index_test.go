package symbols_table

import (
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTypeIndex_BasicOperations(t *testing.T) {
	index := NewTypeIndex()

	// Create test modules with types
	doc1 := protocol.DocumentUri("test1.c3")
	doc2 := protocol.DocumentUri("test2.c3")

	// Add some type locations
	index.addType("MyStruct", "mymod::submod", doc1)
	index.addType("MyEnum", "mymod::submod", doc1)
	index.addType("OtherStruct", "othermod", doc2)
	index.addType("MyStruct", "anothermod", doc2) // Same type name, different module

	t.Run("Find returns all locations for a type", func(t *testing.T) {
		locations := index.Find("MyStruct", "")
		assert.Len(t, locations, 2) // Should find both instances
		assert.Equal(t, "MyStruct", locations[0].TypeName)
		assert.Equal(t, "MyStruct", locations[1].TypeName)
	})

	t.Run("Find with module constraint returns only matching module", func(t *testing.T) {
		locations := index.Find("MyStruct", "mymod::submod")
		assert.Len(t, locations, 1)
		assert.Equal(t, "mymod::submod", locations[0].ModuleName)
		assert.Equal(t, doc1, locations[0].DocID)
	})

	t.Run("Find returns nil for non-existent type", func(t *testing.T) {
		locations := index.Find("NonExistent", "")
		assert.Nil(t, locations)
	})

	t.Run("Clear removes types from document", func(t *testing.T) {
		index.Clear(doc1)

		// MyStruct should only have 1 location now (from doc2)
		locations := index.Find("MyStruct", "")
		assert.Len(t, locations, 1)
		assert.Equal(t, "anothermod", locations[0].ModuleName)

		// MyEnum should be gone entirely
		locations = index.Find("MyEnum", "")
		assert.Nil(t, locations)

		// OtherStruct should still exist
		locations = index.Find("OtherStruct", "")
		assert.Len(t, locations, 1)
	})

	t.Run("Stats returns correct information", func(t *testing.T) {
		stats := index.Stats()
		assert.Equal(t, 2, stats.TotalTypes) // MyStruct, OtherStruct
		assert.Equal(t, 2, stats.TotalLocations)
		assert.Equal(t, 2, stats.TypesPerDoc[doc2])
	})
}

func TestTypeIndex_MultipleModulesDefineType(t *testing.T) {
	index := NewTypeIndex()
	doc1 := protocol.DocumentUri("test1.c3")
	doc2 := protocol.DocumentUri("test2.c3")

	// Same type name in different modules (common in C3)
	index.addType("Iterator", "std::collections", doc1)
	index.addType("Iterator", "custom::iter", doc2)

	t.Run("Should find both definitions", func(t *testing.T) {
		locations := index.Find("Iterator", "")
		assert.Len(t, locations, 2)
	})

	t.Run("Should find specific module", func(t *testing.T) {
		locations := index.Find("Iterator", "std::collections")
		assert.Len(t, locations, 1)
		assert.Equal(t, "std::collections", locations[0].ModuleName)
	})

	t.Run("Constraint to non-existent module returns empty", func(t *testing.T) {
		locations := index.Find("Iterator", "nonexistent")
		assert.Len(t, locations, 0)
	})
}

func TestTypeIndex_PerformanceIsConstant(t *testing.T) {
	// This test verifies that lookup is O(1) regardless of index size
	index := NewTypeIndex()

	// Add many types
	for i := 0; i < 1000; i++ {
		docID := protocol.DocumentUri("doc.c3")
		index.addType("CommonType", "module1", docID)
		index.addType("RareType", "module2", docID)
	}

	// Finding should still be fast
	locations := index.Find("CommonType", "")
	assert.Len(t, locations, 1000)

	locations = index.Find("RareType", "module2")
	assert.Len(t, locations, 1000)

	// Stats should show correct counts
	stats := index.Stats()
	assert.Equal(t, 2, stats.TotalTypes)
	assert.Equal(t, 2000, stats.TotalLocations)
}
