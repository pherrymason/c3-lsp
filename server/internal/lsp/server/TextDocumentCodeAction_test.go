package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentCodeAction_returnsDeterministicEmptyList(t *testing.T) {
	uri := protocol.DocumentUri("file:///tmp/code_action_test.c3")
	srv := buildRenameTestServer(uri, "module app;")

	result, err := srv.TextDocumentCodeAction(nil, &protocol.CodeActionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}})
	if err != nil {
		t.Fatalf("unexpected codeAction error: %v", err)
	}

	actions, ok := result.([]protocol.CodeAction)
	if !ok {
		t.Fatalf("expected []CodeAction result, got %T", result)
	}
	if len(actions) != 0 {
		t.Fatalf("expected empty code action list, got %#v", actions)
	}
}

func TestCodeActionResolve_returnsCopyOfInput(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/code_action_resolve_test.c3"), "module app;")
	kind := protocol.CodeActionKindQuickFix
	input := &protocol.CodeAction{Title: "Fix", Kind: &kind}

	resolved, err := srv.CodeActionResolve(nil, input)
	if err != nil {
		t.Fatalf("unexpected codeAction/resolve error: %v", err)
	}
	if resolved == nil || resolved.Title != "Fix" || resolved.Kind == nil || *resolved.Kind != kind {
		t.Fatalf("expected resolved code action to preserve input, got %#v", resolved)
	}
}
