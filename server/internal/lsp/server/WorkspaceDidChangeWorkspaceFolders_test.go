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
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestWorkspaceDidChangeWorkspaceFolders_addsBuildableRootAndSchedulesIndex(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	srv := NewServer(ServerOpts{
		C3:          c3c.C3Opts{Version: option.None[string](), Path: option.None[string](), StdlibPath: option.None[string](), CompileArgs: []string{}},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{C3FmtPath: option.None[string](), Config: option.None[string]()},
		LogFilepath: option.None[string](),
	}, "test-server", "test")
	srv.initialized.Store(true)

	started := make(chan struct{}, 1)
	srv.workspaceIndexer = func(ctx context.Context, path string) {
		select {
		case started <- struct{}{}:
		default:
		}
	}

	folderURI := protocol.DocumentUri(fs.ConvertPathToURI(root, option.None[string]()))
	err := srv.WorkspaceDidChangeWorkspaceFolders(nil, &protocol.DidChangeWorkspaceFoldersParams{
		Event: protocol.WorkspaceFoldersChangeEvent{Added: []protocol.WorkspaceFolder{{URI: folderURI, Name: "added-root"}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected indexing to be scheduled for added buildable root")
	}

	if got := srv.state.GetProjectRootURI(); got != fs.GetCanonicalPath(root) {
		t.Fatalf("expected project root to be set to added folder, got %q", got)
	}
}

func TestWorkspaceDidChangeWorkspaceFolders_removedRootCancelsIndexingAndClearsTracking(t *testing.T) {
	root := t.TempDir()

	srv := NewServer(ServerOpts{
		C3:          c3c.C3Opts{Version: option.None[string](), Path: option.None[string](), StdlibPath: option.None[string](), CompileArgs: []string{}},
		Diagnostics: DiagnosticsOpts{Enabled: false, Delay: 1},
		Formatting:  FormattingOpts{C3FmtPath: option.None[string](), Config: option.None[string]()},
		LogFilepath: option.None[string](),
	}, "test-server", "test")
	srv.initialized.Store(true)

	canonicalRoot := fs.GetCanonicalPath(root)
	srv.state.SetProjectRootURI(canonicalRoot)
	srv.activeConfigRoot = canonicalRoot

	started := make(chan struct{}, 1)
	cancelObserved := make(chan struct{}, 1)
	var startedCount atomic.Int32
	srv.workspaceIndexer = func(ctx context.Context, path string) {
		startedCount.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		select {
		case cancelObserved <- struct{}{}:
		default:
		}
	}

	srv.indexWorkspaceAtAsync(canonicalRoot)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected indexing to start before removal")
	}

	folderURI := protocol.DocumentUri(fs.ConvertPathToURI(canonicalRoot, option.None[string]()))
	err := srv.WorkspaceDidChangeWorkspaceFolders(nil, &protocol.DidChangeWorkspaceFoldersParams{
		Event: protocol.WorkspaceFoldersChangeEvent{Removed: []protocol.WorkspaceFolder{{URI: folderURI, Name: "removed-root"}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-cancelObserved:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected removed root indexing context to be cancelled")
	}

	if srv.isRootIndexedOrIndexing(canonicalRoot) {
		t.Fatalf("expected root tracking to be cleared after removal")
	}
	if got := srv.state.GetProjectRootURI(); got != "" {
		t.Fatalf("expected project root to be reset after removing active root, got %q", got)
	}
}
