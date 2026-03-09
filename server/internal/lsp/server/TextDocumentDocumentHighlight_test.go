package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDocumentHighlight_function_current_document_only(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/document_highlight_function_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/document_highlight_function_use_test.c3")

	declSource := `module app;

fn void parse_port() {
	parse_port();
}`

	useSource := `module app;

fn void run() {
	parse_port();
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{
		{uri: declURI, source: declSource},
		{uri: useURI, source: useSource},
	})

	pos := byteIndexToLSPPosition(declSource, strings.Index(declSource, "parse_port()"))
	highlights, err := srv.TextDocumentDocumentHighlight(nil, &protocol.DocumentHighlightParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: declURI},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected documentHighlight error: %v", err)
	}
	if len(highlights) != 2 {
		t.Fatalf("expected declaration + local usage highlights only, got: %d", len(highlights))
	}

	if highlights[0].Range.Start.Line != 2 || highlights[1].Range.Start.Line != 3 {
		t.Fatalf("expected sorted same-document highlights on lines 2 and 3, got: %#v", highlights)
	}
}

func TestTextDocumentDocumentHighlight_local_variable_scope_safe(t *testing.T) {
	source := `module app;

fn void a() {
	int value = 1;
	value = value + 1;
}

fn void b() {
	int value = 2;
	value = value + 1;
}`

	uri := protocol.DocumentUri("file:///tmp/document_highlight_local_variable_scope_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "value = 1"))
	highlights, err := srv.TextDocumentDocumentHighlight(nil, &protocol.DocumentHighlightParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected documentHighlight error: %v", err)
	}
	if len(highlights) != 3 {
		t.Fatalf("expected 3 highlights in first function scope, got: %d", len(highlights))
	}
}

func TestTextDocumentDocumentHighlight_cursor_inside_comment_returns_empty(t *testing.T) {
	source := `module app;
fn int bump(int n) {
	// bump should not resolve here
	return bump(n);
}`

	uri := protocol.DocumentUri("file:///tmp/document_highlight_comment_noop_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "bump should"))
	highlights, err := srv.TextDocumentDocumentHighlight(nil, &protocol.DocumentHighlightParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected documentHighlight error: %v", err)
	}
	if len(highlights) != 0 {
		t.Fatalf("expected no highlights for comment cursor, got: %#v", highlights)
	}
}
