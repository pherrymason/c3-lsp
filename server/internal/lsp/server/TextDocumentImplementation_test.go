package server

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentImplementation_falls_back_to_definition_for_free_functions(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
fn int default_config() { return 1; }
fn void main() {
	int conf = default_config();
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/implementation_fallback_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	result, err := srv.TextDocumentImplementation(nil, &protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/implementation_fallback_test.c3"},
			Position: protocol.Position{
				Line:      3,
				Character: 18,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected implementation error: %v", err)
	}

	locations, ok := result.([]protocol.Location)
	if !ok {
		t.Fatalf("expected []protocol.Location result, got %#v", result)
	}
	if len(locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locations))
	}
	if locations[0].URI != "file:///tmp/implementation_fallback_test.c3" {
		t.Fatalf("unexpected location URI: %s", locations[0].URI)
	}
	if locations[0].Range.Start.Line != 1 {
		t.Fatalf("expected declaration line 1, got %d", locations[0].Range.Start.Line)
	}
}

func TestTextDocumentImplementation_returns_nil_inside_string_literal(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	source := `module app;
alias s = int;
fn void main() {
	io::printfn("%s", 1);
}`

	doc := document.NewDocumentFromDocURI("file:///tmp/implementation_literal_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	result, err := srv.TextDocumentImplementation(nil, &protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/implementation_literal_test.c3"},
			Position: protocol.Position{
				Line:      3,
				Character: 15,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected implementation error: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil implementation inside string literal, got %#v", result)
	}
}
