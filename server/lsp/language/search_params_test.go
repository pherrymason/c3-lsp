package language

import (
	"fmt"
	"testing"

	d "github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSearchParams_NewSearchParamsFromPosition_finds_symbol_at_cursor_position(t *testing.T) {
	sourceCode := "int emu;"
	doc := d.NewDocument("filename", sourceCode)

	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(1, 5))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: d.Token{
				Token: "emu",
				//position: buildPosition(1, 5),
				TokenRange: idx.NewRange(0, 4, 0, 7),
			},
			docId:             "filename",
			scopeMode:         InScope,
			continueOnModules: true,
			trackedModules:    make(map[string]int),
		},
		sp,
	)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_all_parent_symbols(t *testing.T) {
	sourceCode := `// This blank line is intended.
system.cpu.init();`
	doc := d.NewDocument("filename", sourceCode)
	// Cursor at "i|init"
	sp, err := NewSearchParamsFromPosition(&doc, buildPosition(2, 12))

	assert.Nil(t, err)
	assert.Equal(
		t,
		SearchParams{
			selectedSymbol: d.NewToken("init", idx.NewRange(1, 11, 1, 15)),
			parentSymbols: []d.Token{
				d.NewToken("system", idx.NewRange(1, 0, 1, 6)),
				d.NewToken("cpu", idx.NewRange(1, 7, 1, 10)),
			},
			docId:             "filename",
			scopeMode:         InScope,
			continueOnModules: true,
			trackedModules:    make(map[string]int),
		},
		sp,
	)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_full_module_path(t *testing.T) {
	cases := []struct {
		source         string
		position       protocol.Position
		expectedModule indexables.ModulePath
		expectedToken  d.Token
	}{
		{
			`// This blank line is intended.
			system::cpu::value;`,
			buildPosition(2, 17),
			indexables.NewModulePath([]string{"system", "cpu"}),
			d.NewToken("value", idx.NewRange(1, 16, 1, 21)),
		},
		{
			`// This blank line is intended.
mybar.color = foo::bar::DEFAULT_BAR_COLOR;`,
			buildPosition(2, 25),
			indexables.NewModulePath([]string{"foo", "bar"}),
			d.NewToken("DEFAULT_BAR_COLOR", idx.NewRange(1, 24, 1, 41)),
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			doc := d.NewDocument("filename", tt.source)
			sp, err := NewSearchParamsFromPosition(&doc, tt.position)

			assert.Nil(t, err)
			assert.Equal(
				t,
				SearchParams{
					selectedSymbol:    tt.expectedToken,
					docId:             "filename",
					modulePath:        tt.expectedModule,
					scopeMode:         InScope,
					continueOnModules: true,
					trackedModules:    make(map[string]int),
				},
				sp,
			)
		})
	}
}
