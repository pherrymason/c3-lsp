package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func hoverFromBody(state *TestState, body string) option.Option[protocol.Hover] {
	cursorlessBody, position := parseBodyWithCursor(body)
	state.registerDoc("x", cursorlessBody)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "x"},
			Position:     position.ToLSPPosition(),
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	search := NewSearchWithoutLog()
	return search.FindHoverInformation("x", &params, &state.state)
}

func TestProjectState_FindHoverInformation(t *testing.T) {
	state := NewTestState()
	state.registerDoc(
		"x",
		`int value = 1;
		fn void main() {
			char value = 3;
		}
	`)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "x"},
			Position: protocol.Position{
				Line:      2,
				Character: 9,
			},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	search := NewSearchWithoutLog()
	hover := search.FindHoverInformation("x", &params, &state.state)

	expectedHover := option.Some(protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "char value",
		},
	})
	assert.Equal(t, expectedHover, hover)
}

func TestProjectState_FindHoverInformationFromDifferentFile(t *testing.T) {
	t.Skip()
	state := NewTestState()
	state.registerDoc(
		"x",
		`module a;
		fn void main() {
			importedMethod();
		}
	`)

	state.registerDoc(
		"y", `
		module a;
		fn void importedMethod() {}
		`)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "x"},
			Position:     protocol.Position{Line: 3, Character: 8},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	search := NewSearchWithoutLog()
	hover := search.FindHoverInformation("x", &params, &state.state)

	expectedHover := option.Some(protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "void importedMethod()",
		},
	})
	assert.Equal(t, expectedHover, hover)
}

func TestProjectState_FindHoverInformation_displays_generic_type_arguments(t *testing.T) {
	t.Run("HashMap variable keeps generic args", func(t *testing.T) {
		state := NewTestState()
		hover := hoverFromBody(&state, `
			module app;
			fn void main() {
				HashMap{String, Feature} m|||ap = {};
			}
		`)

		expectedHover := option.Some(protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "HashMap{String, Feature} map",
			},
		})

		assert.Equal(t, expectedHover, hover)
	})

	t.Run("List variable keeps generic args", func(t *testing.T) {
		state := NewTestState()
		hover := hoverFromBody(&state, `
			module app;
			fn void main() {
				List{int} it|||ems = {};
			}
		`)

		expectedHover := option.Some(protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "List{int} items",
			},
		})

		assert.Equal(t, expectedHover, hover)
	})

	t.Run("Nested generics are rendered recursively", func(t *testing.T) {
		state := NewTestState()
		hover := hoverFromBody(&state, `
			module app;
			fn void main() {
				HashMap{String, List{int}} ne|||sted = {};
			}
		`)

		expectedHover := option.Some(protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "HashMap{String, List{int}} nested",
			},
		})

		assert.Equal(t, expectedHover, hover)
	})
}
