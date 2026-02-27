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
