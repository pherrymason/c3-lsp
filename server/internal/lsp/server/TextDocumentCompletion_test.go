package server

import (
	stdctx "context"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentCompletion_usesCacheForRepeatedRequests(t *testing.T) {
	source := `module app;

struct Todo {
	int alpha;
}

fn void run() {
	Todo t = {};
	t.
}`
	uri := protocol.DocumentUri("file:///tmp/completion_cache_hit_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "t.")+2)

	first, err := srv.TextDocumentCompletion(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("first completion failed: %v", err)
	}
	firstItems := completionItemsFromResult(t, first)
	if !completionItemsContainLabel(firstItems, "alpha") {
		t.Fatalf("expected completion items to include struct field alpha")
	}

	if got := len(srv.completionCache); got != 1 {
		t.Fatalf("expected one cached completion entry after first call, got %d", got)
	}

	second, err := srv.TextDocumentCompletion(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("second completion failed: %v", err)
	}
	secondItems := completionItemsFromResult(t, second)

	if len(firstItems) != len(secondItems) {
		t.Fatalf("expected cached completion to preserve item count, got %d and %d", len(firstItems), len(secondItems))
	}
	if got := len(srv.completionCache); got != 1 {
		t.Fatalf("expected one cached completion entry after repeated call, got %d", got)
	}
}

func TestTextDocumentCompletion_cacheInvalidatesOnDocumentVersionChange(t *testing.T) {
	original := `module app;

struct Todo {
	int alpha;
}

fn void run() {
	Todo t = {};
	t.
}`
	updated := `module app;

struct Todo {
	int beta;
}

fn void run() {
	Todo t = {};
	t.
}`

	uri := protocol.DocumentUri("file:///tmp/completion_cache_version_change_test.c3")
	srv := buildRenameTestServer(uri, original)
	originalPos := byteIndexToLSPPosition(original, strings.Index(original, "t.")+2)

	first, err := srv.TextDocumentCompletion(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     originalPos,
		},
	})
	if err != nil {
		t.Fatalf("completion on original source failed: %v", err)
	}
	firstItems := completionItemsFromResult(t, first)
	if !completionItemsContainLabel(firstItems, "alpha") {
		t.Fatalf("expected completion to include alpha")
	}

	srv.state.UpdateDocument(uri, 2, []interface{}{protocol.TextDocumentContentChangeEventWhole{Text: updated}}, srv.parser)
	updatedPos := byteIndexToLSPPosition(updated, strings.Index(updated, "t.")+2)

	second, err := srv.TextDocumentCompletion(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     updatedPos,
		},
	})
	if err != nil {
		t.Fatalf("completion on updated source failed: %v", err)
	}
	secondItems := completionItemsFromResult(t, second)
	if completionItemsContainLabel(secondItems, "alpha") {
		t.Fatalf("expected stale symbol alpha to be absent after version bump")
	}
	if !completionItemsContainLabel(secondItems, "beta") {
		t.Fatalf("expected updated symbol beta to be present after version bump")
	}
}

func completionItemsFromResult(t *testing.T, result any) []completionItemWithLabelDetails {
	t.Helper()

	items, ok := result.([]completionItemWithLabelDetails)
	if !ok {
		t.Fatalf("expected []completionItemWithLabelDetails, got %T", result)
	}

	return items
}

func completionItemsContainLabel(items []completionItemWithLabelDetails, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}

	return false
}

func TestTextDocumentCompletion_returnsNilWhenRequestContextCancelled(t *testing.T) {
	source := `module app;

struct Todo {
	int alpha;
}

fn void run() {
	Todo t = {};
	t.
}`
	uri := protocol.DocumentUri("file:///tmp/completion_cancelled_request_test.c3")
	srv := buildRenameTestServer(uri, source)
	pos := byteIndexToLSPPosition(source, strings.Index(source, "t.")+2)

	requestCtx, cancel := stdctx.WithCancel(stdctx.Background())
	cancel()

	result, err := srv.textDocumentCompletionWithTrace(nil, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	}, "", requestCtx)
	if err != nil {
		t.Fatalf("expected nil error for cancelled request, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for cancelled request, got: %T", result)
	}
}
