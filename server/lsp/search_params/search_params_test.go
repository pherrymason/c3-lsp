package search_params

import (
	"fmt"
	"testing"

	d "github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/option"
	"github.com/stretchr/testify/assert"
)

func buildPosition(line uint, character uint) indexables.Position {
	return indexables.Position{Line: line - 1, Character: character}
}

type MockParsedModules struct {
	expectedModule string
}

func (m *MockParsedModules) SetModuleForPosition(moduleName string) {
	m.expectedModule = moduleName
}
func (m MockParsedModules) FindModuleInCursorPosition(cursorPosition idx.Position) string {
	return m.expectedModule
}

// ------------------

func TestSearchParams_NewSearchParamsFromPosition_finds_symbol_at_cursor_position(t *testing.T) {
	sourceCode := "module system; int emu;"
	doc := d.NewDocument("filename", sourceCode)
	parsedModules := MockParsedModules{}
	parsedModules.SetModuleForPosition("system")

	sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, buildPosition(1, 20))

	assert.Equal(t, "emu", sp.symbol)
	assert.Equal(t, idx.NewRange(0, 19, 0, 22), sp.symbolRange)
	assert.Equal(t, option.Some("filename"), sp.docId)
	assert.Equal(t, "system", sp.symbolModulePath.GetName())
	assert.Equal(t, true, sp.continueOnModules)
	assert.Equal(t, 0, len(sp.parentAccessPath))
	assert.Equal(t, make(TrackedModules), sp.trackedModules)
	assert.Equal(t, false, sp.moduleSpecified)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_all_parent_symbols(t *testing.T) {
	sourceCode := `// This blank line is intended.
system.cpu.init();`
	doc := d.NewDocument("filename", sourceCode)
	parsedModules := &MockParsedModules{}
	parsedModules.SetModuleForPosition("system")

	// Cursor at "i|init"
	sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, buildPosition(2, 12))

	assert.Equal(t, "init", sp.symbol)
	assert.Equal(t, idx.NewRange(1, 11, 1, 15), sp.symbolRange)
	assert.Equal(t, option.Some("filename"), sp.docId)
	assert.Equal(t, "system", sp.symbolModulePath.GetName())
	assert.Equal(t, true, sp.continueOnModules)
	assert.Equal(t, []d.Token{
		d.NewToken("system", idx.NewRange(1, 0, 1, 6)),
		d.NewToken("cpu", idx.NewRange(1, 7, 1, 10)),
	}, sp.parentAccessPath)
	assert.Equal(t, make(TrackedModules), sp.trackedModules)
	assert.Equal(t, false, sp.moduleSpecified)
}

func TestSearchParams_NewSearchParamsFromPosition_finds_full_module_path(t *testing.T) {
	cases := []struct {
		source              string
		position            indexables.Position
		expectedModule      indexables.ModulePath
		expectedSymbol      string
		expectedSymbolRange indexables.Range
	}{
		{
			`// This blank line is intended.
			system::cpu::value;`,
			buildPosition(2, 17),
			indexables.NewModulePath([]string{"system", "cpu"}),
			"value",
			idx.NewRange(1, 16, 1, 21),
		},
		{
			`// This blank line is intended.
mybar.color = foo::bar::DEFAULT_BAR_COLOR;`,
			buildPosition(2, 25),
			indexables.NewModulePath([]string{"foo", "bar"}),
			"DEFAULT_BAR_COLOR",
			idx.NewRange(1, 24, 1, 41),
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			doc := d.NewDocument("filename", tt.source)
			parsedModules := &MockParsedModules{}
			parsedModules.SetModuleForPosition("system")

			sp := BuildSearchBySymbolUnderCursor(&doc, parsedModules, tt.position)

			assert.Equal(t, tt.expectedSymbol, sp.symbol)
			assert.Equal(t, tt.expectedSymbolRange, sp.symbolRange)
			assert.Equal(t, option.Some("filename"), sp.docId)
			//assert.Equal(t, "system", sp.symbolModulePath.GetName())
			assert.Equal(t, true, sp.moduleSpecified)
			assert.Equal(t, 0, len(sp.parentAccessPath))
			assert.Equal(t, tt.expectedModule, sp.symbolModulePath)

		})
	}
}
