package sourcecode

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
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

			// position at e|mu
			pos := uint(len(tt.source) - 1)
			result := sc.SymbolInPosition(symbols.NewPosition(0, pos))

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

func Test_SourceCode_SymbolInPosition_finds_simple_symbol(t *testing.T) {
	text := "int emu;"

	sc := NewSourceCode(text)

	// position at e|mu
	result := sc.SymbolInPosition(symbols.NewPosition(0, 5))

	expectedWord := NewWord("emu", symbols.NewRange(0, 4, 0, 7))
	assert.Equal(t, expectedWord, result)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_access_path(t *testing.T) {
	text := `// This blank line is intended.
	system.cpu.init();`

	sc := NewSourceCode(text)

	// position at i|nit
	result := sc.SymbolInPosition(symbols.NewPosition(1, 13))

	assert.Equal(t, "init", result.text)
	assert.Equal(t, symbols.NewRange(1, 12, 1, 16), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_access_path_and_method_call(t *testing.T) {
	text := `// This blank line is intended.
	system.cpu().init;`

	sc := NewSourceCode(text)

	// position at i|nit
	result := sc.SymbolInPosition(symbols.NewPosition(1, 15))

	assert.Equal(t, "init", result.text)
	assert.Equal(t, symbols.NewRange(1, 14, 1, 18), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_dot_word(t *testing.T) {
	text := `// This blank line is intended.
	system.cpu.`

	sc := NewSourceCode(text)

	// position at cpu|.
	result := sc.SymbolInPosition(symbols.NewPosition(1, 11))

	assert.Equal(t, ".", result.text)
	assert.Equal(t, symbols.NewRange(1, 11, 1, 12), result.textRange)

	assert.Equal(t, "system", result.parentAccessPath[0].text)
	assert.Equal(t, symbols.NewRange(1, 1, 1, 7), result.parentAccessPath[0].textRange)

	assert.Equal(t, "cpu", result.parentAccessPath[1].text)
	assert.Equal(t, symbols.NewRange(1, 8, 1, 11), result.parentAccessPath[1].textRange)
}

func Test_SourceCode_SymbolInPosition_finds_symbol_with_single_character_symbol(t *testing.T) {
	text := `// This blank line is intended.
	system.cpu.w`

	sc := NewSourceCode(text)

	// position at cpu|.
	result := sc.SymbolInPosition(symbols.NewPosition(1, 12))

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
			sc := NewSourceCode(tt.source)

			// position at v|alue
			result := sc.SymbolInPosition(tt.position)

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
