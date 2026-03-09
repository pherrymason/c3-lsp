package search

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

type symbolLookupCache struct {
	mu      sync.RWMutex
	entries map[string]option.Option[symbols.Indexable]
}

type searchParamsCache struct {
	mu      sync.RWMutex
	entries map[string]search_params.SearchParams
}

func newSearchParamsCache() *searchParamsCache {
	return &searchParamsCache{entries: make(map[string]search_params.SearchParams)}
}

func (c *searchParamsCache) get(key string) (search_params.SearchParams, bool) {
	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return search_params.SearchParams{}, false
	}

	return search_params.CopyWithFreshTracking(v), true
}

func (c *searchParamsCache) set(key string, value search_params.SearchParams) {
	c.mu.Lock()
	if len(c.entries) > 2048 {
		c.entries = make(map[string]search_params.SearchParams)
	}
	c.entries[key] = value
	c.mu.Unlock()
}

type parentResolutionCache struct {
	mu      sync.RWMutex
	entries map[string]option.Option[symbols.Indexable]
}

func newParentResolutionCache() *parentResolutionCache {
	return &parentResolutionCache{entries: make(map[string]option.Option[symbols.Indexable])}
}

func (c *parentResolutionCache) get(key string) (option.Option[symbols.Indexable], bool) {
	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *parentResolutionCache) set(key string, value option.Option[symbols.Indexable]) {
	c.mu.Lock()
	if len(c.entries) > 2048 {
		c.entries = make(map[string]option.Option[symbols.Indexable])
	}
	c.entries[key] = value
	c.mu.Unlock()
}

func newSymbolLookupCache() *symbolLookupCache {
	return &symbolLookupCache{entries: make(map[string]option.Option[symbols.Indexable])}
}

func (c *symbolLookupCache) get(key string) (option.Option[symbols.Indexable], bool) {
	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	return v, ok
}

func (c *symbolLookupCache) set(key string, value option.Option[symbols.Indexable]) {
	c.mu.Lock()
	if len(c.entries) > 2048 {
		c.entries = make(map[string]option.Option[symbols.Indexable])
	}
	c.entries[key] = value
	c.mu.Unlock()
}

func buildSymbolLookupCacheKey(docId string, doc *document.Document, position symbols.Position, stateRevision uint64) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(doc.SourceCode.Text))
	docHash := h.Sum64()

	return fmt.Sprintf("decl|%s|%d|%d|%d|%d", docId, stateRevision, position.Line, position.Character, docHash)
}

func buildSearchParamsCacheKey(docId string, doc *document.Document, position symbols.Position, stateRevision uint64) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(doc.SourceCode.Text))
	docHash := h.Sum64()

	return fmt.Sprintf("sp|%s|%d|%d|%d|%d", docId, stateRevision, position.Line, position.Character, docHash)
}

func buildParentResolutionCacheKey(searchParams search_params.SearchParams, stateRevision uint64) string {
	docId := ""
	docOpt := searchParams.DocId()
	if docOpt.IsSome() {
		docId = docOpt.Get()
	}

	limitModule := ""
	if searchParams.LimitsSearchInModule() {
		limitModule = searchParams.LimitToModule().Get().GetName()
	}

	return fmt.Sprintf(
		"parent|%d|%s|%s|%d|%s|%s|%d|%d",
		stateRevision,
		docId,
		searchParams.ModuleInCursor(),
		searchParams.ScopeMode(),
		limitModule,
		searchParams.GetFullQualifiedName(),
		searchParams.SymbolPosition().Line,
		searchParams.SymbolPosition().Character,
	)
}
