package search_v2

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/c3"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SearchV2
// Redesigned from search system (v1) with separated concerns and cleaner architecture
type SearchV2 struct {
	logger       commonlog.Logger
	debugEnabled bool
	fallback     *search.Search // Fallback to old search for features not yet implemented in V2
}

func NewSearchV2(logger commonlog.Logger, debugEnabled bool) *SearchV2 {
	fallback := search.NewSearch(logger, debugEnabled)
	return &SearchV2{
		logger:       logger,
		debugEnabled: debugEnabled,
		fallback:     &fallback,
	}
}

// FindSimpleSymbol searches for a simple symbol (non-access-path) in the workspace
func (s *SearchV2) FindSimpleSymbol(
	searchParams search_params.SearchParams,
	projState *project_state.ProjectState,
) search.SearchResult {

	symbolName := searchParams.Symbol()

	// Filter keywords
	if c3.IsLanguageKeyword(symbolName) {
		return search.NewSearchResultEmpty(searchParams.TrackTraversedModules())
	}

	s.debug(fmt.Sprintf("FindSimpleSymbol: %s", symbolName))

	finder := NewSymbolFinder()
	collector := NewModuleCollector(projState)

	// Handle case where DocId might not be present
	currentDoc := ""
	if docIdOpt := searchParams.DocId(); docIdOpt.IsSome() {
		currentDoc = docIdOpt.Get()
	}

	ctx := SearchContext{
		CurrentDoc:    currentDoc,
		CurrentModule: searchParams.ModulePathInCursor(),
		Position:      searchParams.SymbolPosition(),
	}

	// Get relevant modules in priority order
	modules := collector.CollectRelevantModules(ctx)

	// Try each module in order (linear flow!)
	for _, moduleScope := range modules {

		// Determine search mode
		searchMode := ModuleRoot
		if moduleScope.DocId == ctx.CurrentDoc {
			searchMode = LocalScope
		}

		// Simple search
		result := finder.FindInScope(
			symbolName,
			moduleScope.Module,
			searchMode,
			ctx.Position,
		)

		if result.IsSome() {
			searchResult := search.NewSearchResult(searchParams.TrackTraversedModules())
			searchResult.Set(result.Get())
			return searchResult
		}
	}

	// Last resort: maybe it's a module name?
	// Note: TrackTraversedModules returns map[string]int, not map[string]bool
	trackedModules := searchParams.TrackTraversedModules()
	if moduleSymbol := s.tryFindAsModuleName(symbolName, projState, trackedModules); moduleSymbol != nil {
		searchResult := search.NewSearchResult(trackedModules)
		searchResult.Set(moduleSymbol)
		return searchResult
	}

	return search.NewSearchResultEmpty(trackedModules)
}

// tryFindAsModuleName checks if the symbol name matches a module name
func (s *SearchV2) tryFindAsModuleName(
	symbolName string,
	projState *project_state.ProjectState,
	traversedModules map[string]int,
) *symbols.Module {

	for _, unitModules := range projState.GetAllUnitModules() {
		for _, module := range unitModules.Modules() {
			if module.GetName() == symbolName {
				// Check if this module was traversed during search
				if _, ok := traversedModules[symbolName]; ok {
					return module
				}
			}
		}
	}

	return nil
}

// ResolveAccessPath is the main entry point for resolving foo.bar.baz style paths
func (s *SearchV2) ResolveAccessPath(
	searchParams search_params.SearchParams,
	projState *project_state.ProjectState,
) search.SearchResult {

	accessPath := searchParams.GetFullAccessPath()
	if len(accessPath) == 0 {
		return search.NewSearchResultEmpty(searchParams.TrackTraversedModules())
	}

	resolver := NewTypeResolver(projState)
	finder := NewMemberFinder(projState, s)
	ctx := NewAccessContext()

	// Find the first symbol using FindSimpleSymbol
	docIdStr := ""
	docIdOpt := searchParams.DocId()
	if docIdOpt.IsSome() {
		docIdStr = docIdOpt.Get()
	}
	firstSearch := search_params.NewSearchParamsBuilder().
		WithSymbolWord(accessPath[0]).
		WithDocId(docIdStr).
		WithContextModuleName(searchParams.ModuleInCursor()).
		WithScopeMode(search_params.InScope).
		Build()

	// Use FindSimpleSymbol for initial lookup
	result := s.FindSimpleSymbol(firstSearch, projState)

	if result.IsNone() {
		return result
	}

	current := result.Get()

	// Walk through each segment of the access path
	for i := 1; i < len(accessPath); i++ {
		segment := accessPath[i]
		isLast := (i == len(accessPath)-1)

		// 1. Resolve to inspectable type
		var ok bool
		current, ctx, ok = resolver.ResolveToInspectable(current, ctx, false)
		if !ok {
			return search.NewSearchResultEmpty(searchParams.TrackTraversedModules())
		}

		// 2. Find the next member or method
		current, ok = finder.FindMemberOrMethod(
			current,
			segment.Text(),
			ctx,
			searchParams.DocId(),
			searchParams.ModuleInCursor(),
		)
		if !ok {
			return search.NewSearchResultEmpty(searchParams.TrackTraversedModules())
		}

		// 3. Update context after finding member (unless it's the last segment)
		if !isLast {
			ctx = ctx.AfterFindingMember(current)
		}
	}

	// Build result
	searchResult := search.NewSearchResult(searchParams.TrackTraversedModules())
	searchResult.Set(current)
	searchResult.SetMembersReadable(ctx.MembersReadable)
	searchResult.SetFromDistinct(ctx.FromDistinct)

	return searchResult
}

// FindSymbolDeclarationInWorkspace is the public entry point for symbol search
func (s *SearchV2) FindSymbolDeclarationInWorkspace(
	docId string,
	position symbols.Position,
	state *project_state.ProjectState,
) option.Option[symbols.Indexable] {

	doc := state.GetDocument(docId)
	searchParams := search_params.BuildSearchBySymbolUnderCursor(
		doc,
		*state.GetUnitModulesByDoc(doc.URI),
		position,
	)

	if c3.IsLanguageKeyword(searchParams.Symbol()) {
		return option.None[symbols.Indexable]()
	}

	// Check if this is an access path (foo.bar.baz)
	if searchParams.HasAccessPath() {
		result := s.ResolveAccessPath(searchParams, state)
		if result.IsSome() {
			return option.Some(result.Get())
		}
		return option.None[symbols.Indexable]()
	}

	// For single symbols, use FindSimpleSymbol
	result := s.FindSimpleSymbol(searchParams, state)
	if result.IsSome() {
		return option.Some(result.Get())
	}
	return option.None[symbols.Indexable]()
}

// BuildCompletionList delegates to the old search implementation for now
// TODO: Implement native completion support in SearchV2
func (s *SearchV2) BuildCompletionList(
	ctx context.CursorContext,
	state *project_state.ProjectState,
) []protocol.CompletionItem {
	return s.fallback.BuildCompletionList(ctx, state)
}

func (s *SearchV2) debug(message string) {
	if s.debugEnabled {
		s.logger.Debug(fmt.Sprintf("[V2] %s", message))
	}
}
