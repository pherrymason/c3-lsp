package search_v2

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func TestAccessContext_uses_search_distinct_constants(t *testing.T) {
	ctx := NewAccessContext()

	assert.Equal(t, search.NotFromDistinct, ctx.FromDistinct)

	inlineDistinct := symbols.NewTypeDef(
		"Meter",
		nil,
		true,
		"",
		"app",
		"a.c3",
		symbols.NewRange(0, 0, 0, 0),
		symbols.NewRange(0, 0, 0, 0),
	)
	resolvedType := symbols.NewStruct(
		"Distance",
		nil,
		nil,
		"app",
		"a.c3",
		symbols.NewRange(0, 0, 0, 0),
		symbols.NewRange(0, 0, 0, 0),
	)

	resolved := ctx.AfterResolving(&inlineDistinct, &resolvedType)
	assert.Equal(t, search.InlineDistinct, resolved.FromDistinct)

	nonInlineDistinct := symbols.NewTypeDef(
		"Second",
		nil,
		false,
		"",
		"app",
		"a.c3",
		symbols.NewRange(0, 0, 0, 0),
		symbols.NewRange(0, 0, 0, 0),
	)
	resolved = ctx.AfterResolving(&nonInlineDistinct, &resolvedType)
	assert.Equal(t, search.NonInlineDistinct, resolved.FromDistinct)
}
