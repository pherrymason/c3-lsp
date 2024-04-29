package document

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDocument_GetSymbolRangeAtIndex_does_not_find_symbol(t *testing.T) {
	doc := NewDocument("x", "x", "a document")
	_, _, error := doc.getSymbolRangeAtIndex(1)

	assert.Equal(t, errors.New("No symbol at position"), error)
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
	doc := NewDocument("x", "x", source)
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			word, _ := doc.symbolInIndex(tt.position)

			assert.Equal(t, tt.expected, word)
		})
	}
}

func TestDocument_HasPointInFrontSymbol(t *testing.T) {
	cases := []struct {
		source   string
		expected bool
		position protocol.Position
	}{
		{"int symbol", false, protocol.Position{0, 6}},
		{"int symbol", false, protocol.Position{0, 0}},
		{"object.symbol", true, protocol.Position{0, 9}},
		{"int symbol0; object.symbol", false, protocol.Position{0, 7}},
		{"object.symbol;int symbol0; ", true, protocol.Position{0, 7}},
		{"object.symbol;int symbol0; ", true, protocol.Position{0, 8}},
		{"object.symbol;int symbol0; ", false, protocol.Position{0, 21}},
	}

	for _, tt := range cases {
		t.Run("HasPointInFront", func(t *testing.T) {
			doc := NewDocument("x", "x", tt.source)
			hasIt := doc.HasPointInFrontSymbol(tt.position)

			assert.Equal(t, tt.expected, hasIt)
		})
	}
}

func TestDocument_ParentSymbolInPosition(t *testing.T) {
	cases := []struct {
		source   string
		expected string
		position protocol.Position
	}{
		{"int symbol", "", protocol.Position{0, 6}},
		{"int symbol", "", protocol.Position{0, 0}},
		{"object.symbol", "object", protocol.Position{0, 9}},
		{"int symbol0; object.symbol", "", protocol.Position{0, 7}},
		{"object.symbol;int symbol0; ", "object", protocol.Position{0, 7}},
		{"object.symbol;int symbol0; ", "object", protocol.Position{0, 8}},
		{"object.symbol;int symbol0; ", "", protocol.Position{0, 21}},
		{`object
				.symbol`, "object", protocol.Position{1, 6}},
		//{`object.
		//		symbol`, "object", protocol.Position{1, 6}},
	}

	for _, tt := range cases {
		t.Run("HasPointInFront", func(t *testing.T) {
			doc := NewDocument("x", "x", tt.source)
			parentSymbol, err := doc.ParentSymbolInPosition(tt.position)

			assert.Equal(t, tt.expected, parentSymbol)
			if tt.expected != "" && err != nil {
				t.Fatalf(fmt.Sprint(err))
			}
		})
	}
}
