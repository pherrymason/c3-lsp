package server

import (
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestTextDocumentDefinition_returns_nil_inside_string_literal(t *testing.T) {
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

	doc := document.NewDocumentFromDocURI("file:///tmp/definition_literal_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	result, err := srv.TextDocumentDefinition(nil, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/definition_literal_test.c3"},
			Position: protocol.Position{
				Line:      3,
				Character: 18,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected definition error: %v", err)
	}

	if result != nil {
		t.Fatalf("expected nil definition inside string literal, got %#v", result)
	}
}

func TestTextDocumentDefinition_resolves_fully_qualified_type_with_hover_fallback(t *testing.T) {
	logger := commonlog.MockLogger{}
	state := project_state.NewProjectState(logger, option.Some("dummy"), false)
	prs := parser.NewParser(logger)
	searchImpl := search.NewSearch(logger, false)

	srv := &Server{
		state:  &state,
		parser: &prs,
		search: &searchImpl,
	}

	depSource := `module bindgen::bg;
struct BGOptions {
	int x;
}`
	depURI := protocol.DocumentUri("file:///tmp/definition_bindgen_dep_test.c3")
	depDoc := document.NewDocumentFromDocURI(depURI, depSource, 1)
	state.RefreshDocumentIdentifiers(depDoc, &prs)

	appSource := `
fn void main() {
	bindgen::bg::BGOptions opts = {};
}`
	appURI := protocol.DocumentUri("file:///tmp/definition_bindgen_app_test.c3")
	appDoc := document.NewDocumentFromDocURI(appURI, appSource, 1)
	state.RefreshDocumentIdentifiers(appDoc, &prs)
	idx := strings.Index(appSource, "BGOptions") + len("BG")

	result, err := srv.TextDocumentDefinition(nil, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: appURI},
			Position:     byteIndexToLSPPosition(appSource, idx),
		},
	})
	if err != nil {
		t.Fatalf("unexpected definition error: %v", err)
	}

	loc, ok := result.(protocol.Location)
	if !ok {
		t.Fatalf("expected protocol.Location result, got %#v", result)
	}
	if loc.URI != depURI {
		t.Fatalf("expected definition URI %s, got %s", depURI, loc.URI)
	}
}
