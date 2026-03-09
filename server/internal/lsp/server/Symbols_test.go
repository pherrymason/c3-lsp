package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDocumentSymbol_returns_hierarchical_symbols(t *testing.T) {
	source := `module app;

struct Fiber {
	int entry;
}

enum State {
	RUNNING,
}

fn void run(Fiber* fiber) {
	fiber.entry = 1;
}`

	uri := protocol.DocumentUri("file:///tmp/document_symbol_hierarchy_test.c3")
	srv := buildRenameTestServer(uri, source)

	result, err := srv.TextDocumentDocumentSymbol(nil, &protocol.DocumentSymbolParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}})
	if err != nil {
		t.Fatalf("unexpected documentSymbol error: %v", err)
	}

	docSymbols, ok := result.([]protocol.DocumentSymbol)
	if !ok {
		t.Fatalf("expected []DocumentSymbol result, got %T", result)
	}
	if len(docSymbols) != 1 {
		t.Fatalf("expected one module symbol, got: %d", len(docSymbols))
	}

	module := docSymbols[0]
	if module.Name != "app" {
		t.Fatalf("expected module symbol 'app', got: %s", module.Name)
	}

	hasFiber := false
	hasRun := false
	hasState := false
	for _, child := range module.Children {
		switch child.Name {
		case "Fiber":
			hasFiber = true
			if len(child.Children) == 0 || child.Children[0].Name != "entry" {
				t.Fatalf("expected struct member child under Fiber, got: %#v", child.Children)
			}
		case "run":
			hasRun = true
		case "State":
			hasState = true
		}
	}

	if !hasFiber || !hasRun || !hasState {
		t.Fatalf("expected Fiber/run/State children under module, got: %#v", module.Children)
	}
}

func TestWorkspaceSymbol_filters_and_returns_locations(t *testing.T) {
	declURI := protocol.DocumentUri("file:///tmp/workspace_symbol_decl_test.c3")
	useURI := protocol.DocumentUri("file:///tmp/workspace_symbol_use_test.c3")

	declSource := `module app;

fn void parse_port() {
}

fn void parse_host() {
}`

	useSource := `module net;

struct Parser {
	int port;
}`

	srv := buildRenameTestServerWithDocuments([]renameTestDocument{{uri: declURI, source: declSource}, {uri: useURI, source: useSource}})

	items, err := srv.WorkspaceSymbol(nil, &protocol.WorkspaceSymbolParams{Query: "parse"})
	if err != nil {
		t.Fatalf("unexpected workspace/symbol error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected non-empty workspace symbol result")
	}

	hasParsePort := false
	hasParseHost := false
	for _, item := range items {
		if !strings.Contains(strings.ToLower(item.Name), "parse") {
			t.Fatalf("expected filtered names to contain query 'parse', got: %s", item.Name)
		}
		if item.Name == "parse_port" {
			hasParsePort = true
		}
		if item.Name == "parse_host" {
			hasParseHost = true
		}
		if item.Location.URI == "" {
			t.Fatalf("expected symbol location URI to be set")
		}
	}

	if !hasParsePort || !hasParseHost {
		t.Fatalf("expected parse_port and parse_host in workspace symbols, got: %#v", items)
	}
}

func TestWorkspaceSymbolResolve_returnsCopyOfInput(t *testing.T) {
	item := &protocol.SymbolInformation{
		Name: "parse_port",
		Kind: protocol.SymbolKindFunction,
		Location: protocol.Location{
			URI:   protocol.DocumentUri("file:///tmp/workspace_symbol_resolve_test.c3"),
			Range: protocol.Range{Start: protocol.Position{Line: 1, Character: 2}, End: protocol.Position{Line: 1, Character: 12}},
		},
	}
	srv := buildRenameTestServer(protocol.DocumentUri("file:///tmp/workspace_symbol_resolve_test.c3"), "module app;")

	resolved, err := srv.WorkspaceSymbolResolve(nil, item)
	if err != nil {
		t.Fatalf("unexpected workspaceSymbol/resolve error: %v", err)
	}
	if resolved == nil {
		t.Fatalf("expected resolved symbol information")
	}
	if resolved.Name != item.Name || resolved.Location.URI != item.Location.URI {
		t.Fatalf("expected resolved symbol to preserve input values: got %#v", resolved)
	}
}

func TestTextDocumentDocumentSymbol_faultdef_has_non_empty_truncated_name_and_selection_range(t *testing.T) {
	source := `module app;

faultdef IO_ERROR, PARSE_ERROR, SOCKET_CLOSED, NETWORK_DOWN;
`

	uri := protocol.DocumentUri("file:///tmp/document_symbol_faultdef_test.c3")
	srv := buildRenameTestServer(uri, source)

	result, err := srv.TextDocumentDocumentSymbol(nil, &protocol.DocumentSymbolParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}})
	if err != nil {
		t.Fatalf("unexpected documentSymbol error: %v", err)
	}

	docSymbols, ok := result.([]protocol.DocumentSymbol)
	if !ok {
		t.Fatalf("expected []DocumentSymbol result, got %T", result)
	}
	if len(docSymbols) != 1 {
		t.Fatalf("expected one module symbol, got: %d", len(docSymbols))
	}

	module := docSymbols[0]
	if len(module.Children) == 0 {
		t.Fatalf("expected module children to include faultdef")
	}

	var fault *protocol.DocumentSymbol
	for i := range module.Children {
		child := &module.Children[i]
		if strings.HasPrefix(child.Name, "faultdef") {
			fault = child
			break
		}
	}

	if fault == nil {
		t.Fatalf("expected a faultdef symbol under module, got: %#v", module.Children)
	}

	if fault.Name != "faultdef IO_ERROR, PARSE_ERROR, SOCKET_CLOSED, ..." {
		t.Fatalf("unexpected faultdef name: %q", fault.Name)
	}

	if fault.SelectionRange.Start.Line == 0 && fault.SelectionRange.Start.Character == 0 &&
		fault.SelectionRange.End.Line == 0 && fault.SelectionRange.End.Character == 0 {
		t.Fatalf("expected non-zero selectionRange for faultdef symbol: %#v", fault.SelectionRange)
	}

	if fault.SelectionRange.Start.Line < fault.Range.Start.Line ||
		(fault.SelectionRange.Start.Line == fault.Range.Start.Line && fault.SelectionRange.Start.Character < fault.Range.Start.Character) {
		t.Fatalf("expected selectionRange start within symbol range: range=%#v selection=%#v", fault.Range, fault.SelectionRange)
	}

	if fault.SelectionRange.End.Line > fault.Range.End.Line ||
		(fault.SelectionRange.End.Line == fault.Range.End.Line && fault.SelectionRange.End.Character > fault.Range.End.Character) {
		t.Fatalf("expected selectionRange end within symbol range: range=%#v selection=%#v", fault.Range, fault.SelectionRange)
	}
}

func TestProtocolHandlerWithExtensions_dispatchesWorkspaceSymbolResolve(t *testing.T) {
	base := &protocol.Handler{}
	base.SetInitialized(true)

	called := false
	handler := &protocolHandlerWithExtensions{
		base: base,
		workspaceSymbolResolve: func(_ *glsp.Context, params *protocol.SymbolInformation) (*protocol.SymbolInformation, error) {
			called = true
			resolved := *params
			resolved.Name = params.Name + "_resolved"
			return &resolved, nil
		},
	}

	payload, err := json.Marshal(protocol.SymbolInformation{Name: "parse_port", Kind: protocol.SymbolKindFunction})
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	result, validMethod, validParams, handleErr := handler.Handle(&glsp.Context{Method: methodWorkspaceSymbolResolve, Params: payload})
	if handleErr != nil {
		t.Fatalf("unexpected handle error: %v", handleErr)
	}
	if !validMethod || !validParams {
		t.Fatalf("expected valid custom method+params, got validMethod=%t validParams=%t", validMethod, validParams)
	}
	if !called {
		t.Fatalf("expected workspaceSymbol/resolve extension callback")
	}

	resolved, ok := result.(*protocol.SymbolInformation)
	if !ok {
		t.Fatalf("expected resolved symbol pointer result, got %T", result)
	}
	if resolved.Name != "parse_port_resolved" {
		t.Fatalf("unexpected resolved name: %#v", resolved)
	}
}
