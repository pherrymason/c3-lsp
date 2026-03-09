package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestWorkspaceExecuteCommand_unknownCommandReturnsError(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/exec_unknown.c3"), "module app;")

	_, err := srv.WorkspaceExecuteCommand(nil, &protocol.ExecuteCommandParams{Command: "c3lsp.unknown"})
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command message, got %v", err)
	}
}

func TestWorkspaceExecuteCommand_rejectsArgumentsForNoArgCommand(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/exec_args.c3"), "module app;")

	_, err := srv.WorkspaceExecuteCommand(nil, &protocol.ExecuteCommandParams{
		Command:   workspaceCommandReindexWorkspace,
		Arguments: []any{"unexpected"},
	})
	if err == nil {
		t.Fatalf("expected argument validation error")
	}
	if !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("expected argument validation message, got %v", err)
	}
}

func TestWorkspaceExecuteCommand_reindexWorkspaceSchedulesIndexing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	uri := protocol.DocumentUri("file:///tmp/exec_reindex.c3")
	srv := buildRenameTestServer(uri, "module app;")
	srv.state.SetProjectRootURI(root)

	indexed := make(chan string, 1)
	srv.workspaceIndexer = func(ctx context.Context, path string) {
		select {
		case indexed <- path:
		default:
		}
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	result, err := srv.WorkspaceExecuteCommand(ctx, &protocol.ExecuteCommandParams{Command: workspaceCommandReindexWorkspace})
	if err != nil {
		t.Fatalf("unexpected reindex error: %v", err)
	}

	select {
	case got := <-indexed:
		expectedRoot := normalizeIndexRoot(root)
		if got != expectedRoot {
			t.Fatalf("expected indexing root %q, got %q", expectedRoot, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected reindex command to schedule indexing")
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map command result, got %T", result)
	}
	if resultMap["command"] != workspaceCommandReindexWorkspace {
		t.Fatalf("unexpected command result payload: %#v", resultMap)
	}
}

func TestWorkspaceExecuteCommand_clearDiagnosticsCacheUsesApplyEditFlow(t *testing.T) {
	uri := protocol.DocumentUri("file:///tmp/exec_clear_diag.c3")
	srv := buildRenameTestServer(uri, "module app;")
	docID := utils.NormalizePath(uri)
	srv.state.SetDocumentDiagnostics(docID, []protocol.Diagnostic{{Message: "boom"}})

	publishCount := 0
	appliedCallCount := 0
	calledMethod := ""
	ctx := &glsp.Context{
		Notify: func(method string, params any) {
			if method == protocol.ServerTextDocumentPublishDiagnostics {
				publishCount++
			}
		},
		Call: func(method string, params any, result any) {
			calledMethod = method
			appliedCallCount++
			response, ok := result.(*protocol.ApplyWorkspaceEditResponse)
			if !ok {
				t.Fatalf("expected apply workspace edit response pointer, got %T", result)
			}
			response.Applied = true
		},
	}

	result, err := srv.WorkspaceExecuteCommand(ctx, &protocol.ExecuteCommandParams{Command: workspaceCommandClearDiagnostics})
	if err != nil {
		t.Fatalf("unexpected clear diagnostics error: %v", err)
	}
	if calledMethod != protocol.ServerWorkspaceApplyEdit {
		t.Fatalf("expected workspace/applyEdit call, got %q", calledMethod)
	}
	if appliedCallCount != 1 {
		t.Fatalf("expected one workspace/applyEdit call, got %d", appliedCallCount)
	}
	if publishCount == 0 {
		t.Fatalf("expected diagnostics clear publish")
	}
	if len(srv.state.GetDocumentDiagnostics()) != 0 {
		t.Fatalf("expected diagnostics cache to be cleared")
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map command result, got %T", result)
	}
	if resultMap["workspaceApplyEdit"] != true {
		t.Fatalf("expected apply-edit marker in command result, got %#v", resultMap)
	}
}

func TestApplyWorkspaceEdit_returnsErrorWhenClientRejectsEdit(t *testing.T) {
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/exec_apply_edit.c3"), "module app;")
	reason := "read-only"

	ctx := &glsp.Context{Call: func(method string, params any, result any) {
		response := result.(*protocol.ApplyWorkspaceEditResponse)
		response.Applied = false
		response.FailureReason = &reason
	}}

	_, err := srv.applyWorkspaceEdit(ctx, protocol.ApplyWorkspaceEditParams{Edit: *emptyWorkspaceEdit()})
	if err == nil {
		t.Fatalf("expected apply edit rejection error")
	}
	if !strings.Contains(err.Error(), reason) {
		t.Fatalf("expected rejection reason in error, got %v", err)
	}
}
