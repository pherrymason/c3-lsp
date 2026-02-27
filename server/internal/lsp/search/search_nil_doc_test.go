package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestFindSymbolDeclarationInWorkspace_ReturnsNoneWhenDocMissing(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	result := search.FindSymbolDeclarationInWorkspace("missing.c3", symbols.NewPosition(0, 0), &state.state)
	if result.IsSome() {
		t.Fatalf("expected no symbol for missing document")
	}
}

func TestFindHoverInformation_ReturnsNoneWhenDocMissing(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	hover := search.FindHoverInformation("missing.c3", &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "missing.c3"},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}, &state.state)

	if hover.IsSome() {
		t.Fatalf("expected no hover for missing document")
	}
}
