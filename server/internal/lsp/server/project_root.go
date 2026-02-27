package server

import (
	"os"
	"path/filepath"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var projectMarkers = []string{"project.json", "c3lsp.json"}

func (s *Server) resolveProjectRootForURI(uri *protocol.DocumentUri) string {
	if uri != nil {
		if path, err := fs.UriToPath(string(*uri)); err == nil {
			if root := findNearestProjectRoot(path); root != "" {
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
	root = fs.GetCanonicalPath(root)

	if root == "" || s.activeConfigRoot == root {
		return
	}

	s.options.C3 = cloneC3Opts(s.workspaceC3Options)
	if s.server != nil {
		s.loadServerConfigurationForWorkspace(root)
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
