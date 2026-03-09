package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentSelectionRange_returns_nested_ranges_for_symbol_position(t *testing.T) {
	source := `module app;

struct Fiber {
	int entry;
}

fn void run(Fiber* fiber) {
	fiber.entry = 1;
}`

	uri := protocol.DocumentUri("file:///tmp/selection_range_nested_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "fiber.entry"))
	ranges, err := srv.TextDocumentSelectionRange(nil, &protocol.SelectionRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Positions:    []protocol.Position{pos},
	})
	if err != nil {
		t.Fatalf("unexpected selectionRange error: %v", err)
	}
	if len(ranges) != 1 {
		t.Fatalf("expected one selection range result, got: %d", len(ranges))
	}

	if ranges[0].Parent == nil {
		t.Fatalf("expected nested selection range chain, got: %#v", ranges[0])
	}
}

func TestTextDocumentSelectionRange_preserves_positions_order(t *testing.T) {
	source := `module app;

fn void a() {
}

fn void b() {
}`

	uri := protocol.DocumentUri("file:///tmp/selection_range_positions_order_test.c3")
	srv := buildRenameTestServer(uri, source)

	posA := byteIndexToLSPPosition(source, strings.Index(source, "fn void a"))
	posB := byteIndexToLSPPosition(source, strings.Index(source, "fn void b"))
	ranges, err := srv.TextDocumentSelectionRange(nil, &protocol.SelectionRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Positions:    []protocol.Position{posA, posB},
	})
	if err != nil {
		t.Fatalf("unexpected selectionRange error: %v", err)
	}
	if len(ranges) != 2 {
		t.Fatalf("expected two selection ranges, got: %d", len(ranges))
	}

	if ranges[0].Range.Start.Line > ranges[1].Range.Start.Line {
		t.Fatalf("expected output order to match input positions, got: %#v", ranges)
	}
}
