package analysis

import (
	"fmt"
	protocol2 "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"testing"
)

func asMarkdown(text string) protocol.MarkupContent {
	return protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: text,
	}
}

func getCompletionList(source string) []protocol.CompletionItem {
	uri := "file://dummy.c3"
	srv, position := startTestServer(source, uri)

	getDocument, _ := srv.documents.GetDocument(uri)

	return BuildCompletionList(getDocument, position, srv.documents, srv.symbolTable)
}

func completionItem(label string, kind protocol.CompletionItemKind, detail string, editRange protocol.Range, documentation string) protocol.CompletionItem {
	item := protocol.CompletionItem{
		Label: label,
		Kind:  cast.ToPtr(kind),
		TextEdit: protocol.TextEdit{
			NewText: label,
			Range:   editRange,
		},
		Detail: cast.ToPtr(detail),
	}
	if documentation != "" {
		item.Documentation = cast.ToPtr(asMarkdown(documentation))
	}

	return item
}

func TestBuildCompletionList_suggests_variables(t *testing.T) {
	t.Run("Should suggest variable names defined in module while in global scope", func(t *testing.T) {
		sourceStart := `
		int! variable = 3;
		float xanadu = 10.0;
		<* doc *>
		float* documented = &xanadu;
		<* const doc *>
		const int MY_CONST = 100;`

		cases := []struct {
			input    string
			expected protocol.CompletionItem
		}{
			{"v",
				completionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(7, 0, 7, 1), ""),
			},
			{"va",
				completionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(7, 0, 7, 2), ""),
			},
			{"x",
				completionItem("xanadu", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(7, 0, 7, 1), ""),
			},
			{"docu",
				completionItem("documented", protocol.CompletionItemKindVariable, "float*", protocol2.NewLSPRange(7, 0, 7, 4), "doc"),
			},
			//{"MY_C",
			//	completionItem("MY_CONST", protocol.CompletionItemKindConstant, "int", protocol2.NewLSPRange(7, 0, 7, 4), "const doc"),
			//},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||")

				assert.Equal(t, 1, len(completionList))
				assert.Equal(t, tt.expected.Label, completionList[0].Label)
				assert.Equal(t, tt.expected.Kind, completionList[0].Kind)
				assert.Equal(t, tt.expected.TextEdit, completionList[0].TextEdit)
				if tt.expected.Documentation == nil {
					assert.Nil(t, completionList[0].Documentation)
				} else {
					assert.Equal(t, tt.expected.Documentation, completionList[0].Documentation)
				}
				assert.Equal(t, *tt.expected.Detail, *completionList[0].Detail)
			})
		}
	})

	t.Run("Should suggest variable names while in function scope", func(t *testing.T) {
		// This test covers cases where treesitter cannot fully parse a mid-written expression/statement
		sourceStart := `
		int! variable = 3;
		float xanadu = 10.0;
		<* doc *>
		float* documented = &xanadu;
		<* const doc *>
		const int MY_CONST = 100;
		fn void main() {`

		line := uint32(8)
		cases := []struct {
			input    string
			expected protocol.CompletionItem
		}{
			{"v",
				completionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(line, 0, line, 1), ""),
			},
			{"va",
				completionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(line, 0, line, 2), ""),
			},
			{"x",
				completionItem("xanadu", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(line, 0, line, 1), ""),
			},
			{"docu",
				completionItem("documented", protocol.CompletionItemKindVariable, "float*", protocol2.NewLSPRange(line, 0, line, 4), "doc"),
			},
			{"MY_C",
				completionItem("MY_CONST", protocol.CompletionItemKindConstant, "int", protocol2.NewLSPRange(line, 0, line, 4), "const doc"),
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, 1, len(completionList))
				assert.Equal(t, tt.expected.Label, completionList[0].Label)
				assert.Equal(t, tt.expected.Kind, completionList[0].Kind)
				assert.Equal(t, tt.expected.TextEdit, completionList[0].TextEdit)
				if tt.expected.Documentation == nil {
					assert.Nil(t, completionList[0].Documentation)
				} else {
					assert.Equal(t, tt.expected.Documentation, completionList[0].Documentation)
				}
				assert.Equal(t, *tt.expected.Detail, *completionList[0].Detail)
			})
		}
	})
}

func TestBuildCompletionList_should_suggest_functions(t *testing.T) {
	t.Run("Should suggest functions names while in function scope", func(t *testing.T) {
		sourceStart := `
		fn void foo(){}
		fn void fooBar(){}
		fn void main() {`

		line := uint32(4)
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"x",
				[]protocol.CompletionItem{},
			},
			{"f",
				[]protocol.CompletionItem{
					completionItem("foo", protocol.CompletionItemKindFunction, "fn void foo()", protocol2.NewLSPRange(line, 0, line, 1), ""),
					completionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 0, line, 1), ""),
				},
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, len(tt.expected), len(completionList))
				for idx, item := range completionList {
					assert.Equal(t, tt.expected[idx].Label, item.Label)
					assert.Equal(t, tt.expected[idx].Kind, item.Kind)
					assert.Equal(t, tt.expected[idx].TextEdit, item.TextEdit)
					if tt.expected[idx].Documentation == nil {
						assert.Nil(t, item.Documentation)
					} else {
						assert.Equal(t, tt.expected[idx].Documentation, item.Documentation)
					}
					assert.Equal(t, *tt.expected[idx].Detail, *item.Detail)
				}
			})
		}
	})

	t.Run("Should suggest struct method names while in function scope", func(t *testing.T) {
		sourceStart := `
		struct Obj{
			int freight;
		}
		fn void Obj.foo(){}
		fn void Obj.fooBar(){}
		fn void Obj.abc(){}
		fn void main() {
		Obj o;`

		line := uint32(9)
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"x",
				[]protocol.CompletionItem{},
			},
			{"o.f",
				[]protocol.CompletionItem{
					completionItem("freight", protocol.CompletionItemKindField, "Struct member", protocol2.NewLSPRange(line, 2, line, 3), ""),
					completionItem("foo", protocol.CompletionItemKindFunction, "fn void foo()", protocol2.NewLSPRange(line, 2, line, 3), ""),
					completionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 3), ""),
				},
			},
			{"o.fooB",
				[]protocol.CompletionItem{
					completionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 6), ""),
				},
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, len(tt.expected), len(completionList))
				for idx, item := range completionList {
					assert.Equal(t, tt.expected[idx].Label, item.Label)
					assert.Equal(t, *tt.expected[idx].Kind, *item.Kind)
					assert.Equal(t, tt.expected[idx].TextEdit, item.TextEdit)
					if tt.expected[idx].Documentation == nil {
						assert.Nil(t, item.Documentation)
					} else {
						assert.Equal(t, tt.expected[idx].Documentation, item.Documentation)
					}
					assert.Equal(t, *tt.expected[idx].Detail, *item.Detail)
				}
			})
		}
	})
}

func TestBuildCompletionList_should_suggest_nothing_when_on_literal(t *testing.T) {

}

func TestBuildCompletionList_suggests_C3_keywords(t *testing.T) {

}
