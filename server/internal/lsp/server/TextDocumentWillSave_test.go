package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentWillSaveWaitUntil_DefaultReturnsNoEdits(t *testing.T) {
	uri := protocol.DocumentUri("file:///tmp/willsave_default.c3")
	srv := buildRenameTestServer(uri, "module app; fn void main() {}")

	edits, err := srv.TextDocumentWillSaveWaitUntil(nil, &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("expected no error by default, got: %v", err)
	}
	if len(edits) != 0 {
		t.Fatalf("expected no edits by default, got: %#v", edits)
	}
}

func TestTextDocumentWillSaveWaitUntil_EnabledUsesFormattingPath(t *testing.T) {
	uri := protocol.DocumentUri("file:///tmp/willsave_enabled.c3")
	srv := buildRenameTestServer(uri, "module app; fn void main() {}")
	srv.options.Formatting.WillSaveWaitUntil = true

	_, err := srv.TextDocumentWillSaveWaitUntil(nil, &protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err == nil {
		t.Fatalf("expected formatter configuration error when will-save-wait-until is enabled")
	}
}
