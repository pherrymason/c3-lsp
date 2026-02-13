package search

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SearchInterface defines the common interface for symbol search implementations.
// This allows switching between the original Search and the new SearchV2 implementations.
type SearchInterface interface {
	// FindSymbolDeclarationInWorkspace searches for a symbol declaration at the given position
	FindSymbolDeclarationInWorkspace(
		docId string,
		position symbols.Position,
		state *project_state.ProjectState,
	) option.Option[symbols.Indexable]

	// FindImplementationsInWorkspace searches for concrete implementations of the symbol at the given position
	FindImplementationsInWorkspace(
		docId string,
		position symbols.Position,
		state *project_state.ProjectState,
	) []symbols.Indexable

	// BuildCompletionList generates completion suggestions for the given cursor context
	BuildCompletionList(
		ctx context.CursorContext,
		state *project_state.ProjectState,
	) []protocol.CompletionItem
}
