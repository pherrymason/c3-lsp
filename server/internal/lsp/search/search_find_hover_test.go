package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
