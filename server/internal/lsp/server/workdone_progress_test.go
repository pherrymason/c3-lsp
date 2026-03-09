package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRequestWindowWorkDoneProgressCreate_callsClient(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/progress_create.c3"), "module app;")

	called := false
	ctx := &glsp.Context{Call: func(method string, params any, result any) {
		called = true
		if method != protocol.ServerWindowWorkDoneProgressCreate {
			t.Fatalf("expected %q, got %q", protocol.ServerWindowWorkDoneProgressCreate, method)
		}
		request, ok := params.(map[string]any)
		if !ok {
			t.Fatalf("expected map payload, got %T", params)
		}
		if request["token"] == nil {
			t.Fatalf("expected non-nil progress token")
		}
	}}

	token := protocol.ProgressToken{Value: "idx-1"}
	if err := srv.requestWindowWorkDoneProgressCreate(ctx, token); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if !called {
		t.Fatalf("expected workDoneProgress/create request")
	}
}

func TestWindowWorkDoneProgressCancel_marksTokenCanceled(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/progress_cancel.c3"), "module app;")
	token := protocol.ProgressToken{Value: "idx-2"}
	srv.markWorkDoneProgressActive(token)

	if err := srv.WindowWorkDoneProgressCancel(nil, &protocol.WorkDoneProgressCancelParams{Token: token}); err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if !srv.workDoneProgressWasCanceled(token) {
		t.Fatalf("expected token to be marked canceled")
	}
}

func TestBeginReportEndWorkDoneProgress_notifiesClient(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/progress_notify.c3"), "module app;")
	var caps protocol.ClientCapabilities
	if err := json.Unmarshal([]byte(`{"window":{"workDoneProgress":true}}`), &caps); err != nil {
		t.Fatalf("failed to build client capabilities: %v", err)
	}
	srv.clientCapabilities = caps

	created := false
	progressEvents := 0
	ctx := &glsp.Context{
		Call: func(method string, params any, result any) {
			if method == protocol.ServerWindowWorkDoneProgressCreate {
				created = true
			}
		},
		Notify: func(method string, params any) {
			if method == protocol.MethodProgress {
				progressEvents++
			}
		},
	}

	token, ok := srv.beginWorkDoneProgress(ctx, "C3 stdlib", "Loading", false)
	if !ok {
		t.Fatalf("expected progress begin to succeed")
	}

	srv.reportWorkDoneProgress(ctx, token, "Building", nil)
	srv.endWorkDoneProgress(ctx, token, "Done")

	if !created {
		t.Fatalf("expected window/workDoneProgress/create call")
	}
	if progressEvents != 3 {
		t.Fatalf("expected 3 progress events, got %d", progressEvents)
	}
}

func TestIndexWorkspaceAtWithContextAndProgress_reportsProgress(t *testing.T) {
	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "project.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, "main.c3"), []byte("module app;\nfn void main(){}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.c3: %v", err)
	}

	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/progress_index.c3"), "module app;")
	var caps protocol.ClientCapabilities
	if err := json.Unmarshal([]byte(`{"window":{"workDoneProgress":true}}`), &caps); err != nil {
		t.Fatalf("failed to build client capabilities: %v", err)
	}
	srv.clientCapabilities = caps

	created := false
	progressEvents := 0
	ctx := &glsp.Context{
		Call: func(method string, params any, result any) {
			if method == protocol.ServerWindowWorkDoneProgressCreate {
				created = true
			}
		},
		Notify: func(method string, params any) {
			if method == protocol.MethodProgress {
				progressEvents++
			}
		},
	}

	if ok := srv.indexWorkspaceAtWithContextAndProgress(context.Background(), workspaceRoot, ctx); !ok {
		t.Fatalf("expected workspace indexing to complete")
	}

	if !created {
		t.Fatalf("expected window/workDoneProgress/create call")
	}
	if progressEvents < 2 {
		t.Fatalf("expected progress notifications during indexing, got %d", progressEvents)
	}
}
