package server

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func writeC3LArchive(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive %s: %v", archivePath, err)
	}

	zw := zip.NewWriter(f)
	for name, content := range files {
		entry, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create archive entry %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write archive entry %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close archive writer %s: %v", archivePath, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close archive file %s: %v", archivePath, err)
	}
}

func TestIndexPriorityDirs_prefers_open_docs_and_impacted_modules(t *testing.T) {
	root := fs.GetCanonicalPath(t.TempDir())
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)

	docAPath := filepath.Join(root, "app", "a.c3")
	docBPath := filepath.Join(root, "dep", "b.c3")

	docA := document.NewDocumentFromString(docAPath, "module app::a; import app::b; fn void a() {}")
	docB := document.NewDocumentFromString(docBPath, "module app::b; fn void b() {}")
	state.RefreshDocumentIdentifiers(&docA, &p)
	state.RefreshDocumentIdentifiers(&docB, &p)

	// Trigger signature change so impacted set includes dependent module docs.
	docBChanged := document.NewDocumentFromString(docBPath, "module app::b; fn int b(int x) { return x; }")
	state.RefreshDocumentIdentifiers(&docBChanged, &p)

	srv := &Server{state: &state}
	priority := srv.indexPriorityDirs(root)

	assert.Contains(t, priority, fs.GetCanonicalPath(filepath.Dir(docAPath)))
	assert.Contains(t, priority, fs.GetCanonicalPath(filepath.Dir(docBPath)))
}

func TestEnsureWorkspaceIndexedForURI_skips_non_buildable_root(t *testing.T) {
	root := fs.GetCanonicalPath(t.TempDir())
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)

	var calls atomic.Int32
	srv := &Server{
		state: &state,
		workspaceIndexer: func(ctx context.Context, path string) {
			calls.Add(1)
		},
	}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(filepath.Join(root, "src", "main.c3"), option.None[string]()))
	srv.ensureWorkspaceIndexedForURI(uri)

	assert.Equal(t, int32(0), calls.Load())
}

func TestParseDependencySearchPaths_supports_json_with_comments(t *testing.T) {
	root := fs.GetCanonicalPath(t.TempDir())
	dep := filepath.Join(root, "deps")
	if err := os.MkdirAll(dep, 0o755); err != nil {
		t.Fatalf("failed to create dependency dir: %v", err)
	}

	projectJSON := `{
	  // dep paths
	  "dependency-search-paths": ["./deps", "./missing", "./deps"]
	}`

	paths := parseDependencySearchPaths(projectJSON, root)
	if len(paths) != 1 {
		t.Fatalf("expected one resolved dependency path, got %v", paths)
	}
	if paths[0] != fs.GetCanonicalPath(dep) {
		t.Fatalf("unexpected dependency path: %v", paths)
	}
}

func TestIndexWorkspaceAtWithContext_indexes_dependency_search_paths(t *testing.T) {
	workspaceRoot := fs.GetCanonicalPath(t.TempDir())
	depRoot := fs.GetCanonicalPath(filepath.Join(workspaceRoot, "..", "deps"))

	if err := os.MkdirAll(filepath.Join(workspaceRoot, "src"), 0o755); err != nil {
		t.Fatalf("failed to create workspace src: %v", err)
	}
	if err := os.MkdirAll(depRoot, 0o755); err != nil {
		t.Fatalf("failed to create dep root: %v", err)
	}

	projectJSON := `{
		"dependency-search-paths": ["../deps"],
		"dependencies": ["sqlite3"],
		"sources": ["src/**"]
	}`
	if err := os.WriteFile(filepath.Join(workspaceRoot, "project.json"), []byte(projectJSON), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	depSource := `module sqlite3;
alias SqliteHandle = void*;`
	if err := os.WriteFile(filepath.Join(depRoot, "sqlite.c3i"), []byte(depSource), 0o644); err != nil {
		t.Fatalf("failed to write sqlite dependency: %v", err)
	}

	mainSource := `module app;
import sqlite3;
fn void main() {
	sqlite3::SqliteHandle db;
}`
	mainPath := filepath.Join(workspaceRoot, "src", "main.c3")
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("failed to write main source: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &p, search: &searchImpl}
	srv.state.SetProjectRootURI(workspaceRoot)
	srv.configureProjectForRoot(workspaceRoot)

	if ok := srv.indexWorkspaceAtWithContext(context.Background(), workspaceRoot); !ok {
		t.Fatalf("expected indexing to complete")
	}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(mainPath, option.None[string]()))
	idx := strings.Index(mainSource, "SqliteHandle") + 2
	pos := byteIndexToLSPPosition(mainSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for dependency symbol SqliteHandle")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if !strings.Contains(content.Value, "SqliteHandle") {
		t.Fatalf("expected hover to contain dependency symbol name, got: %s", content.Value)
	}
}

func TestIndexWorkspaceAtWithContext_indexes_dependency_c3l_directory(t *testing.T) {
	workspaceRoot := fs.GetCanonicalPath(t.TempDir())
	depRoot := fs.GetCanonicalPath(filepath.Join(workspaceRoot, "..", "deps"))
	depPkgRoot := filepath.Join(depRoot, "sqlite3.c3l", "src")

	if err := os.MkdirAll(filepath.Join(workspaceRoot, "src"), 0o755); err != nil {
		t.Fatalf("failed to create workspace src: %v", err)
	}
	if err := os.MkdirAll(depPkgRoot, 0o755); err != nil {
		t.Fatalf("failed to create dependency package dir: %v", err)
	}

	projectJSON := `{
		"dependency-search-paths": ["../deps"],
		"dependencies": ["sqlite3"],
		"sources": ["src/**"]
	}`
	if err := os.WriteFile(filepath.Join(workspaceRoot, "project.json"), []byte(projectJSON), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	depSource := `module sqlite3;
alias SqliteHandle = void*;`
	if err := os.WriteFile(filepath.Join(depPkgRoot, "sqlite.c3i"), []byte(depSource), 0o644); err != nil {
		t.Fatalf("failed to write sqlite dependency: %v", err)
	}

	mainSource := `module app;
import sqlite3;
fn void main() {
	sqlite3::SqliteHandle db;
}`
	mainPath := filepath.Join(workspaceRoot, "src", "main.c3")
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("failed to write main source: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &p, search: &searchImpl}
	srv.state.SetProjectRootURI(workspaceRoot)
	srv.configureProjectForRoot(workspaceRoot)

	if ok := srv.indexWorkspaceAtWithContext(context.Background(), workspaceRoot); !ok {
		t.Fatalf("expected indexing to complete")
	}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(mainPath, option.None[string]()))
	idx := strings.Index(mainSource, "SqliteHandle") + 2
	pos := byteIndexToLSPPosition(mainSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for dependency symbol SqliteHandle")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if !strings.Contains(content.Value, "SqliteHandle") {
		t.Fatalf("expected hover to contain dependency symbol name, got: %s", content.Value)
	}
}

func TestIndexWorkspaceAtWithContext_indexes_dependency_c3l_archive(t *testing.T) {
	workspaceRoot := fs.GetCanonicalPath(t.TempDir())
	depRoot := fs.GetCanonicalPath(filepath.Join(workspaceRoot, "..", "deps"))

	if err := os.MkdirAll(filepath.Join(workspaceRoot, "src"), 0o755); err != nil {
		t.Fatalf("failed to create workspace src: %v", err)
	}
	if err := os.MkdirAll(depRoot, 0o755); err != nil {
		t.Fatalf("failed to create dep root: %v", err)
	}

	projectJSON := `{
		"dependency-search-paths": ["../deps"],
		"dependencies": ["sqlite3"],
		"sources": ["src/**"]
	}`
	if err := os.WriteFile(filepath.Join(workspaceRoot, "project.json"), []byte(projectJSON), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	writeC3LArchive(t, filepath.Join(depRoot, "sqlite3.c3l"), map[string]string{
		"manifest.json": `{"provides":"sqlite3"}`,
		"sqlite.c3i":    "module sqlite3;\nalias SqliteHandle = void*;",
	})

	mainSource := `module app;
import sqlite3;
fn void main() {
	sqlite3::SqliteHandle db;
}`
	mainPath := filepath.Join(workspaceRoot, "src", "main.c3")
	if err := os.WriteFile(mainPath, []byte(mainSource), 0o644); err != nil {
		t.Fatalf("failed to write main source: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &p, search: &searchImpl}
	srv.state.SetProjectRootURI(workspaceRoot)
	srv.configureProjectForRoot(workspaceRoot)

	if ok := srv.indexWorkspaceAtWithContext(context.Background(), workspaceRoot); !ok {
		t.Fatalf("expected indexing to complete")
	}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(mainPath, option.None[string]()))
	idx := strings.Index(mainSource, "SqliteHandle") + 2
	pos := byteIndexToLSPPosition(mainSource, idx)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for dependency symbol SqliteHandle")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if !strings.Contains(content.Value, "SqliteHandle") {
		t.Fatalf("expected hover to contain dependency symbol name, got: %s", content.Value)
	}
}
