package search_params

import (
	"fmt"
	"testing"

	d "github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/document/sourcecode"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
	"github.com/pherrymason/c3-lsp/option"
	"github.com/stretchr/testify/assert"
)

func buildPosition(line uint, character uint) idx.Position {
	return idx.Position{Line: line - 1, Character: character}
}

// ------------------

func TestSearchParams_BuildSearchBySymbolUnderCursor_finds_symbol_at_cursor_position(t *testing.T) {
	sourceCode := "module system; int emu;"
	doc := d.NewDocument("filename", sourceCode)
	parsedModules := symbols_table.NewParsedModules("filename")
	parsedModules.RegisterModule(idx.NewModule("system", "filename", idx.NewRange(0, 0, 0, 0), idx.NewRange(0, 0, 0, 23)))

	// position at int e|mu
	sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, buildPosition(1, 20))

	assert.Equal(t, "emu", sp.word.Text())
	assert.Equal(t, idx.NewRange(0, 19, 0, 22), sp.word.TextRange())
	assert.Equal(t, option.Some("filename"), sp.limitToDocId)
	assert.Equal(t, "system", sp.moduleInCursor.GetName())

	assert.Equal(t, 0, len(sp.word.AccessPath()))
	assert.Equal(t, make(TrackedModules), sp.trackedModules)
	assert.Equal(t, false, sp.word.HasModulePath())
}

func TestSearchParams_BuildSearchBySymbolUnderCursor_finds_all_parent_symbols(t *testing.T) {
	sourceCode := `// This blank line is intended.
system.cpu.init();`
	doc := d.NewDocument("filename", sourceCode)
	parsedModules := symbols_table.NewParsedModules("filename")
	parsedModules.RegisterModule(idx.NewModule("system", "filename", idx.NewRange(0, 0, 0, 0), idx.NewRange(0, 0, 1, 18)))

	// Cursor at "i|init"
	sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, buildPosition(2, 12))

	assert.Equal(t, "init", sp.Symbol())
	assert.Equal(t, idx.NewRange(1, 11, 1, 15), sp.word.TextRange())
	assert.Equal(t, option.Some("filename"), sp.limitToDocId)
	assert.Equal(t, "system", sp.moduleInCursor.GetName())

	assert.Equal(t, []sourcecode.Word{
		sourcecode.NewWord("system", idx.NewRange(1, 0, 1, 6)),
		sourcecode.NewWord("cpu", idx.NewRange(1, 7, 1, 10)),
	}, sp.word.AccessPath())
	assert.Equal(t, make(TrackedModules), sp.trackedModules)
	assert.Equal(t, false, sp.word.HasModulePath())
}

func TestSearchParams_BuildSearchBySymbolUnderCursor_finds_full_module_path(t *testing.T) {
	cases := []struct {
		source   string
		position idx.Position
		//expectedContextModule idx.ModulePath
		expectedSymbol       string
		expectedSymbolRange  idx.Range
		expectedSymbolModule option.Option[string]
	}{
		{
			`// This blank line is intended.
			system::cpu::value;`,
			buildPosition(2, 17), // Cursor position cpu::v|alue
			"value",
			idx.NewRange(1, 16, 1, 21),
			option.Some("system::cpu"),
		},
		{
			`// This blank line is intended.
			mybar.color = foo::bar::DEFAULT_BAR_COLOR;`,
			buildPosition(2, 28), // Cursor position foo::bar::D|EFAULT_BAR_COLOR
			"DEFAULT_BAR_COLOR",
			idx.NewRange(1, 27, 1, 44),
			option.Some("foo::bar"),
		},
		{
			`// This blank line is intended.
			mybar.color = foo::bar::DEFAULT_BAR_COLOR;`,
			buildPosition(2, 26), // Cursor position foo::bar:|:DEFAULT_BAR_COLOR
			":",
			idx.NewRange(1, 26, 1, 27),
			option.Some("foo::bar"),
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			doc := d.NewDocument("filename", tt.source)
			parsedModules := symbols_table.NewParsedModules("filename")
			parsedModules.RegisterModule(idx.NewModule("system", "filename", idx.NewRange(0, 0, 0, 0), idx.NewRange(0, 0, 10, 30)))

			sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, tt.position)

			assert.Equal(t, tt.expectedSymbol, sp.Symbol())
			assert.Equal(t, tt.expectedSymbolRange, sp.word.TextRange())
			assert.Equal(t, option.Some("filename"), sp.limitToDocId)
			assert.Equal(t, "system", sp.moduleInCursor.GetName())
			assert.Equal(t, true, sp.HasModuleSpecified())
			assert.Equal(t, false, sp.word.HasAccessPath())
		})
	}
}
