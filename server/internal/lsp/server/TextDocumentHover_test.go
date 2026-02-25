package server

import (
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestGenericTypeSuffixAtPosition(t *testing.T) {
	t.Run("simple generic", func(t *testing.T) {
		source := "List{int} l;"
		suffix, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 1}))
		if !ok {
			t.Fatalf("expected generic suffix to be found")
		}
		if suffix != "{int}" {
			t.Fatalf("unexpected suffix: got %q", suffix)
		}
	})

	t.Run("nested generic with spacing", func(t *testing.T) {
		source := "HashMap { String, List{int} } m;"
		suffix, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 2}))
		if !ok {
			t.Fatalf("expected generic suffix to be found")
		}
		if suffix != "{ String, List{int} }" {
			t.Fatalf("unexpected suffix: got %q", suffix)
		}
	})

	t.Run("not a generic type usage", func(t *testing.T) {
		source := "List l;"
		_, ok := genericTypeSuffixAtPosition(source, symbols.NewPositionFromLSPPosition(protocol.Position{Line: 0, Character: 1}))
		if ok {
			t.Fatalf("expected no generic suffix")
		}
	})
}

func TestTextDocumentHover_includes_generic_arguments_for_hovered_type_identifier(t *testing.T) {
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
	struct List {
		int value;
	}

	fn void main() {
		List{int} list;
	}`

	doc := document.NewDocumentFromDocURI("file:///tmp/hover_generic_test.c3", source, 1)
	state.RefreshDocumentIdentifiers(doc, &prs)

	hover, err := srv.TextDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///tmp/hover_generic_test.c3"},
			Position: protocol.Position{
				Line:      6,
				Character: 4,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected hover error: %v", err)
	}
	if hover == nil {
		t.Fatalf("expected hover response")
	}

	content, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected markup content in hover")
	}

	if !strings.Contains(content.Value, "List{int}") {
		t.Fatalf("expected hover to include generic type arguments, got: %s", content.Value)
	}
}
