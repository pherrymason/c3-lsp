package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentLinkedEditingRange_returnsRangesForLocalVariable(t *testing.T) {
	source := `module app;

fn void run() {
	int value = 1;
	value = value + 1;
}`
	uri := protocol.DocumentUri("file:///tmp/linked_edit_value.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "value = 1"))
	result, err := srv.TextDocumentLinkedEditingRange(nil, &protocol.LinkedEditingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected linked editing error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected linked editing result")
	}
	if len(result.Ranges) < 2 {
		t.Fatalf("expected at least declaration and usage ranges, got %#v", result.Ranges)
	}
	if result.WordPattern == nil || *result.WordPattern == "" {
		t.Fatalf("expected non-empty linked editing word pattern")
	}
}

func TestTextDocumentLinkedEditingRange_returnsNilForCommentCursor(t *testing.T) {
	source := `module app;
fn void run() {
	// value in comment
	int value = 1;
}`
	uri := protocol.DocumentUri("file:///tmp/linked_edit_comment.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "value in comment"))
	result, err := srv.TextDocumentLinkedEditingRange(nil, &protocol.LinkedEditingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected linked editing error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for comment cursor, got %#v", result)
	}
}
