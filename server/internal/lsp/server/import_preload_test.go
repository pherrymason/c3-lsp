package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestPreloadImportedRootModulesForURI_loads_root_import_descendants(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	visitorsPath := filepath.Join(srcDir, "visitors.c3")
	visitorsSource := "module bgimpl::vtor;\nimport bgimpl;\nfn void run() { check::apply(\"x\", null); }\n"
	if err := os.WriteFile(visitorsPath, []byte(visitorsSource), 0o644); err != nil {
		t.Fatalf("failed to write visitors file: %v", err)
	}

	checkPath := filepath.Join(srcDir, "check.c3")
	checkSource := "module bgimpl::check;\nfn bool apply(String s, BGCheckFn f) { return true; }\n"
	if err := os.WriteFile(checkPath, []byte(checkSource), 0o644); err != nil {
		t.Fatalf("failed to write check file: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	server := &Server{
		state:             &state,
		parser:            &prs,
		rootCache:         projectRootCacheState{cache: make(map[string]string)},
		importPreloadDone: map[string]struct{}{},
	}

	visitorsURI := protocol.DocumentUri(fs.ConvertPathToURI(visitorsPath, option.None[string]()))
	visitorsDoc := document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1)
	state.RefreshDocumentIdentifiers(visitorsDoc, &prs)

	if got := state.GetDocument(fs.GetCanonicalPath(checkPath)); got != nil {
		t.Fatalf("expected check file to be unloaded before preload")
	}

	server.preloadImportedRootModulesForURI(visitorsURI)

	if got := state.GetDocument(fs.GetCanonicalPath(checkPath)); got == nil {
		t.Fatalf("expected check file to be loaded from bgimpl root import preload")
	}
}

func TestPreloadImportedRootModulesForURI_loads_descendants_in_nested_project_layout(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	nestedRoot := filepath.Join(tmpDir, "bindgen.c3l")
	srcDir := filepath.Join(nestedRoot, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create nested src dir: %v", err)
	}

	visitorsPath := filepath.Join(srcDir, "visitors.c3")
	visitorsSource := "module bgimpl::vtor;\nimport bindgen::bg, bgimpl;\nfn void run() { check::apply(\"x\", null); }\n"
	if err := os.WriteFile(visitorsPath, []byte(visitorsSource), 0o644); err != nil {
		t.Fatalf("failed to write visitors file: %v", err)
	}

	checkPath := filepath.Join(srcDir, "check.c3")
	checkSource := "module bgimpl::check;\nfn bool apply(String s, BGCheckFn f) { return true; }\n"
	if err := os.WriteFile(checkPath, []byte(checkSource), 0o644); err != nil {
		t.Fatalf("failed to write check file: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	server := &Server{
		state:             &state,
		parser:            &prs,
		rootCache:         projectRootCacheState{cache: make(map[string]string)},
		importPreloadDone: map[string]struct{}{},
	}

	visitorsURI := protocol.DocumentUri(fs.ConvertPathToURI(visitorsPath, option.None[string]()))
	visitorsDoc := document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1)
	state.RefreshDocumentIdentifiers(visitorsDoc, &prs)

	if got := state.GetDocument(fs.GetCanonicalPath(checkPath)); got != nil {
		t.Fatalf("expected check file to be unloaded before preload")
	}

	server.preloadImportedRootModulesForURI(visitorsURI)

	if got := state.GetDocument(fs.GetCanonicalPath(checkPath)); got == nil {
		t.Fatalf("expected check file to be loaded from bgimpl root import preload in nested layout")
	}
}

func TestPreloadImportedRootModulesForURI_loads_module_by_declaration_not_filename_guess(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to write project.json: %v", err)
	}

	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	visitorsPath := filepath.Join(srcDir, "visitors.c3")
	visitorsSource := "module bgimpl::vtor;\nimport bgimpl;\nfn void run() { wter::constDecl(0, null, \"\", null, \"\", {}); }\n"
	if err := os.WriteFile(visitorsPath, []byte(visitorsSource), 0o644); err != nil {
		t.Fatalf("failed to write visitors file: %v", err)
	}

	writersPath := filepath.Join(srcDir, "writers.c3")
	writersSource := "module bgimpl::wter;\nfn void constDecl(usz out, WriteState* state, String enumName, String? transCursor, String val, WriteAttrs attrs) {}\n"
	if err := os.WriteFile(writersPath, []byte(writersSource), 0o644); err != nil {
		t.Fatalf("failed to write writers file: %v", err)
	}

	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	server := &Server{
		state:             &state,
		parser:            &prs,
		rootCache:         projectRootCacheState{cache: make(map[string]string)},
		importPreloadDone: map[string]struct{}{},
	}

	visitorsURI := protocol.DocumentUri(fs.ConvertPathToURI(visitorsPath, option.None[string]()))
	visitorsDoc := document.NewDocumentFromDocURI(visitorsURI, visitorsSource, 1)
	state.RefreshDocumentIdentifiers(visitorsDoc, &prs)

	if got := state.GetDocument(fs.GetCanonicalPath(writersPath)); got != nil {
		t.Fatalf("expected writers file to be unloaded before preload")
	}

	server.preloadImportedRootModulesForURI(visitorsURI)

	if got := state.GetDocument(fs.GetCanonicalPath(writersPath)); got == nil {
		t.Fatalf("expected writers file to be loaded by module declaration match for bgimpl::wter")
	}
}

func TestExtractDeclaredModuleName_ignores_comment_prose_and_matches_real_declaration(t *testing.T) {
	source := []byte(`<*
 The module contains translator functions.
 Translators of top-level declarations also support
 module wrapping.
*>
module bgimpl::ttor;
`)

	got := extractDeclaredModuleName(source)
	if got != "bgimpl::ttor" {
		t.Fatalf("expected bgimpl::ttor, got %q", got)
	}
}
