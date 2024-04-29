package language

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
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
	sourceCode := `// This blank line is intended.
system.cpu.init();`
	doc := document.NewDocument("filename", "module", sourceCode)
	// Cursor at "i|init"
	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(2, 12))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: Token{token: "init", position: buildPosition(2, 12)},
			parentSymbols: []Token{
				{token: "cpu", position: buildPosition(2, 7)},
				{token: "system", position: buildPosition(2, 0)},
			},
			docId:    "filename",
			findMode: InPosition,
		},
		sp,
	)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_full_module_path(t *testing.T) {
	sourceCode := `// This blank line is intended.
system::cpu::value;`
	doc := document.NewDocument("filename", "module", sourceCode)

	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(2, 13))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: Token{token: "value", position: buildPosition(2, 13)},
			docId:          "filename",
			modulePath:     indexables.NewModulePath([]string{"cpu", "system"}),
			findMode:       InPosition,
		},
		sp,
	)
}
