package language

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSearchParams_NewSearchParamsFromPosition_finds_symbol_at_cursor_position(t *testing.T) {
	sourceCode := "int emu;"
	doc := document.NewDocument("filename", "module", sourceCode)

	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(1, 5))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol:    Token{token: "emu", position: buildPosition(1, 5)},
			docId:             "filename",
			scopeMode:         InScope,
			continueOnModules: true,
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
			docId:             "filename",
			scopeMode:         InScope,
			continueOnModules: true,
		},
		sp,
	)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_full_module_path(t *testing.T) {
	cases := []struct {
		source         string
		position       protocol.Position
		expectedModule indexables.ModulePath
		expectedToken  string
	}{
		{
			`// This blank line is intended.
			system::cpu::value;`,
			buildPosition(2, 17),
			indexables.NewModulePath([]string{"system", "cpu"}),
			"value",
		},
		{
			`// This blank line is intended.
mybar.color = foo::bar::DEFAULT_BAR_COLOR;`,
			buildPosition(2, 25),
			indexables.NewModulePath([]string{"foo", "bar"}),
			"DEFAULT_BAR_COLOR",
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			doc := document.NewDocument("filename", "module", tt.source)
			sp, err := NewSearchParamsFromPosition(&doc, tt.position)

			assert.Nil(t, err)
			assert.Equal(
				t,
				SearchParams{
					selectedSymbol:    Token{token: tt.expectedToken, position: tt.position},
					docId:             "filename",
					modulePath:        tt.expectedModule,
					scopeMode:         InScope,
					continueOnModules: true,
				},
				sp,
			)
		})
	}
}
