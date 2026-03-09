package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentMoniker_returnsExportMonikerForFunctionSymbol(t *testing.T) {
	source := `module app;

fn void run() {
	int value = 1;
	value = value + 1;
}`
	uri := protocol.DocumentUri("file:///tmp/moniker_local_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "run()")+1)
	result, err := srv.TextDocumentMoniker(nil, &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected moniker error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected single moniker, got %#v", result)
	}
	if result[0].Kind == nil || *result[0].Kind != protocol.MonikerKindExport {
		t.Fatalf("expected export moniker kind, got %#v", result[0])
	}
	if result[0].Unique != protocol.UniquenessLevelProject {
		t.Fatalf("expected project uniqueness, got %#v", result[0])
	}
}

func TestTextDocumentMoniker_returnsNilWhenNoSymbolFound(t *testing.T) {
	source := `module app;
fn void run() {
	// comment only
}`
	uri := protocol.DocumentUri("file:///tmp/moniker_none_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "comment only"))
	result, err := srv.TextDocumentMoniker(nil, &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected moniker error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil moniker result, got %#v", result)
	}
}

func TestTextDocumentMoniker_returnsExportMonikerForModuleTarget(t *testing.T) {
	source := `module app::net;

fn void run() {}
`
	uri := protocol.DocumentUri("file:///tmp/moniker_module_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "app::net"))
	result, err := srv.TextDocumentMoniker(nil, &protocol.MonikerParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected moniker error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected single moniker, got %#v", result)
	}
	if result[0].Kind == nil || *result[0].Kind != protocol.MonikerKindExport {
		t.Fatalf("expected export moniker kind, got %#v", result[0])
	}
	if !strings.Contains(result[0].Identifier, "module:app") {
		t.Fatalf("expected module moniker identifier, got %#v", result[0])
	}
}
