package server

import "github.com/tliron/glsp/protocol_3_16"

// completionCacheMaxEntries is the maximum number of entries kept in the
// completion cache.  Oldest entries are evicted first (FIFO).
const completionCacheMaxEntries = 256

type completionCacheKey struct {
	DocURI         string
	DocVersion     int32
	StateRevision  uint64
	Line           uint32
	Character      uint32
	TriggerKind    int
	TriggerChar    string
	SymbolAtCursor string
	SnippetSupport bool
}

func completionTriggerKey(params *protocol.CompletionParams) (int, string) {
	if params == nil || params.Context == nil {
		return 0, ""
	}

	kind := int(params.Context.TriggerKind)
	triggerCharacter := ""
	if params.Context.TriggerCharacter != nil {
		triggerCharacter = *params.Context.TriggerCharacter
	}

	return kind, triggerCharacter
}

func (h *Server) completionCacheGet(key completionCacheKey) ([]completionItemWithLabelDetails, bool) {
	h.completionCacheMu.Lock()
	defer h.completionCacheMu.Unlock()

	if h.completionCache == nil {
		return nil, false
	}

	items, ok := h.completionCache[key]
	if !ok {
		return nil, false
	}

	return cloneCompletionItemsWithLabelDetails(items), true
}

func (h *Server) completionCacheSet(key completionCacheKey, items []completionItemWithLabelDetails) {
	h.completionCacheMu.Lock()
	defer h.completionCacheMu.Unlock()

	if h.completionCache == nil {
		h.completionCache = make(map[completionCacheKey][]completionItemWithLabelDetails, completionCacheMaxEntries)
	}
	if h.completionCacheOrder == nil {
		h.completionCacheOrder = make([]completionCacheKey, 0, completionCacheMaxEntries)
	}

	if _, exists := h.completionCache[key]; !exists {
		h.completionCacheOrder = append(h.completionCacheOrder, key)
	}

	h.completionCache[key] = cloneCompletionItemsWithLabelDetails(items)

	// Evict oldest entries.  Copy remaining keys into a fresh slice with the
	// original capacity so the backing array does not grow without bound.
	for len(h.completionCacheOrder) > completionCacheMaxEntries {
		evict := h.completionCacheOrder[0]
		delete(h.completionCache, evict)
		remaining := h.completionCacheOrder[1:]
		fresh := make([]completionCacheKey, len(remaining), completionCacheMaxEntries)
		copy(fresh, remaining)
		h.completionCacheOrder = fresh
	}
}

func cloneCompletionItemsWithLabelDetails(items []completionItemWithLabelDetails) []completionItemWithLabelDetails {
	if len(items) == 0 {
		return []completionItemWithLabelDetails{}
	}

	clone := make([]completionItemWithLabelDetails, len(items))
	copy(clone, items)
	return clone
}
