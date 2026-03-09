package search_v2

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	searchv1 "github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

// TypeResolver handles resolving symbols to their underlying types
type TypeResolver struct {
	projState *project_state.ProjectState
}

func NewTypeResolver(projState *project_state.ProjectState) *TypeResolver {
	return &TypeResolver{projState: projState}
}

// ResolveToInspectable fully resolves a symbol to an inspectable type
func (r *TypeResolver) ResolveToInspectable(
	symbol symbols.Indexable,
	ctx AccessContext,
	isLastSegment bool,
) (symbols.Indexable, AccessContext, bool) {
	const MAX_RESOLUTION_DEPTH = 100

	for depth := 0; depth < MAX_RESOLUTION_DEPTH; depth++ {
		// If we're at the last segment and hit a distinct, don't resolve it
		if isLastSegment {
			if _, ok := symbol.(*symbols.TypeDef); ok {
				return symbol, ctx, true
			}
		}

		// If already inspectable, we're done
		if r.isInspectable(symbol) {
			return symbol, ctx, true
		}

		// Resolve one level
		originalSymbol := symbol
		symbol = r.resolveOneLevel(symbol)
		if symbol == nil {
			return nil, ctx, false
		}

		// Update context based on the resolution
		ctx = ctx.AfterResolving(originalSymbol, symbol)
	}

	return nil, ctx, false // Hit max depth
}

func (r *TypeResolver) resolveOneLevel(symbol symbols.Indexable) symbols.Indexable {
	return searchv1.ResolveOneLevelSymbol(symbol, r.projState, nil)
}

func (r *TypeResolver) isInspectable(elm symbols.Indexable) bool {
	switch elm.(type) {
	case *symbols.Variable, *symbols.Function, *symbols.StructMember, *symbols.Alias, *symbols.TypeDef:
		return false
	default:
		return true
	}
}
