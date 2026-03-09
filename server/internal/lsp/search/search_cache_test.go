package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestFindSymbolDeclarationInWorkspace_cacheInvalidatesAcrossDocumentRefresh(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	bodyA, posA := parseBodyWithCursor(`
		module app;
		fn void main() {
			int a = 1;
			a|||;
		}
	`)
	state.registerDoc("app.c3", bodyA)

	first := search.FindSymbolDeclarationInWorkspace("app.c3", posA, state.state)
	if assert.True(t, first.IsSome()) {
		assert.Equal(t, "a", first.Get().GetName())
	}

	bodyB, posB := parseBodyWithCursor(`
		module app;
		fn void main() {
			int b = 1;
			b|||;
		}
	`)
	state.registerDoc("app.c3", bodyB)

	second := search.FindSymbolDeclarationInWorkspace("app.c3", posB, state.state)
	if assert.True(t, second.IsSome()) {
		assert.Equal(t, "b", second.Get().GetName())
	}
}

func TestBuildParentResolutionCacheKey_includes_symbol_position(t *testing.T) {
	spA := search_params.NewSearchParams(
		"value",
		symbols.NewRange(10, 4, 10, 9),
		"app",
		option.Some("app.c3"),
	)
	spB := search_params.NewSearchParams(
		"value",
		symbols.NewRange(20, 8, 20, 13),
		"app",
		option.Some("app.c3"),
	)

	keyA := buildParentResolutionCacheKey(spA, 7)
	keyB := buildParentResolutionCacheKey(spB, 7)

	assert.NotEqual(t, keyA, keyB)
}
