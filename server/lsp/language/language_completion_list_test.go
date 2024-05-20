package language

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/cast"
	"github.com/pherrymason/c3-lsp/lsp/document"
	protocol_utils "github.com/pherrymason/c3-lsp/lsp/protocol"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/option"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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

func TestLanguage_BuildCompletionList(t *testing.T) {
	state := NewTestState()

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

				doc := state.GetDoc("test.c3")
				position := buildPosition(4, 1) // Cursor after `v|`

				completionList := state.language.BuildCompletionList(&doc, position)

				assert.Equal(t, 1, len(completionList))
				assert.Equal(t, tt.expected, completionList[0])
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
				doc := state.GetDoc("test.c3")
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := state.language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
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
				doc := state.GetDoc("test.c3")
				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := state.language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestLanguage_BuildCompletionList_structs(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	state := NewTestState(logger)

	t.Run("Should suggest struct members", func(t *testing.T) {
		expectedKind := protocol.CompletionItemKindField
		cases := []struct {
			name           string
			source         string
			cursorPosition symbols.Position
			expected       []protocol.CompletionItem
		}{
			{
				"suggest all struct members",
				`struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn void Square.toCircle() {}
				fn void main() {
					Square inst;
					inst.
				}`,
				buildPosition(6, 10), // Cursor after `inst.|`
				[]protocol.CompletionItem{
					{Label: "color", Kind: &expectedKind},
					{Label: "height", Kind: &expectedKind},
					{Label: "width", Kind: &expectedKind},
					{Label: "toCircle", Kind: cast.CompletionItemKindPtr(protocol.CompletionItemKindMethod)},
				}},

			{
				"suggest members starting with `w`",
				`struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn void main() {
					Square inst;
					inst.w
				}`,
				buildPosition(5, 11), // Cursor after `inst.w|`
				[]protocol.CompletionItem{
					{Label: "width", Kind: &expectedKind},
				}},

			{
				"suggest members of Color",
				`
				struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn uint Color.toHex() {}
				fn void main() {
					Square inst;
					inst.color.
				}`,
				buildPosition(7, 16), // Cursor after `inst.color.|`
				[]protocol.CompletionItem{
					{Label: "blue", Kind: &expectedKind},
					{Label: "green", Kind: &expectedKind},
					{Label: "red", Kind: &expectedKind},
				}},

			{
				"suggest members of Color starting with `r`",
				`
				struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn uint Color.toHex() {}
				fn void main() {
					Square inst;
					inst.color.r
				}`,
				buildPosition(7, 17), // Cursor after `inst.color.r|`
				[]protocol.CompletionItem{
					{Label: "red", Kind: &expectedKind},
				}},
			{
				"suggest methods of Color starting with `t`",
				`
				struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn uint Color.toHex() {}
				fn void main() {
					Square inst;
					inst.color.t
				}`,
				buildPosition(7, 17), // Cursor after `inst.color.r|`
				[]protocol.CompletionItem{
					{Label: "red", Kind: &expectedKind},
				}},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				state.registerDoc("test.c3", tt.source)
				doc := state.GetDoc("test.c3")
				//position := buildPosition(7, 9+uint(len(tt.source))) // Cursor after `<input>|`

				completionList := state.language.BuildCompletionList(&doc, tt.cursorPosition)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestLanguage_BuildCompletionList_enums(t *testing.T) {
	parser := createParser()
	language := NewLanguage(commonlog.MockLogger{})

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
				doc := document.NewDocument("test.c3", source+tt.input+`}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

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
				doc := document.NewDocument("test.c3", source+tt.input+`
				}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestLanguage_BuildCompletionList_faults(t *testing.T) {
	parser := createParser()
	language := NewLanguage(commonlog.MockLogger{})

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
				doc := document.NewDocument("test.c3", source+tt.input+`}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(4, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

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
				doc := document.NewDocument("test.c3", source+tt.input+`
				}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestLanguage_BuildCompletionList_modules(t *testing.T) {
	parser := createParser()
	language := NewLanguage(commonlog.MockLogger{})

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
				[]protocol.CompletionItem{
					CreateCompletionItem("app", protocol.CompletionItemKindModule),
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
					protocol.CompletionItem{
						Label:  "app::foo",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
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
				doc := document.NewDocument("test.c3", tt.source)
				language.RefreshDocumentIdentifiers(&doc, &parser)

				completionList := language.BuildCompletionList(&doc, tt.position)

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
					protocol.CompletionItem{
						Label:  "app",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app",
							Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
						},
					},
					protocol.CompletionItem{
						Label:  "app::window",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window",
							Range:   protocol_utils.NewLSPRange(3, 4, 3, 5),
						},
					},
					protocol.CompletionItem{
						Label:  "app::window::errors",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
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
					protocol.CompletionItem{
						Label:  "app::foo",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::foo",
							Range:   protocol_utils.NewLSPRange(5, 4, 5, 9),
						},
					},
					protocol.CompletionItem{
						Label:  "app::window",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
						TextEdit: protocol.TextEdit{
							NewText: "app::window",
							Range:   protocol_utils.NewLSPRange(5, 4, 5, 9),
						},
					},
					protocol.CompletionItem{
						Label:  "app::window::errors",
						Kind:   cast.CompletionItemKindPtr(protocol.CompletionItemKindModule),
						Detail: cast.StrPtr("Module"),
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

				state.registerDoc("app.c3", tt.source)
				doc := state.GetDoc("app.c3")

				completionList := state.language.BuildCompletionList(&doc, tt.position)

				assert.Equal(t, len(tt.expected), len(completionList), "Different items to suggest")
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func CreateCompletionItem(label string, kind protocol.CompletionItemKind) protocol.CompletionItem {
	return protocol.CompletionItem{Label: label, Kind: &kind}
}
