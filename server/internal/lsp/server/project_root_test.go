package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFindNearestProjectRoot_PrefersNearestMarker(t *testing.T) {
	tmpDir := t.TempDir()

	root := filepath.Join(tmpDir, "workspace")
	projectDir := filepath.Join(root, "project-a")
	srcDir := filepath.Join(projectDir, "src")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "c3lsp.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create workspace c3lsp.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	docPath := filepath.Join(srcDir, "main.c3")
	if got := findNearestProjectRoot(docPath); got != projectDir {
		t.Fatalf("findNearestProjectRoot(%q) = %q, expected %q", docPath, got, projectDir)
	}
}

func TestFindNearestProjectRoot_FallsBackToC3LspJson(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	srcDir := filepath.Join(workspaceDir, "nested", "src")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspaceDir, "c3lsp.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create c3lsp.json: %v", err)
	}

	docPath := filepath.Join(srcDir, "main.c3")
	if got := findNearestProjectRoot(docPath); got != workspaceDir {
		t.Fatalf("findNearestProjectRoot(%q) = %q, expected %q", docPath, got, workspaceDir)
	}
}

func TestIsBuildableProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()

	if isBuildableProjectRoot(tmpDir) {
		t.Fatalf("isBuildableProjectRoot(%q) = true, expected false", tmpDir)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	if !isBuildableProjectRoot(tmpDir) {
		t.Fatalf("isBuildableProjectRoot(%q) = false, expected true", tmpDir)
	}
}

func TestResolveProjectRootForURI_usesCacheUntilInvalidated(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceDir := filepath.Join(tmpDir, "workspace")
	projectDir := filepath.Join(workspaceDir, "nested")
	srcDir := filepath.Join(projectDir, "src")

	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	if err := os.WriteFile(filepath.Join(workspaceDir, "c3lsp.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create workspace c3lsp.json: %v", err)
	}

	docPath := filepath.Join(srcDir, "main.c3")
	docURI := protocol.DocumentUri(fs.ConvertPathToURI(docPath, option.None[string]()))

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	state.SetProjectRootURI(workspaceDir)

	srv := &Server{state: &state, rootCache: projectRootCacheState{cache: make(map[string]string)}}

	if got := srv.resolveProjectRootForURI(&docURI); got != workspaceDir {
		t.Fatalf("resolveProjectRootForURI(%q) = %q, expected %q", docURI, got, workspaceDir)
	}

	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create nested project.json: %v", err)
	}

	if got := srv.resolveProjectRootForURI(&docURI); got != workspaceDir {
		t.Fatalf("expected cached project root %q before invalidation, got %q", workspaceDir, got)
	}

	srv.invalidateProjectRootCache()
	if got := srv.resolveProjectRootForURI(&docURI); got != projectDir {
		t.Fatalf("expected nested project root %q after invalidation, got %q", projectDir, got)
	}
}
