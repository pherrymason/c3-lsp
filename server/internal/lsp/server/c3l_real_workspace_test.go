package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRealWorkspace_c3l_archive_dependency_resolves_sqlite3_symbol(t *testing.T) {
	workspaceRoot := os.Getenv("C3LSP_PERF_WORKSPACE")
	if workspaceRoot == "" {
		workspaceRoot = defaultPerfWorkspace
	}

	workspaceRoot = fs.GetCanonicalPath(workspaceRoot)
	if workspaceRoot == "" {
		t.Skip("workspace path is empty")
	}

	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		t.Skipf("workspace does not exist: %s", workspaceRoot)
	}

	targetPath := filepath.Join(workspaceRoot, "lib", "blem.c3l", "app_example", "http_todo_app_example.c3")
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Skipf("failed to read target file %s: %v", targetPath, err)
	}
	targetSource := string(targetBytes)

	needle := "sqlite3::SqliteHandle db"
	idx := strings.Index(targetSource, needle)
	if idx < 0 {
		t.Skipf("needle not found in %s: %q", targetPath, needle)
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

	uri := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))
	pos := byteIndexToLSPPosition(targetSource, idx+len("sqlite3::SqliteHand"))
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for sqlite3 archive dependency symbol")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if !strings.Contains(content.Value, "SqliteHandle") {
		t.Fatalf("expected hover to contain SqliteHandle, got: %s", content.Value)
	}
	if !strings.Contains(content.Value, "sqlite3") {
		t.Fatalf("expected hover to reference sqlite3 module, got: %s", content.Value)
	}
}

func TestRealWorkspace_bindgen_first_hover_resolves_BGOptions_without_warmup(t *testing.T) {
	workspaceRoot := fs.GetCanonicalPath("/Users/f00lg/github/c3/bindgen.c3l")
	if workspaceRoot == "" {
		t.Skip("bindgen workspace path is empty")
	}

	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		t.Skipf("workspace does not exist: %s", workspaceRoot)
	}

	targetPath := filepath.Join(workspaceRoot, "examples", "glfw.c3")
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Skipf("failed to read target file %s: %v", targetPath, err)
	}
	targetSource := string(targetBytes)

	needle := "BGOptions opts"
	idx := strings.Index(targetSource, needle)
	if idx < 0 {
		t.Skipf("needle not found in %s: %q", targetPath, needle)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &p, search: &searchImpl}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))
	pos := byteIndexToLSPPosition(targetSource, idx+7)
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for BGOptions on first hover")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if snapshot := srv.state.Snapshot(); snapshot == nil || len(snapshot.ModulesByName("bindgen::bg")) == 0 {
		t.Fatalf("expected bindgen::bg to be loaded after hover, got hover: %s", content.Value)
	}
	if !strings.Contains(content.Value, "In module **[bindgen::bg]**") {
		t.Fatalf("expected resolved bindgen hover, got: %s", content.Value)
	}
}

func TestRealWorkspace_bindgen_first_hover_resolves_bgstr_is_between_without_warmup(t *testing.T) {
	workspaceRoot := fs.GetCanonicalPath("/Users/f00lg/github/c3/bindgen.c3l")
	if workspaceRoot == "" {
		t.Skip("bindgen workspace path is empty")
	}

	if info, err := os.Stat(workspaceRoot); err != nil || !info.IsDir() {
		t.Skipf("workspace does not exist: %s", workspaceRoot)
	}

	targetPath := filepath.Join(workspaceRoot, "examples", "glfw.c3")
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Skipf("failed to read target file %s: %v", targetPath, err)
	}
	targetSource := string(targetBytes)

	needle := "bgstr::is_between"
	idx := strings.Index(targetSource, needle)
	if idx < 0 {
		t.Skipf("needle not found in %s: %q", targetPath, needle)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	p := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{state: &state, parser: &p, search: &searchImpl}

	uri := protocol.DocumentUri(fs.ConvertPathToURI(targetPath, option.None[string]()))
	pos := byteIndexToLSPPosition(targetSource, idx+len("bgstr::is_bet"))
	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     pos,
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover for bgstr::is_between on first hover")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected hover markdown content")
	}
	if !strings.Contains(content.Value, "is_between") {
		t.Fatalf("expected resolved bindgen function hover, got: %s", content.Value)
	}
	if !strings.Contains(content.Value, "bindgen::bgstr") {
		t.Fatalf("expected bindgen::bgstr module hover, got: %s", content.Value)
	}
}
