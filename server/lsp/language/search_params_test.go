package language

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/stretchr/testify/assert"
)

func TestSearchParams_NewSearchParamsFromPosition_finds_symbol_at_cursor_position(t *testing.T) {
	sourceCode := "int emu;"
	doc := document.NewDocument("filename", "module", sourceCode)

	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(1, 5))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: Token{token: "emu", position: buildPosition(1, 5)},
			docId:          "filename",
			findMode:       InPosition,
		},
		sp,
	)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_all_parent_symbols(t *testing.T) {
	sourceCode := "system.cpu.init();"
	doc := document.NewDocument("filename", "module", sourceCode)

	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(1, 12))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: Token{token: "init", position: buildPosition(1, 12)},
			parentSymbols: []Token{
				Token{token: "cpu", position: buildPosition(1, 7)},
				Token{token: "system", position: buildPosition(1, 0)},
			},
			docId:    "filename",
			findMode: InPosition,
		},
		sp,
	)
}
