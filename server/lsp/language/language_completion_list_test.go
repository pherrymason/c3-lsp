package language

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func Test_isCompletingAChain(t *testing.T) {
	cases := []struct {
		name                     string
		input                    string
		position                 indexables.Position
		expected                 bool
		expectedPreviousPosition indexables.Position
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
	//parser := createParser()
	//language := NewLanguage(logger)

	t.Run("Should suggest struct members", func(t *testing.T) {
		expectedKind := protocol.CompletionItemKindField
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"suggest all struct members",
				".",
				[]protocol.CompletionItem{
					{Label: "color", Kind: &expectedKind},
					{Label: "height", Kind: &expectedKind},
					{Label: "width", Kind: &expectedKind},
				}},

			{
				"suggest members starting with `w`",
				".w",
				[]protocol.CompletionItem{
					{Label: "width", Kind: &expectedKind},
				}},

			{
				"suggest members of Color",
				".color.", []protocol.CompletionItem{
					{Label: "blue", Kind: &expectedKind},
					{Label: "green", Kind: &expectedKind},
					{Label: "red", Kind: &expectedKind},
				}},

			{
				"suggest members of Color starting with `r`",
				".color.r",
				[]protocol.CompletionItem{
					{Label: "red", Kind: &expectedKind},
				}},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				state.registerDoc("test.c3",
					`
				struct Color { int red; int green; int blue; }
				struct Square { int width; int height; Color color; }
				fn void main() {
					Square inst;
					inst`+tt.input+`}`,
				)
				doc := state.GetDoc("test.c3")
				position := buildPosition(6, 9+uint(len(tt.input))) // Cursor after `<input>|`

				completionList := state.language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}

func TestLanguage_BuildCompletionList_enums(t *testing.T) {
	t.Skip()
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
					CreateCompletionItem("RED", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("GREEN", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("BLUE", protocol.CompletionItemKindEnumMember),
					CreateCompletionItem("COBALT", protocol.CompletionItemKindEnumMember),
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

func CreateCompletionItem(label string, kind protocol.CompletionItemKind) protocol.CompletionItem {
	return protocol.CompletionItem{Label: label, Kind: &kind}
}
