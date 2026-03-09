package server

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// projectRootCacheState groups the project-root resolution cache.
type projectRootCacheState struct {
	mu     sync.Mutex
	cache  map[string]string
	hits   atomic.Uint64
	misses atomic.Uint64
}

var projectMarkers = []string{"project.json", "c3lsp.json"}

func (s *Server) resolveProjectRootForURI(uri *protocol.DocumentUri) string {
	if uri != nil {
		if path, err := fs.UriToPath(string(*uri)); err == nil {
			if root := s.resolveProjectRootForPath(path); root != "" {
				return root
			}

			return fallbackRootPath(path, s.state.GetProjectRootURI())
		}
	}

	workspaceRoot := s.state.GetProjectRootURI()
	if workspaceRoot != "" {
		return workspaceRoot
	}

	return ""
}

func (s *Server) resolveProjectRootForPath(path string) string {
	canonicalPath := fs.GetCanonicalPath(path)
	if canonicalPath == "" {
		canonicalPath = path
	}

	s.rootCache.mu.Lock()
	if cached, ok := s.rootCache.cache[canonicalPath]; ok {
		s.rootCache.mu.Unlock()
		s.rootCache.hits.Add(1)
		return cached
	}
	s.rootCache.mu.Unlock()
	s.rootCache.misses.Add(1)

	root := findNearestProjectRoot(canonicalPath)

	s.rootCache.mu.Lock()
	if s.rootCache.cache == nil {
		s.rootCache.cache = make(map[string]string)
	}
	s.rootCache.cache[canonicalPath] = root
	s.rootCache.mu.Unlock()

	return root
}

func (s *Server) invalidateProjectRootCache() {
	if s == nil {
		return
	}

	s.rootCache.mu.Lock()
	if len(s.rootCache.cache) == 0 {
		s.rootCache.mu.Unlock()
		return
	}
	s.rootCache.cache = make(map[string]string)
	s.rootCache.mu.Unlock()
}

func (s *Server) projectRootCacheCounters() (uint64, uint64) {
	if s == nil {
		return 0, 0
	}

	hits := s.rootCache.hits.Load()
	misses := s.rootCache.misses.Load()
	return hits, misses
}

func findNearestProjectRoot(path string) string {
	if path == "" {
		return ""
	}

	dir := normalizeToDirectory(path)
	for {
		for _, marker := range projectMarkers {
			if fileExists(filepath.Join(dir, marker)) {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return ""
}

func isBuildableProjectRoot(path string) bool {
	if path == "" {
		return false
	}

	dir := normalizeToDirectory(path)
	return fileExists(filepath.Join(dir, "project.json"))
}

func (s *Server) configureProjectForRoot(root string) {
	s.configureProjectForRootWithContext(root, nil)
}

func (s *Server) configureProjectForRootWithContext(root string, context *glsp.Context) {
	root = fs.GetCanonicalPath(root)

	if root == "" || s.activeConfigRoot == root {
		return
	}

	s.options.C3 = cloneC3Opts(s.workspaceC3Options)
	s.workspaceDependencyDirs = loadDependencySearchPathsForRoot(root)
	if s.server != nil {
		s.loadServerConfigurationForWorkspace(root, context)
	}
	s.activeConfigRoot = root
}

func fallbackRootPath(path string, workspaceRoot string) string {
	if workspaceRoot != "" {
		return workspaceRoot
	}

	if path == "" {
		return ""
	}

	return normalizeToDirectory(path)
}

func normalizeToDirectory(path string) string {
	cleanPath := fs.GetCanonicalPath(path)
	if info, err := os.Stat(cleanPath); err == nil && !info.IsDir() {
		return filepath.Dir(cleanPath)
	}

	if filepath.Ext(cleanPath) != "" {
		return filepath.Dir(cleanPath)
	}

	return cleanPath
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
