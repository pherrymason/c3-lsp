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
	parser := createParser()
	language := NewLanguage(commonlog.MockLogger{})

	//documents := installDocuments(&language, &parser)

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

				doc := document.NewDocument("test.c3", source+"\n"+tt.input)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(4, 1) // Cursor after `v|`

				completionList := language.BuildCompletionList(&doc, position)

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
				{Label: "variable", Kind: &expectedKind},
				{Label: "value", Kind: &expectedKind},
			}},
			{"val", []protocol.CompletionItem{
				{Label: "value", Kind: &expectedKind},
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				doc := document.NewDocument("test.c3", source+`
`+tt.input+`
				}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(5, uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})

	t.Run("Should suggest struct members", func(t *testing.T) {
		source := `
		struct Square { int width; int height; }
		fn void main() {
			Square inst;
			inst`
		expectedKind := protocol.CompletionItemKindField
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{".", []protocol.CompletionItem{
				{Label: "width", Kind: &expectedKind},
				{Label: "height", Kind: &expectedKind},
			}},

			{".w", []protocol.CompletionItem{
				{Label: "width", Kind: &expectedKind},
			}},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				doc := document.NewDocument("test.c3", source+tt.input+`}`)
				language.RefreshDocumentIdentifiers(&doc, &parser)
				position := buildPosition(5, 7+uint(len(tt.input))) // Cursor after `<input>|`

				completionList := language.BuildCompletionList(&doc, position)

				assert.Equal(t, len(tt.expected), len(completionList))
				assert.Equal(t, tt.expected, completionList)
			})
		}
	})
}
