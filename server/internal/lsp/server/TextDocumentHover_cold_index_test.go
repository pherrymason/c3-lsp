package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	lspcontext "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type coldStartRetrySearchMock struct {
	symbol symbols.Indexable
	calls  atomic.Int32
}

func (m *coldStartRetrySearchMock) FindSymbolDeclarationInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) option.Option[symbols.Indexable] {
	if m.calls.Add(1) == 1 {
		return option.None[symbols.Indexable]()
	}

	return option.Some(m.symbol)
}

func (m *coldStartRetrySearchMock) FindImplementationsInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) []symbols.Indexable {
	return nil
}

func (m *coldStartRetrySearchMock) FindReferencesInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState, _ bool) []protocol.Location {
	return nil
}

func (m *coldStartRetrySearchMock) BuildCompletionList(_ lspcontext.CursorContext, _ *project_state.ProjectState) []protocol.CompletionItem {
	return nil
}

type coldStartNoneSearchMock struct{}

func (coldStartNoneSearchMock) FindSymbolDeclarationInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) option.Option[symbols.Indexable] {
	return option.None[symbols.Indexable]()
}

func (coldStartNoneSearchMock) FindImplementationsInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState) []symbols.Indexable {
	return nil
}

func (coldStartNoneSearchMock) FindReferencesInWorkspace(_ string, _ symbols.Position, _ *project_state.ProjectState, _ bool) []protocol.Location {
	return nil
}

func (coldStartNoneSearchMock) BuildCompletionList(_ lspcontext.CursorContext, _ *project_state.ProjectState) []protocol.CompletionItem {
	return nil
}

func TestTextDocumentHover_retries_after_short_indexing_wait(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	docPath := filepath.Join(tmpDir, "src", "main.c3")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	docURI := protocol.DocumentUri(fs.ConvertPathToURI(docPath, option.None[string]()))
	source := "module app;\nfn void main() {\n  BGOptions opts;\n}\n"
	doc := document.NewDocumentFromDocURI(string(docURI), source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	decl := symbols.NewVariable(
		"BGOptions",
		symbols.NewTypeFromString("int", ""),
		"bindgen",
		doc.URI,
		symbols.NewRange(2, 2, 2, 11),
		symbols.NewRange(2, 2, 2, 11),
	)
	searchMock := &coldStartRetrySearchMock{symbol: &decl}

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: searchMock,
	}

	root := fs.GetCanonicalPath(tmpDir)
	srv.idx.mu.Lock()
	srv.ensureIndexingStateMapsLocked()
	srv.setRootState(root, rootStateIndexing)
	srv.idx.mu.Unlock()

	go func() {
		time.Sleep(25 * time.Millisecond)
		srv.markRootIndexed(root)
	}()

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 2, Character: 4},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response after bounded wait retry")
	}
	if searchMock.calls.Load() < 2 {
		t.Fatalf("expected hover lookup to retry after indexing wait, got %d calls", searchMock.calls.Load())
	}
}

func TestTextDocumentHover_returns_indexing_placeholder_while_project_is_indexing(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	docPath := filepath.Join(tmpDir, "src", "main.c3")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	docURI := protocol.DocumentUri(fs.ConvertPathToURI(docPath, option.None[string]()))
	source := "module app;\nfn void main() {\n  BGOptions opts;\n}\n"
	doc := document.NewDocumentFromDocURI(string(docURI), source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: coldStartNoneSearchMock{},
	}

	root := fs.GetCanonicalPath(tmpDir)
	srv.idx.mu.Lock()
	srv.ensureIndexingStateMapsLocked()
	srv.setRootState(root, rootStateIndexing)
	srv.idx.mu.Unlock()

	resolvedRoot, stateBefore, ok := srv.hoverRootState(docURI)
	if !ok || stateBefore != rootStateIndexing {
		t.Fatalf("expected buildable indexing root before hover, got ok=%v state=%s set_root=%s resolved_root=%s", ok, stateBefore, root, resolvedRoot)
	}

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 2, Character: 4},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover placeholder while indexing")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover placeholder")
	}
	if !strings.Contains(content.Value, "Indexing project symbols") {
		t.Fatalf("expected indexing placeholder message, got: %s", content.Value)
	}
}

func TestTextDocumentHover_starts_work_done_progress_when_hover_triggers_indexing(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	docPath := filepath.Join(tmpDir, "src", "main.c3")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	source := "module app;\nfn void main() {\n  BGOptions opts;\n}\n"
	if err := os.WriteFile(docPath, []byte(source), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	docURI := protocol.DocumentUri(fs.ConvertPathToURI(docPath, option.None[string]()))
	doc := document.NewDocumentFromDocURI(string(docURI), source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: coldStartNoneSearchMock{},
	}
	var caps protocol.ClientCapabilities
	if err := json.Unmarshal([]byte(`{"window":{"workDoneProgress":true}}`), &caps); err != nil {
		t.Fatalf("failed to build client capabilities: %v", err)
	}
	srv.clientCapabilities = caps

	created := make(chan struct{}, 1)
	progress := make(chan string, 16)
	ctx := &glsp.Context{
		Call: func(method string, params any, result any) {
			if method == protocol.ServerWindowWorkDoneProgressCreate {
				select {
				case created <- struct{}{}:
				default:
				}
			}
		},
		Notify: func(method string, params any) {
			if method != protocol.MethodProgress {
				return
			}
			payload, ok := params.(map[string]any)
			if !ok {
				return
			}
			value, ok := payload["value"].(map[string]any)
			if !ok {
				return
			}
			if message, ok := value["message"].(*string); ok && message != nil {
				select {
				case progress <- *message:
				default:
				}
			}
			if message, ok := value["message"].(string); ok {
				select {
				case progress <- message:
				default:
				}
			}
		},
	}

	srv.workspaceIndexer = func(indexCtx context.Context, root string) {
		time.Sleep(50 * time.Millisecond)
		srv.indexWorkspaceAtWithContextAndProgress(indexCtx, root, ctx)
	}

	hover, err := srv.TextDocumentHover(ctx, &protocol.HoverParams{TextDocumentPositionParams: protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: 2, Character: 4},
	}})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover placeholder while indexing")
	}

	select {
	case <-created:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected hover-triggered indexing to create workDoneProgress")
	}

	foundProgressMessage := false
	deadline := time.After(1 * time.Second)
	for !foundProgressMessage {
		select {
		case msg := <-progress:
			if strings.Contains(msg, "Scanning root") || strings.Contains(msg, "Resolving workspace") || strings.Contains(msg, "Indexed") {
				foundProgressMessage = true
			}
		case <-deadline:
			t.Fatalf("expected hover-triggered indexing progress messages")
		}
	}
}

func TestTextDocumentHover_module_token_does_not_short_circuit_to_nil_while_indexing(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "project.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("failed to create project.json: %v", err)
	}

	docPath := filepath.Join(tmpDir, "src", "visitors.c3")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}

	docURI := protocol.DocumentUri(fs.ConvertPathToURI(docPath, option.None[string]()))
	source := "module bgimpl::vtor;\nimport bgimpl;\nfn void run(String s) {\n  String t = trans::apply(s, null, tmem);\n}\n"
	doc := document.NewDocumentFromDocURI(string(docURI), source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: coldStartNoneSearchMock{},
	}

	root := fs.GetCanonicalPath(tmpDir)
	srv.idx.mu.Lock()
	srv.ensureIndexingStateMapsLocked()
	srv.setRootState(root, rootStateIndexing)
	srv.idx.mu.Unlock()

	idx := strings.Index(source, "trans::") + len("tran")
	if idx < 0 {
		t.Fatalf("failed to locate trans:: token in source")
	}

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     byteIndexToLSPPosition(source, idx),
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected indexing placeholder hover for module token while indexing")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markdown hover placeholder for module token")
	}
	if !strings.Contains(content.Value, "Indexing project symbols") {
		t.Fatalf("expected indexing placeholder message, got: %s", content.Value)
	}
}
