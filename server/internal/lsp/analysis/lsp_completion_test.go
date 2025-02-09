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

func createCompletionItem(label string, kind protocol.CompletionItemKind, detail string, editRange protocol.Range) protocol.CompletionItem {
	return createCompletionItemWithDoc(label, kind, detail, editRange, "")
}

func createCompletionItemWithDoc(label string, kind protocol.CompletionItemKind, detail string, editRange protocol.Range, documentation string) protocol.CompletionItem {
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
				createCompletionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(7, 0, 7, 1)),
			},
			{"va",
				createCompletionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(7, 0, 7, 2)),
			},
			{"x",
				createCompletionItem("xanadu", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(7, 0, 7, 1)),
			},
			{"docu",
				createCompletionItemWithDoc("documented", protocol.CompletionItemKindVariable, "float*", protocol2.NewLSPRange(7, 0, 7, 4), "doc"),
			},
			//{"MY_C",
			//	createCompletionItem("MY_CONST", protocol.CompletionItemKindConstant, "int", protocol2.NewLSPRange(7, 0, 7, 4), "const doc"),
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
				createCompletionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(line, 0, line, 1)),
			},
			{"va",
				createCompletionItem("variable", protocol.CompletionItemKindVariable, "int!", protocol2.NewLSPRange(line, 0, line, 2)),
			},
			{"x",
				createCompletionItem("xanadu", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(line, 0, line, 1)),
			},
			{"docu",
				createCompletionItemWithDoc("documented", protocol.CompletionItemKindVariable, "float*", protocol2.NewLSPRange(line, 0, line, 4), "doc"),
			},
			{"MY_C",
				createCompletionItemWithDoc("MY_CONST", protocol.CompletionItemKindConstant, "int", protocol2.NewLSPRange(line, 0, line, 4), "const doc"),
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

	t.Run("Should macro names", func(t *testing.T) {
		sourceStart := `
		<* abc *>
		macro process(x){}
		macro @process(x){}
		macro empty(){}
		macro @empty(){}
		macro int transform(int x; @body){ return 5; }
		macro replace(float* x; @body(int* a, float b)){}
		fn void main() {`

		line := uint32(9)
		cases := []struct {
			input    string
			expected protocol.CompletionItem
		}{
			{"p",
				createCompletionItemWithDoc("process", protocol.CompletionItemKindFunction, "macro(x)", protocol2.NewLSPRange(line, 0, line, 1), "abc"),
			},
			{"proc",
				createCompletionItemWithDoc("process", protocol.CompletionItemKindFunction, "macro(x)", protocol2.NewLSPRange(line, 0, line, 4), "abc"),
			},
			{"emp",
				createCompletionItem("empty", protocol.CompletionItemKindFunction, "macro()", protocol2.NewLSPRange(line, 0, line, 3)),
			},
			{"trans",
				createCompletionItem("transform", protocol.CompletionItemKindFunction, "macro(int x; @body)", protocol2.NewLSPRange(line, 0, line, 5)),
			},
			{"repla",
				createCompletionItem("replace", protocol.CompletionItemKindFunction, "macro(float* x; @body(int* a, float b))", protocol2.NewLSPRange(line, 0, line, 5)),
			},
		}

		for n, tt := range cases {
			t.Run(fmt.Sprintf("Case #%d", n), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, 1, len(completionList))
				assert.Equal(t, tt.expected.Label, completionList[0].Label)
				assert.Equal(t, protocol.CompletionItemKindFunction, *completionList[0].Kind)
				assert.Equal(t, tt.expected.TextEdit, completionList[0].TextEdit)
				if tt.expected.Documentation == nil {
					assert.Nil(t, completionList[0].Documentation)
				} else {
					assert.Equal(t, tt.expected.Documentation, completionList[0].Documentation)
				}
				assert.NotNil(t, completionList[0].Detail)
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
					createCompletionItem("foo", protocol.CompletionItemKindFunction, "fn void foo()", protocol2.NewLSPRange(line, 0, line, 1)),
					createCompletionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 0, line, 1)),
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
		fn void foo(){} // To confuse algorithm
		fn void main() {
		Obj o;`

		line := uint32(10)
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"x",
				[]protocol.CompletionItem{},
			},
			{"o.f",
				[]protocol.CompletionItem{
					createCompletionItem("foo", protocol.CompletionItemKindFunction, "fn void foo()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("freight", protocol.CompletionItemKindField, "Struct member", protocol2.NewLSPRange(line, 2, line, 3)),
				},
			},
			{"o.fooB",
				[]protocol.CompletionItem{
					createCompletionItem("fooBar", protocol.CompletionItemKindFunction, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 6)),
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

	t.Run("Should suggest deep struct method names while in function scope", func(t *testing.T) {
		sourceStart := `
		struct Deep{int freight;}
		fn void Deep.fooDeep(){}	
		struct Obj{int freight;Deep dep;}
		fn void Obj.foo(){}
		fn void foo(){} // To confuse algorithm
		fn void main() {
		Obj o;`

		line := uint32(8)
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"o.f",
				[]protocol.CompletionItem{
					createCompletionItem("foo", protocol.CompletionItemKindFunction, "fn void foo()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("freight", protocol.CompletionItemKindField, "Struct member", protocol2.NewLSPRange(line, 2, line, 3)),
				},
			},
			{"o.dep.foo",
				[]protocol.CompletionItem{
					createCompletionItem("fooDeep", protocol.CompletionItemKindFunction, "fn void fooDeep()", protocol2.NewLSPRange(line, 6, line, 9)),
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

func TestBuildCompletionList_should_suggest_enums(t *testing.T) {
	t.Run("Should suggest Enum type", func(t *testing.T) {
		sourceStart := `
		enum Cough { COH, COUGH, COUGHCOUGH}
		<* doc *>
		enum Color { RED, GREEN, BLUE }
		fn void main(){`

		line := uint32(5)
		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"A",
				[]protocol.CompletionItem{},
			},
			{"Co",
				[]protocol.CompletionItem{
					createCompletionItemWithDoc("Color", protocol.CompletionItemKindEnum, "Enum", protocol2.NewLSPRange(line, 0, line, 2), "doc"),
					createCompletionItem("Cough", protocol.CompletionItemKindEnum, "Enum", protocol2.NewLSPRange(line, 0, line, 2)),
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

	t.Run("Should suggest Enumerable type", func(t *testing.T) {
		sourceStart := `
		enum Cough { COH, COUGH, COUGHCOUGH}
		enum Color { RED, GREEN, BLUE, COBALT }
		fn void main() {`
		line := uint32(4)
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"Find enumerables starting with string",
				"CO",
				[]protocol.CompletionItem{
					createCompletionItem("COBALT", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 0, line, 2)),
					createCompletionItem("COH", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 0, line, 2)),
					createCompletionItem("COUGH", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 0, line, 2)),
					createCompletionItem("COUGHCOUGH", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 0, line, 2)),
				}},

			{
				"Find all enum enumerables when prefixed with enum name",
				"Color.",
				[]protocol.CompletionItem{
					createCompletionItem("BLUE", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("COBALT", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("GREEN", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("RED", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
				}},
			{
				"Find matching enum enumerables",
				"Color.COB",
				[]protocol.CompletionItem{
					createCompletionItem("COBALT", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 9)),
				},
			},
		}

		for _, tt := range cases {
			t.Run(fmt.Sprintf("Case #%s", tt.input), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, len(tt.expected), len(completionList))
				for idx, item := range completionList {
					assert.Equal(t, tt.expected[idx].Label, item.Label)
					assert.Equal(t, tt.expected[idx].Kind, item.Kind)
					assert.Equal(t, tt.expected[idx].TextEdit, item.TextEdit, "test edit is wrong")
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
