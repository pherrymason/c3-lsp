package server

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/c3c"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestWorkspaceDidChangeWatchedFiles_reloads_c3lsp_json(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "c3lsp.json"), []byte(`{"debug":false}`), 0o644); err != nil {
		t.Fatalf("failed to write c3lsp.json: %v", err)
	}

	srv := NewServer(ServerOpts{
		C3:          c3c.C3Opts{Version: option.None[string](), Path: option.None[string](), StdlibPath: option.None[string](), CompileArgs: []string{}},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{C3FmtPath: option.None[string](), Config: option.None[string]()},
		LogFilepath: option.None[string](),
	}, "test-server", "test")

	srv.initialized.Store(true)
	srv.state.SetProjectRootURI(root)
	srv.configureProjectForRoot(root)

	if srv.options.Debug {
		t.Fatalf("expected debug false initially")
	}

	if err := os.WriteFile(filepath.Join(root, "c3lsp.json"), []byte(`{"debug":true}`), 0o644); err != nil {
		t.Fatalf("failed to rewrite c3lsp.json: %v", err)
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	err := srv.WorkspaceDidChangeWatchedFiles(ctx, &protocol.DidChangeWatchedFilesParams{Changes: []protocol.FileEvent{{
		URI:  protocol.DocumentUri(fs.ConvertPathToURI(filepath.Join(root, "c3lsp.json"), option.None[string]())),
		Type: protocol.FileChangeTypeChanged,
	}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !srv.options.Debug {
		t.Fatalf("expected debug true after c3lsp.json reload")
	}
}

func TestTextDocumentDidSave_reloads_c3lsp_json(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "c3lsp.json"), []byte(`{"debug":false}`), 0o644); err != nil {
		t.Fatalf("failed to write c3lsp.json: %v", err)
	}

	srv := NewServer(ServerOpts{
		C3:          c3c.C3Opts{Version: option.None[string](), Path: option.None[string](), StdlibPath: option.None[string](), CompileArgs: []string{}},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{C3FmtPath: option.None[string](), Config: option.None[string]()},
		LogFilepath: option.None[string](),
	}, "test-server", "test")

	srv.initialized.Store(true)
	srv.state.SetProjectRootURI(root)
	srv.configureProjectForRoot(root)

	if srv.options.Debug {
		t.Fatalf("expected debug false initially")
	}

	if err := os.WriteFile(filepath.Join(root, "c3lsp.json"), []byte(`{"debug":true}`), 0o644); err != nil {
		t.Fatalf("failed to rewrite c3lsp.json: %v", err)
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	err := srv.TextDocumentDidSave(ctx, &protocol.DidSaveTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{
		URI: protocol.DocumentUri(fs.ConvertPathToURI(filepath.Join(root, "c3lsp.json"), option.None[string]())),
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !srv.options.Debug {
		t.Fatalf("expected debug true after c3lsp.json save reload")
	}
}

func TestWorkspaceDidChangeWatchedFiles_reindexes_and_refreshes_changed_source(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}
	sourcePath := filepath.Join(root, "main.c3")
	if err := os.WriteFile(sourcePath, []byte("module app; fn void main() {}"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	srv := NewServer(ServerOpts{
		C3:          c3c.C3Opts{Version: option.None[string](), Path: option.None[string](), StdlibPath: option.None[string](), CompileArgs: []string{}},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{C3FmtPath: option.None[string](), Config: option.None[string]()},
		LogFilepath: option.None[string](),
	}, "test-server", "test")

	srv.initialized.Store(true)
	srv.state.SetProjectRootURI(root)

	indexed := make(chan struct{}, 1)
	var indexCount atomic.Int32
	srv.workspaceIndexer = func(ctx context.Context, path string) {
		indexCount.Add(1)
		select {
		case indexed <- struct{}{}:
		default:
		}
	}

	if err := os.WriteFile(sourcePath, []byte("module app; fn void main() { int x = 1; }"), 0o644); err != nil {
		t.Fatalf("failed to update source file: %v", err)
	}

	ctx := &glsp.Context{Notify: func(string, any) {}}
	err := srv.WorkspaceDidChangeWatchedFiles(ctx, &protocol.DidChangeWatchedFilesParams{Changes: []protocol.FileEvent{{
		URI:  protocol.DocumentUri(fs.ConvertPathToURI(sourcePath, option.None[string]())),
		Type: protocol.FileChangeTypeChanged,
	}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-indexed:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected changed source to trigger reindex")
	}

	if indexCount.Load() == 0 {
		t.Fatalf("expected at least one indexing call")
	}

	doc := srv.state.GetDocument(fs.GetCanonicalPath(sourcePath))
	if doc == nil {
		t.Fatalf("expected changed source document to be refreshed in project state")
	}
	if doc.SourceCode.Text != "module app; fn void main() { int x = 1; }" {
		t.Fatalf("expected refreshed source text, got %q", doc.SourceCode.Text)
	}
}
