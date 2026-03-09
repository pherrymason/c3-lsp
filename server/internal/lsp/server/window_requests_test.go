package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRequestWindowShowMessage_sendsPayloadAndReturnsResponse(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/window_show_message.c3"), "module app;")

	ctx := &glsp.Context{Call: func(method string, params any, result any) {
		if method != protocol.ServerWindowShowMessageRequest {
			t.Fatalf("expected %q, got %q", protocol.ServerWindowShowMessageRequest, method)
		}

		request, ok := params.(protocol.ShowMessageRequestParams)
		if !ok {
			t.Fatalf("expected show message request params, got %T", params)
		}
		if request.Message != "message" {
			t.Fatalf("unexpected message payload: %#v", request)
		}

		out, ok := result.(**protocol.MessageActionItem)
		if !ok {
			t.Fatalf("expected message action item result pointer, got %T", result)
		}
		*out = &protocol.MessageActionItem{Title: "Open"}
	}}

	action, err := srv.requestWindowShowMessage(ctx, protocol.MessageTypeInfo, "message", []protocol.MessageActionItem{{Title: "Open"}})
	if err != nil {
		t.Fatalf("unexpected show message request error: %v", err)
	}
	if action == nil || action.Title != "Open" {
		t.Fatalf("expected selected action, got %#v", action)
	}
}

func TestRequestWindowShowDocument_sendsPayloadAndReturnsResult(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/window_show_doc.c3"), "module app;")

	called := false
	ctx := &glsp.Context{Call: func(method string, params any, result any) {
		called = true
		if method != protocol.ServerWindowShowDocument {
			t.Fatalf("expected %q, got %q", protocol.ServerWindowShowDocument, method)
		}

		request, ok := params.(protocol.ShowDocumentParams)
		if !ok {
			t.Fatalf("expected show document params, got %T", params)
		}
		if string(request.URI) != "file:///tmp/project.json" {
			t.Fatalf("unexpected document uri: %q", request.URI)
		}

		out, ok := result.(*protocol.ShowDocumentResult)
		if !ok {
			t.Fatalf("expected show document result pointer, got %T", result)
		}
		out.Success = true
	}}

	result, err := srv.requestWindowShowDocument(ctx, protocol.ShowDocumentParams{URI: protocol.URI("file:///tmp/project.json")})
	if err != nil {
		t.Fatalf("unexpected show document error: %v", err)
	}
	if !called || result == nil || !result.Success {
		t.Fatalf("expected successful show document request, got called=%t result=%#v", called, result)
	}
}

func TestOfferProjectConfigOpen_opensDocumentWhenActionSelected(t *testing.T) {
	root := t.TempDir()
	projectFile := filepath.Join(root, "project.json")
	if err := os.WriteFile(projectFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/window_offer.c3"), "module app;")

	methods := []string{}
	ctx := &glsp.Context{Call: func(method string, params any, result any) {
		methods = append(methods, method)
		switch method {
		case string(protocol.ServerWindowShowMessageRequest):
			out := result.(**protocol.MessageActionItem)
			*out = &protocol.MessageActionItem{Title: openProjectConfigActionTitle}
		case string(protocol.ServerWindowShowDocument):
			request := params.(protocol.ShowDocumentParams)
			if request.URI == "" {
				t.Fatalf("expected show document uri to be set")
			}
			out := result.(*protocol.ShowDocumentResult)
			out.Success = true
		}
	}}

	srv.offerProjectConfigOpen(ctx, root)

	if len(methods) != 2 {
		t.Fatalf("expected showMessageRequest + showDocument calls, got %v", methods)
	}
	if methods[0] != string(protocol.ServerWindowShowMessageRequest) || methods[1] != string(protocol.ServerWindowShowDocument) {
		t.Fatalf("unexpected method call order: %v", methods)
	}
}
