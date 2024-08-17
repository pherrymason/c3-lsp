package search

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/context"
	protocol_utils "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func filterOutKeywordSuggestions(completionList []protocol.CompletionItem) []protocol.CompletionItem {
	filteredCompletionList := []protocol.CompletionItem{}
	for _, item := range completionList {
		if *item.Kind != protocol.CompletionItemKindKeyword {
			filteredCompletionList = append(filteredCompletionList, item)
		}
	}

	return filteredCompletionList
}

func Test_isCompletingAChain(t *testing.T) {
	cases := []struct {
		name                     string
		input                    string
		position                 symbols.Position
		expected                 bool
		expectedPreviousPosition symbols.Position
	}{
		{
			name:     "Writing blank line",
			input:    "\nw",
			position: buildPosition(3, 0),
			expected: false,
		},
		{
			name:     "Writing with previous sentence behind",
			input:    "w",
			position: buildPosition(2, 16),
			expected: false,
		},
		{
			name:                     "Writing a struct member (including dot)",
			input:                    "\naStruct.",
			position:                 buildPosition(3, 8),
			expected:                 true,
			expectedPreviousPosition: buildPosition(3, 6),
		},
		{
			name:                     "Writing a struct member (including dot + character)",
			input:                    "\naStruct.w",
			position:                 buildPosition(3, 9),
			expected:                 true,
			expectedPreviousPosition: buildPosition(3, 6),
		},
	}

	source := `
	int value = 1;`

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.NewDocument("test.c3", source+tt.input)
			result, previousPosition := isCompletingAChain(&doc, tt.position)

			assert.Equal(
				t,
				tt.expected,
				result,
			)
			assert.Equal(
				t,
				tt.expectedPreviousPosition,
				previousPosition,
				"Previous position should be just before last character.",
			)
		})
	}
}

func Test_isCompletingAModulePath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		position symbols.Position
		expected bool
	}{
		{
			name:     "Writing blank line",
			input:    "\nw",
			position: buildPosition(2, 1),
			expected: true,
		},
		{
			name:     "Writing single character, may be a module path",
			input:    "w",
			position: buildPosition(1, 1),
			expected: true,
		},
		{
			name:     "Writing a dot character, invalidates writing a path",
			input:    "w.",
			position: buildPosition(1, 2),
			expected: false,
		},
		{
			name:     "Writing a module name (including one :)",
			input:    "aModule:",
			position: buildPosition(1, 8),
			expected: true,
		},
		{
			name:     "Writing a module name (including both ::)",
			input:    "aModule::",
			position: buildPosition(1, 9),
			expected: true,
		},
		{
			name:     "Writing a module name (including :: + character)",
			input:    "aModule::A",
			position: buildPosition(1, 10),
			expected: true,
		},
		{
			name:     "Having a previous sentence with module path does not interfere",
			input:    "app::what=1; a.",
			position: buildPosition(1, 15),
			expected: false,
		},
		{
			name:     "Having a previous sentence with module path does not interfere 2",
			input:    "app::what=1; aModule::A",
			position: buildPosition(1, 23),
			expected: true,
		},
		{
			name:     "Having a previous sentence with module path does not interfere 3",
			input:    "app::what=1; aModule",
			position: buildPosition(1, 20),
			expected: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			doc := document.NewDocument("test.c3", tt.input)
			result, _ := isCompletingAModulePath(&doc, tt.position)

			assert.Equal(
				t,
				tt.expected,
				result,
			)
		})
	}
}

func Test_extractModulePath(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected option.Option[symbols.ModulePath]
	}{
		{
			name:     "Writing single character, may be a module path",
			input:    "w",
			expected: option.None[symbols.ModulePath](),
		},
		{
			name:     "Writing a dot character, invalidates writing a path",
			input:    "w.",
			expected: option.None[symbols.ModulePath](),
		},
		{
			name:     "Writing a module name (including one :)",
			input:    "aModule:",
			expected: option.None[symbols.ModulePath](),
		},
		{
			name:     "Writing a module name (including both ::)",
			input:    "aModule::",
			expected: option.Some[symbols.ModulePath](symbols.NewModulePathFromString("aModule")),
		},
		{
			name:     "Writing a module name (including :: + character)",
			input:    "aModule::A",
			expected: option.Some[symbols.ModulePath](symbols.NewModulePathFromString("aModule")),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExplicitModulePath(tt.input)

			assert.Equal(
				t,
				tt.expected,
				result,
			)
		})
	}
}

func TestBuildCompletionList_should_return_nil_when_cursor_is_in_literal(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()
	state.registerDoc(
		"test.c3",
		`module foo; 
		printf("main.");`,
	)

	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position:  buildPosition(2, 15),
			DocURI:    "test.c3",
			IsLiteral: true,
		},
		&state.state)

	assert.Equal(t, 0, len(completionList))
}

func TestBuildCompletionList_suggests_C3_keywords(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{input: "vo", expected: []string{"void"}},
		{input: "bo", expected: []string{"bool"}},
		{input: "ch", expected: []string{"char"}},
		{input: "do", expected: []string{"double", "do"}},
		{input: "fl", expected: []string{"float", "float16", "float128"}},
		{input: "ic", expected: []string{"ichar"}},
		{input: "in", expected: []string{"int", "int128", "inline"}},
		{input: "ipt", expected: []string{"iptr"}},
		{input: "is", expected: []string{"isz"}},
		{input: "lo", expected: []string{"long"}},
		{input: "sh", expected: []string{"short"}},
		{input: "uin", expected: []string{"uint", "uint128"}},
		{input: "ul", expected: []string{"ulong"}},
		{input: "up", expected: []string{"uptr"}},
		{input: "ush", expected: []string{"ushort"}},
		{input: "us", expected: []string{"usz", "ushort"}},
		{input: "an", expected: []string{"any", "anyfault"}},
		{input: "type", expected: []string{"typeid"}},
		{input: "ass", expected: []string{"assert"}},
		{input: "as", expected: []string{"asm", "assert"}},
		{input: "bit", expected: []string{"bitstruct"}},
		{input: "br", expected: []string{"break"}},
		{input: "ca", expected: []string{"case", "catch"}},
		{input: "con", expected: []string{"const", "continue"}},
		{input: "de", expected: []string{"def", "default", "defer"}},
		{input: "di", expected: []string{"distinct"}},
		{input: "d", expected: []string{"def", "default", "defer", "distinct", "do", "double"}},
		{input: "el", expected: []string{"else"}},
		{input: "en", expected: []string{"enum"}},
		{input: "ex", expected: []string{"extern"}},
		{input: "fa", expected: []string{"false", "fault"}},
		{input: "fo", expected: []string{"for", "foreach", "foreach_r"}},
		{input: "tl", expected: []string{"tlocal"}},
		{input: "im", expected: []string{"import"}},
		{input: "ma", expected: []string{"macro"}},
		{input: "mo", expected: []string{"module"}},
		{input: "ne", expected: []string{"nextcase"}},
		{input: "nu", expected: []string{"null"}},
		{input: "re", expected: []string{"return"}},
		{input: "sta", expected: []string{"static"}},
		{input: "str", expected: []string{"struct"}},
		{input: "sw", expected: []string{"switch"}},
		{input: "tru", expected: []string{"true"}},
		{input: "tr", expected: []string{"true", "try"}},
		{input: "un", expected: []string{"union"}},
		{input: "va", expected: []string{"var"}},
		{input: "wh", expected: []string{"while"}},
		/*
			"$alignof", "$assert", "$case", "$default",
			"$defined", "$echo", "$embed", "$exec",
			"$else", "$endfor", "$endforeach", "$endif",
			"$endswitch", "$eval", "$evaltype", "$error",
			"$extnameof", "$for", "$foreach", "$if",
			"$include", "$nameof", "$offsetof", "$qnameof",
			"$sizeof", "$stringify", "$switch", "$typefrom",
			"$typeof", "$vacount", "$vatype", "$vaconst",
			"$varef", "$vaarg", "$vaexpr", "$vasplat",*/
	}

	state := NewTestState()
	search := NewSearchWithoutLog()

	for _, tt := range cases {
		t.Run(fmt.Sprintf("Should suggest C3 keywords: %s", strings.Join(tt.expected, ", ")), func(t *testing.T) {
			state.registerDoc(
				"test.c3",
				tt.input,
			)

			position := buildPosition(1, 1) // Cursor after `<input>|`

			completionList := search.BuildCompletionList(
				context.CursorContext{
					Position: position,
					DocURI:   "test.c3",
				},
				&state.state)

			expectedMap := make(map[string]bool)
			for _, exp := range tt.expected {
				expectedMap[exp] = true
			}

			for _, item := range completionList {
				assert.Equal(t, protocol.CompletionItemKindKeyword, *item.Kind)
				if _, exists := expectedMap[item.Label]; !exists {
					t.Errorf("unexpected completion: %s", item.Label)
				} else {
					delete(expectedMap, item.Label)
				}
			}

			if len(expectedMap) > 0 {
				t.Errorf("missing expected completions: %v", expectedMap)
			}
		})
	}
}

func TestBuildCompletionList(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	t.Run("Should suggest variable names defined in module", func(t *testing.T) {
		source := `
		int variable = 3;
		int xanadu = 10;`
		expectedKind := protocol.CompletionItemKindVariable
		cases := []struct {
			input    string
			expected protocol.CompletionItem
		}{
			{"v", protocol.CompletionItem{Label: "variable", Kind: &expectedKind}},
			{"va", protocol.CompletionItem{Label: "variable", Kind: &expectedKind}},
			{"x", protocol.CompletionItem{Label: "xanadu", Kind: &expectedKind}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				state.registerDoc(
					"test.c3",
					source+"\n"+tt.input,
				)

				position := buildPosition(4, 1) // Cursor after `v|`

				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				filteredCompletionList := filterOutKeywordSuggestions(completionList)

				assert.Equal(t, 1, len(filteredCompletionList))
				assert.Equal(t, tt.expected, filteredCompletionList[0])
			})
		}
	})

	t.Run("Should suggest variable names defined in module and inside current function", func(t *testing.T) {
		source := `
		int variable = 3;
		fn void main() {
			int value = 4;`
		expectedKind := protocol.CompletionItemKindVariable
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"v", []protocol.CompletionItem{
				{Label: "value", Kind: &expectedKind},
				{Label: "variable", Kind: &expectedKind},
			}},
			{"val", []protocol.CompletionItem{
				{Label: "value", Kind: &expectedKind},
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				state.registerDoc(
					"test.c3",
					source+"\n"+tt.input+"\n}",
				)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				filteredCompletionList := filterOutKeywordSuggestions(completionList)

				assert.Equal(t, len(tt.expected), len(filteredCompletionList))
				assert.Equal(t, tt.expected, filteredCompletionList)
			})
		}
	})

	t.Run("Should suggest function names", func(t *testing.T) {
		sourceStart := `
		fn void process(){}
		fn void main() {`
		sourceEnd := `
		}`

		expectedKind := protocol.CompletionItemKindFunction
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"p", []protocol.CompletionItem{
				{Label: "process", Kind: &expectedKind},
			}},
			{"proc", []protocol.CompletionItem{
				{Label: "process", Kind: &expectedKind},
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				state.registerDoc(
					"test.c3",
					sourceStart+"\n"+tt.input+"\n"+sourceEnd,
				)
				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestBuildCompletionList_struct_suggest_all_its_members(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	expectedKind := protocol.CompletionItemKindField

	source := `struct Color { int red; int green; int blue; }
	struct Square { int width; int height; Color color; }
	fn void Square.toCircle() {}
	fn void main() {
		Square inst;
		inst.
	}`
	position := buildPosition(6, 7) // Cursor after `inst.|`

	state := NewTestState(logger)
	state.registerDoc("test.c3", source)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: position,
			DocURI:   "test.c3",
		},
		&state.state)

	assert.Equal(t, 4, len(completionList))
	assert.Equal(t, []protocol.CompletionItem{
		{Label: "color", Kind: &expectedKind},
		{Label: "height", Kind: &expectedKind},
		{
			Label: "Square.toCircle",
			Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
			TextEdit: protocol.TextEdit{
				NewText: "toCircle",
				Range:   protocol_utils.NewLSPRange(5, 7, 5, 8),
			},
		},
		{Label: "width", Kind: &expectedKind},
	}, completionList)
}

func TestBuildCompletionList_struct_suggest_members_starting_with_prefix(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	expectedKind := protocol.CompletionItemKindField

	source := `struct Color { int red; int green; int blue; }
	struct Square { int width; int height; Color color; }
	fn void main() {
		Square inst;
		inst.w
	}`
	position := buildPosition(5, 8) // Cursor after `inst.|`

	state := NewTestState(logger)
	state.registerDoc("test.c3", source)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: position,
			DocURI:   "test.c3",
		},
		&state.state)

	filteredCompletionList := filterOutKeywordSuggestions(completionList)

	assert.Equal(t, 1, len(filteredCompletionList))
	assert.Equal(t, []protocol.CompletionItem{
		{Label: "width", Kind: &expectedKind},
	},
		filteredCompletionList)
}

func TestBuildCompletionList_struct_suggest_members_of_substruct(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	expectedKind := protocol.CompletionItemKindField

	source := `
	struct Color { int red; int green; int blue; }
	struct Square { int width; int height; Color color; }
	fn uint Color.toHex() {}
	fn void main() {
		Square inst;
		inst.color.
	}`
	position := buildPosition(7, 13) // Cursor after `inst.|`

	state := NewTestState(logger)
	state.registerDoc("test.c3", source)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: position,
			DocURI:   "test.c3",
		},
		&state.state)

	assert.Equal(t, 4, len(completionList))
	assert.Equal(t, []protocol.CompletionItem{
		{Label: "blue", Kind: &expectedKind},
		{
			Label: "Color.toHex",
			Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
			TextEdit: protocol.TextEdit{
				NewText: "toHex",
				Range:   protocol_utils.NewLSPRange(6, 13, 6, 14),
			},
		},
		{Label: "green", Kind: &expectedKind},
		{Label: "red", Kind: &expectedKind},
	},
		completionList)
}

func TestBuildCompletionList_struct_suggest_members_with_prefix_of_substruct(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	expectedKind := protocol.CompletionItemKindField

	source := `
	struct Color { int red; int green; int blue; }
	struct Square { int width; int height; Color color; }
	fn uint Color.toHex() {}
	fn void main() {
		Square inst;
		inst.color.r
	}`
	position := buildPosition(7, 14) // Cursor after `inst.|`

	state := NewTestState(logger)
	state.registerDoc("test.c3", source)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: position,
			DocURI:   "test.c3",
		},
		&state.state)

	filteredCompletionList := filterOutKeywordSuggestions(completionList)

	assert.Equal(t, 1, len(filteredCompletionList))
	assert.Equal(t, []protocol.CompletionItem{
		{Label: "red", Kind: &expectedKind},
	},
		filteredCompletionList)
}

func TestBuildCompletionList_struct_suggest_method_with_prefix_of_substruct(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")

	source := `
	struct Color { int red; int green; int blue; }
	struct Square { int width; int height; Color color; }
	fn uint Color.toHex() {}
	fn void main() {
		Square inst;
		inst.color.t
	}`
	position := buildPosition(7, 14) // Cursor after `inst.|`

	state := NewTestState(logger)
	state.registerDoc("test.c3", source)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: position,
			DocURI:   "test.c3",
		},
		&state.state)
	filteredCompletionList := filterOutKeywordSuggestions(completionList)
	assert.Equal(t, 1, len(filteredCompletionList))
	assert.Equal(t, []protocol.CompletionItem{
		{
			Label: "Color.toHex",
			Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
			TextEdit: protocol.TextEdit{
				NewText: "toHex",
				Range:   protocol_utils.NewLSPRange(6, 13, 6, 14),
			},
		},
	},
		filteredCompletionList)
}

func TestBuildCompletionList_enums(t *testing.T) {
	logger := commonlog.MockLogger{}

	t.Run("Should suggest Enum type", func(t *testing.T) {
		source := `
		enum Cough { COH, COUGH, COUGHCOUGH}
		enum Color { RED, GREEN, BLUE }
`
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"Co", []protocol.CompletionItem{
				CreateCompletionItem("Color", protocol.CompletionItemKindEnum),
				CreateCompletionItem("Cough", protocol.CompletionItemKindEnum),
			}},
			{"Col", []protocol.CompletionItem{
				CreateCompletionItem("Color", protocol.CompletionItemKindEnum),
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				state := NewTestState(logger)
				state.registerDoc("test.c3", source+tt.input+`}`)

				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})

	t.Run("Should suggest Enumerable type", func(t *testing.T) {
		source := `
		enum Cough { COH, COUGH, COUGHCOUGH}
		enum Color { RED, GREEN, BLUE, COBALT }
		fn void main() {
`
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"Find enumerables starting with string",
				"CO",
				[]protocol.CompletionItem{
					CreateCompletionItem("COBALT", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGHCOUGH", protocol.CompletionItemKindEnumMember),
				}},

			{
				"Find all enum enumerables when prefixed with enum name",
				"Color.",
				[]protocol.CompletionItem{
					CreateCompletionItem("BLUE", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COBALT", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("GREEN", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("RED", protocol.CompletionItemKindEnumMember),
				}},
			{
				"Find matching enum enumerables",
				"Color.COB",
				[]protocol.CompletionItem{
					CreateCompletionItem("COBALT", protocol.CompletionItemKindEnumMember),
				},
			},
		}

		for _, tt := range cases {
			t.Run(fmt.Sprintf("Autocomplete enumerables: #%s", tt.name), func(t *testing.T) {
				state := NewTestState(logger)
				state.registerDoc("test.c3", source+tt.input+`}`)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestBuildCompletionList_faults(t *testing.T) {
	t.Run("Should suggest Fault type", func(t *testing.T) {
		source := `
		fault WindowError { COH, COUGH, COUGHCOUGH}
		fault WindowFileError { NOT_FOUND, NO_PERMISSIONS }
`
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"Wind", []protocol.CompletionItem{
				CreateCompletionItem("WindowError", protocol.CompletionItemKindEnum),
				CreateCompletionItem("WindowFileError", protocol.CompletionItemKindEnum),
			}},
			{"WindowFile", []protocol.CompletionItem{
				CreateCompletionItem("WindowFileError", protocol.CompletionItemKindEnum),
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				state := NewTestState()
				state.registerDoc("test.c3", source+tt.input+`}`)
				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})

	t.Run("Should suggest Fault constant type", func(t *testing.T) {
		source := `
		fault WindowError { COH, COUGH, COUGHCOUGH}
		fault WindowFileError { NOT_FOUND, NO_PERMISSIONS, COULD_NOT_CREATE }
		fn void main() {
`
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"Find constants starting with string",
				"CO",
				[]protocol.CompletionItem{
					CreateCompletionItem("COH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGHCOUGH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COULD_NOT_CREATE", protocol.CompletionItemKindEnumMember),
				}},

			{
				"Find all fault constants when prefixed with fault name",
				"WindowError.",
				[]protocol.CompletionItem{
					CreateCompletionItem("COH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGH", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COUGHCOUGH", protocol.CompletionItemKindEnumMember),
				}},
			{
				"Find matching fault constants",
				"WindowFileError.NOT",
				[]protocol.CompletionItem{
					CreateCompletionItem("NOT_FOUND", protocol.CompletionItemKindEnumMember),
				},
			},
		}

		for _, tt := range cases {
			t.Run(fmt.Sprintf("Autocomplete contants: #%s", tt.name), func(t *testing.T) {
				state := NewTestState()
				state.registerDoc("test.c3", source+tt.input+`}`)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestBuildCompletionList_modules(t *testing.T) {
	//parser := createParser()
	//language := NewProjectState(commonlog.MockLogger{}, option.Some("dummy"), false)

	t.Run("Should suggest module names present in same document", func(t *testing.T) {
		cases := []struct {
			source   string
			position symbols.Position
			expected []protocol.CompletionItem
			skip     bool
		}{
			{
				`
				module app;
				int version = 1;
				a
				`,
				buildPosition(4, 5), // Cursor at `a|`
				[]protocol.CompletionItem{{
					Label:  "app",
					Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
					Detail: cast.ToPtr("Module"),
					TextEdit: protocol.TextEdit{
						NewText: "app",
						Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
					},
				},
				},
				true,
			},
			{
				`
				module app;
				int version = 1;
				module app::foo;

				app::`,
				buildPosition(6, 9), // Cursor at `a|`
				[]protocol.CompletionItem{
					{
						Label:  "app::foo",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::foo",
							Range: protocol.Range{
								Start: protocol.Position{Line: 5, Character: 4},
								End:   protocol.Position{Line: 5, Character: 9},
							},
						},
					},
					CreateCompletionItem("version", protocol.CompletionItemKindVariable),
				},
				false,
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				if tt.skip {
					t.Skip()
				}
				state := NewTestState()
				state.registerDoc("test.c3", tt.source)

				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: tt.position,
						DocURI:   "test.c3",
					},
					&state.state)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})

	t.Run("Should suggest module names loaded in scope", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc(
			"app_window.c3",
			`module app::window;

			module app::window::errors;
			`,
		)

		cases := []struct {
			source   string
			position symbols.Position
			expected []protocol.CompletionItem
			skip     bool
		}{
			{
				`
				module app;
				int version = 1;
				a
				`,
				buildPosition(4, 5), // Cursor at `a|`
				[]protocol.CompletionItem{
					{
						Label:  "app",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app",
							Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
						},
					},
					{
						Label:  "app::window",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window",
							Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
						},
					},
					{
						Label:  "app::window::errors",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window::errors",
							Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
						},
					},
				},
				false,
			},
			{
				`
				module app;
				int version = 1;
				module app::foo;
				
				app::`,
				buildPosition(6, 9), // Cursor at `a|`
				[]protocol.CompletionItem{
					{
						Label:  "app::foo",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::foo",
							Range:   protocol_utils.NewLSPRange(5, 4, 5, 9),
						},
					},
					{
						Label:  "app::window",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window",
							Range:   protocol_utils.NewLSPRange(5, 4, 5, 9),
						},
					},
					{
						Label:  "app::window::errors",
						Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
						Detail: cast.ToPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window::errors",
							Range:   protocol_utils.NewLSPRange(5, 4, 5, 9),
						},
					},
					CreateCompletionItem("version", protocol.CompletionItemKindVariable),
				},
				false,
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {
				if tt.skip {
					t.Skip()
				}

				//state := NewTestState()
				state.registerDoc("app.c3", tt.source)
				search := NewSearchWithoutLog()
				completionList := search.BuildCompletionList(
					context.CursorContext{
						Position: tt.position,
						DocURI:   "app.c3",
					},
					&state.state)

				filteredCompletionList := filterOutKeywordSuggestions(completionList)

				assert.Equal(t, len(tt.expected), len(filteredCompletionList), "Different items to suggest")
				assert.Equal(t, tt.expected, filteredCompletionList)
			})
		}
	})
}

func TestBuildCompletionList_interfaces(t *testing.T) {
	t.Run("should complete interface name", func(t *testing.T) {

		//doc := state.GetDoc("app.c3")
		//completionList := state.language.BuildCompletionList(&doc, buildPosition(5, 18))

		state := NewTestState()
		state.registerDoc(
			"app.c3",
			`interface EmulatorConsole
		{
			fn void run();
		}
		struct Emu (Emul){}
		`)
		search := NewSearchWithoutLog()
		completionList := search.BuildCompletionList(
			context.CursorContext{
				Position: buildPosition(5, 18),
				DocURI:   "app.c3",
			},
			&state.state)

		assert.Equal(t, 1, len(completionList), "Different items to suggest")
		assert.Equal(
			t,
			[]protocol.CompletionItem{
				{
					Label: "EmulatorConsole",
					Kind:  cast.ToPtr(protocol.CompletionItemKindInterface),
				},
			},
			completionList,
		)
	})
}

func CreateCompletionItem(label string, kind protocol.CompletionItemKind) protocol.CompletionItem {
	return protocol.CompletionItem{Label: label, Kind: &kind}
}

func TestBuildCompletionList_should_resolve_(t *testing.T) {
	state := NewTestState()
	state.registerDoc(
		"app.c3",
		`module app;
		import my::io;
		fn void main() {
			io::
		}`)

	state.registerDoc(
		"my.c3",
		`module my::io;
				int suggestion = 10;
				`,
	)
	state.registerDoc(
		"trap.c3",
		`module invalid::io;
		int invalidSuggestion = 10;
		`,
	)

	search := NewSearchWithoutLog()
	completionList := search.BuildCompletionList(
		context.CursorContext{
			Position: buildPosition(4, 7),
			DocURI:   "app.c3",
		},
		&state.state)

	assert.Equal(t, 1, len(completionList), "Wrong number of items to suggest")
	assert.Equal(
		t,
		[]protocol.CompletionItem{
			{
				Label: "suggestion",
				Kind:  cast.ToPtr(protocol.CompletionItemKindVariable),
			},
		},
		completionList,
	)
}
