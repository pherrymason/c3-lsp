package server

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/commonlog"
)

func TestIndexWorkspaceAtAsync_nonBlockingAndDeduplicated(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var callCount atomic.Int32

	srv := &Server{
		state: &state,
		idx: indexingCoordinator{
			indexed:  make(map[string]bool),
			indexing: make(map[string]bool),
		},
		workspaceIndexer: func(ctx context.Context, path string) {
			callCount.Add(1)
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-ctx.Done():
			}
		},
	}

	root := fs.GetCanonicalPath(t.TempDir())

	before := time.Now()
	srv.indexWorkspaceAtAsync(root)
	if time.Since(before) > 50*time.Millisecond {
		t.Fatalf("expected async indexing call to return quickly")
	}

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected async indexer to start")
	}

	if !srv.isRootIndexedOrIndexing(root) {
		t.Fatalf("expected root to be marked as indexing")
	}

	srv.indexWorkspaceAtAsync(root)
	if callCount.Load() != 1 {
		t.Fatalf("expected deduplicated indexing call, got %d", callCount.Load())
	}

	close(release)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.isRootIndexed(root) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !srv.isRootIndexed(root) {
		t.Fatalf("expected root to be marked indexed after async completion")
	}
}

func TestIndexWorkspaceAtAsync_timeoutCancelsIndexing(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	started := make(chan struct{}, 1)
	ctxDone := make(chan struct{})

	srv := &Server{
		state: &state,
		idx: indexingCoordinator{
			indexed:  make(map[string]bool),
			indexing: make(map[string]bool),
			cancels:  make(map[string]context.CancelFunc),
		},
		workspaceIndexer: func(ctx context.Context, path string) {
			select {
			case started <- struct{}{}:
			default:
			}
			// Block until the context is cancelled (by timeout).
			<-ctx.Done()
			close(ctxDone)
		},
	}

	// Set a very short timeout (10 ms).
	srv.options.Runtime.IndexTimeoutMs = 10
	srv.options.Runtime.IndexTimeout = 10 * time.Millisecond

	root := fs.GetCanonicalPath(t.TempDir())
	srv.indexWorkspaceAtAsync(root)

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected async indexer to start")
	}

	// The timeout should fire and cancel the context.
	select {
	case <-ctxDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected indexing context to be cancelled by timeout")
	}

	// After the timeout the root must not be marked as indexed.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !srv.isRootIndexedOrIndexing(root) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if srv.isRootIndexed(root) {
		t.Fatalf("timed-out indexing must not mark root as indexed")
	}
}

func TestIndexWorkspaceAtAsync_canCancelStaleJob(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var callCount atomic.Int32

	srv := &Server{
		state: &state,
		idx: indexingCoordinator{
			indexed:  make(map[string]bool),
			indexing: make(map[string]bool),
			cancels:  make(map[string]context.CancelFunc),
		},
		workspaceIndexer: func(ctx context.Context, path string) {
			callCount.Add(1)
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
			case <-ctx.Done():
			}
		},
	}

	root := fs.GetCanonicalPath(t.TempDir())
	srv.indexWorkspaceAtAsync(root)

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected async indexer to start")
	}

	srv.cancelRootIndexing(root)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !srv.isRootIndexedOrIndexing(root) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if srv.isRootIndexedOrIndexing(root) {
		t.Fatalf("expected root to stop indexing after cancellation")
	}
	if srv.isRootIndexed(root) {
		t.Fatalf("expected cancelled indexing to not mark root as indexed")
	}

	close(release)

	srv.indexWorkspaceAtAsync(root)
	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected indexing to be restartable after cancellation")
	}
}

func TestIndexWorkspaceAtAsync_generationGuard_ignores_stale_completion(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	allowFirstFinish := make(chan struct{})
	var callCount atomic.Int32
	var firstCtx context.Context
	var firstCtxMu sync.Mutex

	srv := &Server{
		state: &state,
		idx: indexingCoordinator{
			indexed:  make(map[string]bool),
			indexing: make(map[string]bool),
			cancels:  make(map[string]context.CancelFunc),
		},
		workspaceIndexer: func(ctx context.Context, path string) {
			current := callCount.Add(1)
			if current == 1 {
				firstCtxMu.Lock()
				firstCtx = ctx
				firstCtxMu.Unlock()
				<-ctx.Done()
				<-allowFirstFinish
				return
			}
		},
	}

	root := fs.GetCanonicalPath(t.TempDir())
	srv.indexWorkspaceAtAsync(root)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		firstCtxMu.Lock()
		ctxSet := firstCtx != nil
		firstCtxMu.Unlock()
		if ctxSet {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	firstCtxMu.Lock()
	if firstCtx == nil {
		firstCtxMu.Unlock()
		t.Fatalf("expected first indexing context to be captured")
	}
	firstCtxMu.Unlock()

	srv.cancelRootIndexing(root)
	srv.markRootIndexingStoppedForGeneration(root, 1)
	srv.indexWorkspaceAtAsync(root)

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.isRootIndexed(root) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !srv.isRootIndexed(root) {
		t.Fatalf("expected second generation to mark root indexed")
	}

	close(allowFirstFinish)
	time.Sleep(50 * time.Millisecond)

	if !srv.isRootIndexed(root) {
		t.Fatalf("expected stale generation completion to be ignored")
	}
}
