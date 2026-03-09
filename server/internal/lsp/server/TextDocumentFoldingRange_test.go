package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentFoldingRange_returns_symbol_and_comment_ranges(t *testing.T) {
	source := `module app;

<*
 @param n : "value"
*>
struct Fiber {
	int entry;
}

fn void run() {
	if (true) {
		io::printn("x");
	}
}`

	uri := protocol.DocumentUri("file:///tmp/folding_ranges_symbols_comments_test.c3")
	srv := buildRenameTestServer(uri, source)

	ranges, err := srv.TextDocumentFoldingRange(nil, &protocol.FoldingRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("unexpected foldingRange error: %v", err)
	}
	if len(ranges) == 0 {
		t.Fatalf("expected folding ranges")
	}

	if !hasFoldingRange(ranges, 2, 4) {
		t.Fatalf("expected doc comment folding range [2,4], got: %#v", ranges)
	}
	if !hasFoldingRange(ranges, 5, 7) {
		t.Fatalf("expected struct folding range [5,7], got: %#v", ranges)
	}
	if !hasFoldingRange(ranges, 9, 13) {
		t.Fatalf("expected function folding range [9,13], got: %#v", ranges)
	}
}

func TestTextDocumentFoldingRange_nil_params_returns_empty(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/folding_nil_test.c3"), "module app;")
	ranges, err := srv.TextDocumentFoldingRange(nil, nil)
	if err != nil {
		t.Fatalf("unexpected foldingRange error: %v", err)
	}
	if len(ranges) != 0 {
		t.Fatalf("expected no ranges for nil params, got: %#v", ranges)
	}
}

func hasFoldingRange(ranges []protocol.FoldingRange, start int, end int) bool {
	for _, r := range ranges {
		if int(r.StartLine) == start && int(r.EndLine) == end {
			return true
		}
	}

	return false
}
