package server

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
)

// indexingTimeout returns the per-root indexing timeout from RuntimeOpts.
// A value of 0 means no timeout.
func (s *Server) indexingTimeout() time.Duration {
	return s.options.Runtime.IndexTimeout
}

type workspaceIndexState string

const (
	rootStateNotIndexed workspaceIndexState = "not_indexed"
	rootStateIndexing   workspaceIndexState = "indexing"
	rootStateIndexed    workspaceIndexState = "indexed"
	rootStateStale      workspaceIndexState = "stale"
)

func (s *Server) ensureIndexingStateMapsLocked() {
	if s.idx.indexed == nil {
		s.idx.indexed = make(map[string]bool)
	}
	if s.idx.indexing == nil {
		s.idx.indexing = make(map[string]bool)
	}
	if s.idx.cancels == nil {
		s.idx.cancels = make(map[string]context.CancelFunc)
	}
	if s.idx.rootStates == nil {
		s.idx.rootStates = make(map[string]workspaceIndexState)
	}
	if s.idx.generations == nil {
		s.idx.generations = make(map[string]uint64)
	}
}

func (s *Server) setRootState(root string, state workspaceIndexState) {
	root = normalizeIndexRoot(root)
	if root == "" {
		return
	}

	s.idx.rootStates[root] = state
	s.idx.indexed[root] = state == rootStateIndexed
	s.idx.indexing[root] = state == rootStateIndexing
}

func (s *Server) isRootIndexedOrIndexing(root string) bool {
	root = normalizeIndexRoot(root)
	if root == "" {
		return false
	}

	s.idx.mu.Lock()
	defer s.idx.mu.Unlock()
	s.ensureIndexingStateMapsLocked()

	state, ok := s.idx.rootStates[root]
	if !ok {
		return s.idx.indexed[root] || s.idx.indexing[root]
	}

	return state == rootStateIndexed || state == rootStateIndexing
}

func (s *Server) isRootIndexed(root string) bool {
	root = normalizeIndexRoot(root)
	if root == "" {
		return false
	}

	s.idx.mu.Lock()
	defer s.idx.mu.Unlock()
	s.ensureIndexingStateMapsLocked()

	state, ok := s.idx.rootStates[root]
	if !ok {
		return s.idx.indexed[root]
	}

	return state == rootStateIndexed
}

func (s *Server) rootState(root string) workspaceIndexState {
	root = normalizeIndexRoot(root)
	if root == "" {
		return rootStateNotIndexed
	}

	s.idx.mu.Lock()
	defer s.idx.mu.Unlock()
	s.ensureIndexingStateMapsLocked()

	state, ok := s.idx.rootStates[root]
	if !ok {
		if s.idx.indexing[root] {
			return rootStateIndexing
		}
		if s.idx.indexed[root] {
			return rootStateIndexed
		}
		return rootStateNotIndexed
	}

	return state
}

func (s *Server) indexWorkspaceAtAsync(path string) {
	s.indexWorkspaceAtAsyncWithProgress(path, nil)
}

func (s *Server) indexWorkspaceAtAsyncWithProgress(path string, lspContext *glsp.Context) {
	root := normalizeIndexRoot(path)
	if root == "" {
		return
	}

	s.idx.mu.Lock()
	s.ensureIndexingStateMapsLocked()
	if s.idx.rootStates[root] == rootStateIndexed || s.idx.rootStates[root] == rootStateIndexing {
		s.idx.mu.Unlock()
		return
	}

	generation := s.idx.generations[root] + 1
	s.idx.generations[root] = generation

	// Apply a timeout so that a stuck indexer (e.g. c3c hanging on a huge
	// workspace) cannot block the server indefinitely.
	// Default: 5 minutes.  Override: C3LSP_INDEX_TIMEOUT_MS=0 → no timeout.
	timeout := s.indexingTimeout()
	var ctx context.Context
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	s.setRootState(root, rootStateIndexing)
	s.idx.cancels[root] = cancel
	s.idx.mu.Unlock()

	go func() {
		defer cancel() // release resources even on early return

		completed := false
		if s.workspaceIndexer != nil {
			s.workspaceIndexer(ctx, root)
			select {
			case <-ctx.Done():
				completed = false
			default:
				completed = true
			}
		} else {
			completed = s.indexWorkspaceAtWithContextAndProgress(ctx, root, lspContext)
		}

		if !completed {
			// Distinguish between timeout and cancellation for diagnostics.
			if ctx.Err() == context.DeadlineExceeded {
				if s.server != nil && s.server.Log != nil {
					s.server.Log.Warning("indexing timed out", "root", root, "after", timeout.String())
				}
			}
		}

		if completed {
			s.markRootIndexedForGeneration(root, generation)
		} else {
			s.markRootIndexingStoppedForGeneration(root, generation)
		}
	}()
}

func (s *Server) cancelRootIndexing(root string) {
	root = normalizeIndexRoot(root)
	if root == "" {
		return
	}

	s.idx.mu.Lock()
	s.ensureIndexingStateMapsLocked()
	cancel, ok := s.idx.cancels[root]
	s.idx.mu.Unlock()
	if ok {
		cancel()
	}
}

func (s *Server) clearRootTracking(root string) {
	root = normalizeIndexRoot(root)
	if root == "" {
		return
	}

	s.idx.mu.Lock()
	defer s.idx.mu.Unlock()
	s.ensureIndexingStateMapsLocked()

	delete(s.idx.indexed, root)
	delete(s.idx.indexing, root)
	delete(s.idx.cancels, root)
	delete(s.idx.rootStates, root)
	delete(s.idx.generations, root)
}

func (s *Server) markRootIndexed(root string) {
	s.markRootIndexedForGeneration(root, 0)
}

func (s *Server) markRootIndexedForGeneration(root string, generation uint64) {
	root = normalizeIndexRoot(root)
	if root == "" {
		return
	}

	s.idx.mu.Lock()
	s.ensureIndexingStateMapsLocked()
	if generation > 0 {
		currentGeneration := s.idx.generations[root]
		if currentGeneration != generation {
			s.idx.mu.Unlock()
			return
		}
	}

	delete(s.idx.indexing, root)
	delete(s.idx.cancels, root)
	s.setRootState(root, rootStateIndexed)
	s.idx.mu.Unlock()
}

func (s *Server) markRootIndexingStoppedForGeneration(root string, generation uint64) {
	root = normalizeIndexRoot(root)
	if root == "" {
		return
	}

	s.idx.mu.Lock()
	s.ensureIndexingStateMapsLocked()
	if generation > 0 {
		currentGeneration := s.idx.generations[root]
		if currentGeneration != generation {
			s.idx.mu.Unlock()
			return
		}
	}

	delete(s.idx.indexing, root)
	delete(s.idx.cancels, root)
	if s.idx.rootStates[root] == rootStateIndexed {
		s.setRootState(root, rootStateStale)
	} else {
		s.setRootState(root, rootStateNotIndexed)
	}
	s.idx.mu.Unlock()
}

func normalizeIndexRoot(root string) string {
	root = fs.GetCanonicalPath(root)
	if root == "" {
		return ""
	}

	if runtime.GOOS == "darwin" && strings.HasPrefix(root, "/private/") {
		return strings.TrimPrefix(root, "/private")
	}

	return root
}
