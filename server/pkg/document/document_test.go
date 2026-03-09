package document

import (
	"errors"
	"testing"

	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDocument_GetSymbolRangeAtIndex_does_not_find_symbol(t *testing.T) {
	doc := NewDocument("x", "a document")
	_, _, error := doc.getWordIndexLimits(1)

	assert.Equal(t, errors.New("no symbol at position"), error)
}

func TestWordInIndex(t *testing.T) {
	cases := []struct {
		name     string
		expected string
		position int
	}{
		{"start of doc", "hello", 1},
		{"word", "expected", 14},
		{"word with underscore", "bye_bye", 24},
	}

	source := "hello this is expected bye_bye"
	doc := NewDocument("x", source)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := doc.getWordIndexLimits(tt.position)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			word := doc.SourceCode.Text[start : end+1]

			assert.Equal(t, tt.expected, word)
		})
	}
}

func TestWordInIndex_handles_multibyte_unicode_boundaries(t *testing.T) {
	doc := NewDocument("x", "héllo world")

	start, end, err := doc.getWordIndexLimits(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.Equal(t, "héllo", doc.SourceCode.Text[start:end+1])
}

func TestDocument_ApplyChanges_applies_incremental_changes_deterministically(t *testing.T) {
	makeDoc := func() Document {
		return NewDocument("x", "abc\ndef\n")
	}

	changes := []interface{}{
		protocol.TextDocumentContentChangeEvent{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 1},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			Text: "BC",
		},
		protocol.TextDocumentContentChangeEvent{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 1, Character: 2},
			},
			Text: "DE",
		},
	}

	docA := makeDoc()
	docA.ApplyChanges(changes)

	docB := makeDoc()
	docB.ApplyChanges(changes)

	assert.Equal(t, "aBC\nDEf\n", docA.SourceCode.Text)
	assert.Equal(t, docA.SourceCode.Text, docB.SourceCode.Text)
}

func TestDocument_HasPointInFrontSymbol(t *testing.T) {
	cases := []struct {
		source   string
		expected bool
		position idx.Position
	}{
		{"int symbol", false, idx.Position{Line: 0, Character: 6}},
		{"int symbol", false, idx.Position{Line: 0, Character: 0}},
		{"object.symbol", true, idx.Position{Line: 0, Character: 9}},
		{"int symbol0; object.symbol", false, idx.Position{Line: 0, Character: 7}},
		{"object.symbol;int symbol0; ", true, idx.Position{Line: 0, Character: 7}},
		{"object.symbol;int symbol0; ", true, idx.Position{Line: 0, Character: 8}},
		{"object.symbol;int symbol0; ", false, idx.Position{Line: 0, Character: 21}},
	}

	for _, tt := range cases {
		t.Run("HasPointInFront", func(t *testing.T) {
			doc := NewDocument("x", tt.source)
			hasIt := doc.HasPointInFrontSymbol(tt.position)

			assert.Equal(t, tt.expected, hasIt)
		})
	}
}

func TestDocument_ParentSymbolInPosition(t *testing.T) {
	cases := []struct {
		source   string
		expected string
		position idx.Position
	}{
		{"int symbol", "", idx.Position{Line: 0, Character: 6}},
		{"int symbol", "", idx.Position{Line: 0, Character: 0}},
		{"object.symbol", "object", idx.Position{Line: 0, Character: 9}},
		{"int symbol0; object.symbol", "", idx.Position{Line: 0, Character: 7}},
		{"object.symbol;int symbol0; ", "object", idx.Position{Line: 0, Character: 7}},
		{"object.symbol;int symbol0; ", "object", idx.Position{Line: 0, Character: 8}},
		{"object.symbol;int symbol0; ", "", idx.Position{Line: 0, Character: 21}},
		{`object
				.symbol`, "object", idx.Position{Line: 1, Character: 6}},
		//{`object.
		//		symbol`, "object", idx.Position{1, 6}},
	}

	for _, tt := range cases {
		t.Run("HasPointInFront", func(t *testing.T) {
			doc := NewDocument("x", tt.source)
			parentSymbol, err := doc.ParentSymbolInPosition(tt.position)

			assert.Equal(t, tt.expected, parentSymbol.Text())
			if tt.expected != "" && err != nil {
				t.Fatal(err)
			}
		})
	}
}
