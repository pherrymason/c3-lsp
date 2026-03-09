package server

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentPrepareCallHierarchy_returnsFunctionItem(t *testing.T) {
	source := `module app;

fn void callee() {
}

fn void caller() {
	callee();
}`
	uri := protocol.DocumentUri("file:///tmp/call_hierarchy_prepare_test.c3")
	srv := buildRenameTestServer(uri, source)

	pos := byteIndexToLSPPosition(source, strings.Index(source, "callee() {")+1)
	items, err := srv.TextDocumentPrepareCallHierarchy(nil, &protocol.CallHierarchyPrepareParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     pos,
		},
	})
	if err != nil {
		t.Fatalf("unexpected prepareCallHierarchy error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one call hierarchy item, got %#v", items)
	}
	if items[0].Name != "callee" {
		t.Fatalf("expected callee item, got %#v", items[0])
	}
}

func TestCallHierarchyIncomingCalls_returnsCaller(t *testing.T) {
	source := `module app;

fn void callee() {
}

fn void caller() {
	callee();
}`
	uri := protocol.DocumentUri("file:///tmp/call_hierarchy_incoming_test.c3")
	srv := buildRenameTestServer(uri, source)

	calleePos := byteIndexToLSPPosition(source, strings.Index(source, "callee() {")+1)
	items, err := srv.TextDocumentPrepareCallHierarchy(nil, &protocol.CallHierarchyPrepareParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}, Position: calleePos},
	})
	if err != nil || len(items) == 0 {
		t.Fatalf("expected prepared call hierarchy item, err=%v items=%#v", err, items)
	}

	incoming, err := srv.CallHierarchyIncomingCalls(nil, &protocol.CallHierarchyIncomingCallsParams{Item: items[0]})
	if err != nil {
		t.Fatalf("unexpected incomingCalls error: %v", err)
	}
	if len(incoming) == 0 {
		t.Fatalf("expected incoming caller entries")
	}
	if incoming[0].From.Name != "caller" {
		t.Fatalf("expected caller as incoming source, got %#v", incoming[0])
	}
}

func TestCallHierarchyOutgoingCalls_returnsCallee(t *testing.T) {
	source := `module app;

fn void callee() {
}

fn void caller() {
	callee();
}`
	uri := protocol.DocumentUri("file:///tmp/call_hierarchy_outgoing_test.c3")
	srv := buildRenameTestServer(uri, source)

	callerPos := byteIndexToLSPPosition(source, strings.Index(source, "caller() {")+1)
	items, err := srv.TextDocumentPrepareCallHierarchy(nil, &protocol.CallHierarchyPrepareParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}, Position: callerPos},
	})
	if err != nil || len(items) == 0 {
		t.Fatalf("expected prepared call hierarchy item, err=%v items=%#v", err, items)
	}

	outgoing, err := srv.CallHierarchyOutgoingCalls(nil, &protocol.CallHierarchyOutgoingCallsParams{Item: items[0]})
	if err != nil {
		t.Fatalf("unexpected outgoingCalls error: %v", err)
	}
	if len(outgoing) == 0 {
		t.Fatalf("expected outgoing callee entries")
	}
	if outgoing[0].To.Name != "callee" {
		t.Fatalf("expected callee as outgoing target, got %#v", outgoing[0])
	}
}
