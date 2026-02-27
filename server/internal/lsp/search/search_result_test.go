package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestSearchResultSet_NilIndexableProducesNone(t *testing.T) {
	result := NewSearchResult(TrackedModules{})

	var variable *symbols.Variable
	var indexable symbols.Indexable = variable

	result.Set(indexable)

	assert.True(t, result.IsNone())
}
