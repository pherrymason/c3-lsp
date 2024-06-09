package sourcecode

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/symbols_table"
	"github.com/stretchr/testify/assert"
)

func Test_SourceCode_SymbolInPosition_finds_symbol(t *testing.T) {
	cases := []struct {
		source        string
		expSymbol     string
		expAccessPath []string
		expModulePath []string
	}{
		{"hola", "hola", []string{}, []string{}},
		{"hola.adios", "adios", []string{"hola"}, []string{}},
		{"hola.adios.w", "w", []string{"hola", "adios"}, []string{}},
		{"hola().adios.w", "w", []string{"hola", "adios"}, []string{}},
		{"hola.adios.", ".", []string{"hola", "adios"}, []string{}},
		{"mod::hola", "hola", []string{}, []string{"mod"}},
		{"mod::hola.adios", "adios", []string{"hola"}, []string{"mod"}},
		{"mod::hola.adios.w", "w", []string{"hola", "adios"}, []string{"mod"}},
		{"mod::hola.adios.", ".", []string{"hola", "adios"}, []string{"mod"}},
		{"mod::mod2::hola.adios.", ".", []string{"hola", "adios"}, []string{"mod", "mod2"}},
	}

	for _, tt := range cases {
		t.Run(fmt.Sprintf("Test %s", tt.source), func(t *testing.T) {
			sc := NewSourceCode(tt.source)
			unitModule := symbols_table.UnitModules{}

			// position at e|mu
			pos := uint(len(tt.source) - 1)
			result := sc.SymbolInPosition(symbols.NewPosition(0, pos), &unitModule)

			assert.Equal(t, tt.expSymbol, result.Text())

			list := []string{}
			for _, m := range result.parentAccessPath {
				list = append(list, m.Text())
			}
			assert.Equal(t, tt.expAccessPath, list)

			list = []string{}
			for _, m := range result.ModulePath() {
				list = append(list, m.Text())
			}
			assert.Equal(t, tt.expModulePath, list)
		})
	}
}

func Test_SourceCode_SymbolInPosition_finds_simple_symbol_trap_with_parenthesis(t *testing.T) {
	text := "something(value);"
	unitModule := symbols_table.UnitModules{}
	sc := NewSourceCode(text)

	// position at e|mu
	result := sc.SymbolInPosition(symbols.NewPosition(0, 11), &unitModule)

	expectedWord := NewWord("value", symbols.NewRange(0, 10, 0, 15))
	assert.Equal(t, expectedWord, result)
}

func Test_SourceCode_SymbolInPosition_finds_simple_symbol(t *testing.T) {
	unitModule := symbols_table.UnitModules{}
	text := "int emu;"
	sc := NewSourceCode(text)

	// position at e|mu
	result := sc.SymbolInPosition(symbols.NewPosition(0, 5), &unitModule)

	expectedWord := NewWord("emu", symbols.NewRange(0, 4, 0, 7))
	assert.Equal(t, expectedWord, result)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_access_path(t *testing.T) {
	unitModule := symbols_table.UnitModules{}
	text := `// This blank line is intended.
	system.cpu.init();`

	sc := NewSourceCode(text)

	// position at i|nit
	result := sc.SymbolInPosition(symbols.NewPosition(1, 13), &unitModule)

	assert.Equal(t, "init", result.text)
	assert.Equal(t, symbols.NewRange(1, 12, 1, 16), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_access_path_and_method_call(t *testing.T) {
	unitModule := symbols_table.UnitModules{}
	text := `// This blank line is intended.
	system.cpu().init;`

	sc := NewSourceCode(text)

	// position at i|nit
	result := sc.SymbolInPosition(symbols.NewPosition(1, 15), &unitModule)

	assert.Equal(t, "init", result.text)
	assert.Equal(t, symbols.NewRange(1, 14, 1, 18), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_dot_word(t *testing.T) {
	unitModule := symbols_table.UnitModules{}
	text := `// This blank line is intended.
	system.cpu.`

	sc := NewSourceCode(text)

	// position at cpu|.
	result := sc.SymbolInPosition(symbols.NewPosition(1, 11), &unitModule)

	assert.Equal(t, ".", result.text)
	assert.Equal(t, symbols.NewRange(1, 11, 1, 12), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_single_character_symbol(t *testing.T) {
	unitModule := symbols_table.UnitModules{}
	text := `// This blank line is intended.
	system.cpu.w`

	sc := NewSourceCode(text)

	// position at cpu|.
	result := sc.SymbolInPosition(symbols.NewPosition(1, 12), &unitModule)

	assert.Equal(t, "w", result.text)
	assert.Equal(t, symbols.NewRange(1, 12, 1, 13), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_module_path(t *testing.T) {
	cases := []struct {
		source               string
		position             symbols.Position
		expectedSymbol       string
		expectedModule       string
		expectedSymbolRange  symbols.Range
		expectedModuleRanges []symbols.Range
	}{
		{
			`// This blank line is intended.
	system::cpu::value;`,
			symbols.NewPosition(1, 15),
			"value",
			"system::cpu",
			symbols.NewRange(1, 14, 1, 19),
			[]symbols.Range{
				symbols.NewRange(1, 1, 1, 7),
				symbols.NewRange(1, 9, 1, 12),
			},
		},
		{
			`// This blank line is intended.
			another = sentence; system::cpu::value;`,
			symbols.NewPosition(1, 37),
			"value",
			"system::cpu",
			symbols.NewRange(1, 36, 1, 41),
			[]symbols.Range{
				symbols.NewRange(1, 23, 1, 29),
				symbols.NewRange(1, 31, 1, 34),
			},
		},
		{
			`// This blank line is intended.
			another = system::cpu::value;`,
			symbols.NewPosition(1, 27),
			"value",
			"system::cpu",
			symbols.NewRange(1, 26, 1, 31),
			[]symbols.Range{
				symbols.NewRange(1, 13, 1, 19),
				symbols.NewRange(1, 21, 1, 24),
			},
		},
		{
			`// This blank line is intended.
			system::cpu::`,
			symbols.NewPosition(1, 15),
			":",
			"system::cpu",
			symbols.NewRange(1, 15, 1, 16),
			[]symbols.Range{
				symbols.NewRange(1, 3, 1, 9),
				symbols.NewRange(1, 11, 1, 14),
			},
		},
	}

	for i, tt := range cases {
		t.Run(fmt.Sprintf("Test %d", i), func(t *testing.T) {
			unitModule := symbols_table.UnitModules{}
			sc := NewSourceCode(tt.source)

			// position at v|alue
			result := sc.SymbolInPosition(tt.position, &unitModule)

			assert.Equal(t, tt.expectedSymbol, result.text)
			assert.Equal(t, tt.expectedSymbolRange, result.textRange)
			assert.Equal(t, true, len(result.modulePath) > 0)

			assert.Equal(t, "system", result.modulePath[0].text)
			assert.Equal(t, tt.expectedModuleRanges[0], result.modulePath[0].textRange)

			assert.Equal(t, "cpu", result.modulePath[1].text)
			assert.Equal(t, tt.expectedModuleRanges[1], result.modulePath[1].textRange)
		})
	}
}

func Test_SourceCode_SymbolInPosition_should_resolve_full_module_paths(t *testing.T) {

	// Resolves full name of module, even if sentence uses short name
	module := symbols.NewModule("file", "file", symbols.NewRange(0, 0, 0, 0), symbols.NewRange(0, 0, 1, 13))
	module.AddImports([]string{"std::io"})

	unitModule := symbols_table.NewParsedModules("file")
	unitModule.RegisterModule(module)
	source := `import std::io;
	io::printf()`
	position := symbols.NewPosition(1, 8)

	sc := NewSourceCode(source)

	// position at v|alue
	result := sc.SymbolInPosition(position, &unitModule)

	assert.Equal(t, "printf", result.text)
	assert.Equal(t, symbols.NewRange(1, 5, 1, 11), result.textRange)
	assert.Equal(t, true, len(result.modulePath) == 1)

	assert.Equal(t, "io", result.modulePath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 3), result.modulePath[0].textRange)

	assert.Equal(t, true, len(result.resolvedModulePath) == 2)
	assert.Equal(t, "std", result.resolvedModulePath[0])
	assert.Equal(t, "io", result.resolvedModulePath[1])
}
