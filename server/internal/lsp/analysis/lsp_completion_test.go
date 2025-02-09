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
					createCompletionItem("foo", protocol.CompletionItemKindMethod, "fn void foo()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("fooBar", protocol.CompletionItemKindMethod, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("freight", protocol.CompletionItemKindField, "Struct member", protocol2.NewLSPRange(line, 2, line, 3)),
				},
			},
			{"o.fooB",
				[]protocol.CompletionItem{
					createCompletionItem("fooBar", protocol.CompletionItemKindMethod, "fn void fooBar()", protocol2.NewLSPRange(line, 2, line, 6)),
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
					createCompletionItem("foo", protocol.CompletionItemKindMethod, "fn void foo()", protocol2.NewLSPRange(line, 2, line, 3)),
					createCompletionItem("freight", protocol.CompletionItemKindField, "Struct member", protocol2.NewLSPRange(line, 2, line, 3)),
				},
			},
			{"o.dep.foo",
				[]protocol.CompletionItem{
					createCompletionItem("fooDeep", protocol.CompletionItemKindMethod, "fn void fooDeep()", protocol2.NewLSPRange(line, 6, line, 9)),
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

	t.Run("Should suggest enum associated values", func(t *testing.T) {
		sourceStart := `
		enum Color : (int assoc, float abc) {
			RED = { 1, 2.0 },
			BLUE = { 2, 4.0 }
		}
		fn void main() {`
		line := uint32(6)
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"Do not find associated values on enum type",
				"Color.a",
				[]protocol.CompletionItem{}},
			{
				"Find associated values on explicit constant",
				"Color.RED.a",
				[]protocol.CompletionItem{
					createCompletionItem("abc", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(line, 6, line, 9)),
					createCompletionItem("assoc", protocol.CompletionItemKindVariable, "int", protocol2.NewLSPRange(line, 6, line, 9)),
				}},

			{
				"Find matching associated values on explicit constant",
				"Color.RED.asso",
				[]protocol.CompletionItem{
					createCompletionItem("assoc", protocol.CompletionItemKindVariable, "int", protocol2.NewLSPRange(line, 6, line, 9)),
				}},

			{
				"Find associated values on enum instance variable",
				`Color clr = Color.RED;
clr.a`,
				[]protocol.CompletionItem{
					createCompletionItem("abc", protocol.CompletionItemKindVariable, "float", protocol2.NewLSPRange(line, 6, line, 9)),
					createCompletionItem("assoc", protocol.CompletionItemKindVariable, "int", protocol2.NewLSPRange(line, 6, line, 9)),
				}},

			{
				"Find matching associated values on enum instance variable",
				`Color clr = Color.RED;
clr.asso`,
				[]protocol.CompletionItem{
					createCompletionItem("assoc", protocol.CompletionItemKindVariable, "int", protocol2.NewLSPRange(line, 6, line, 9)),
				}},
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

	t.Run("Should suggest Enum methods", func(t *testing.T) {
		sourceStart := `
		enum Color { RED, GREEN, BLUE, COBALT }
		fn Color Color.transparentize(self) {}
		fn void main() {
`
		line := uint32(4)
		cases := []struct {
			name     string
			input    string
			expected []protocol.CompletionItem
		}{
			{
				"Find enum methods by type name prefix",
				"Color.",
				[]protocol.CompletionItem{
					createCompletionItem("BLUE", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("COBALT", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("GREEN", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					createCompletionItem("RED", protocol.CompletionItemKindEnumMember, "Enum Value", protocol2.NewLSPRange(line, 6, line, 6)),
					{
						Label: "transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 6, line, 6),
						},
						Detail: cast.ToPtr("fn Color transparentize(Color self)"),
					},
				}},
			{
				"Find matching enum method by type name prefix",
				"Color.transpa",
				[]protocol.CompletionItem{
					{
						Label: "transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 6, line, 13),
						},
						Detail: cast.ToPtr("fn Color transparentize(Color self)"),
					},
				},
			},
			{
				"Find enum methods with explicit enum value prefix",
				"Color.GREEN.",
				[]protocol.CompletionItem{
					{
						Label: "transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 12, line, 12),
						},
						Detail: cast.ToPtr("fn Color transparentize(Color self)"),
					},
				},
			},
			{
				"Find matching enum methods with explicit enum value prefix",
				"Color.GREEN.transp",
				[]protocol.CompletionItem{
					{
						Label: "transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 12, line, 18),
						},
						Detail: cast.ToPtr("fn Color transparentize(Color self)"),
					},
				},
			},
			{
				"Find enum methods by instance variable prefix",
				`Color green = Color.GREEN;
green.`,
				[]protocol.CompletionItem{
					{
						Label: "Color.transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 33, line, 33),
						},
						Detail: cast.ToPtr("fn Color(Color self)"),
					},
				},
			},
			{
				"Find matching enum method by instance variable prefix",
				`Color green = Color.GREEN;
green.transp`,
				[]protocol.CompletionItem{
					{
						Label: "Color.transparentize",
						Kind:  cast.ToPtr(protocol.CompletionItemKindMethod),
						TextEdit: protocol.TextEdit{
							NewText: "transparentize",
							Range:   protocol2.NewLSPRange(line, 6, line, 12),
						},
						Detail: cast.ToPtr("fn Color(Color self)"),
					},
				},
			},
		}

		for _, tt := range cases {
			t.Run(fmt.Sprintf("Autocomplete enum methods: #%s", tt.name), func(t *testing.T) {
				completionList := getCompletionList(sourceStart + tt.input + "|||\n}")

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

func TestBuildCompletionList_definitions(t *testing.T) {

	t.Run("Should suggest definitions", func(t *testing.T) {
		sourceStart := `
		<* abc *>
		def Kilo = int;
		def KiloPtr = Kilo*;
		def MyFunction = fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*);
		def MyMap = HashMap(<String, Feature>);
		def Camera = raylib::Camera;

		def func = a(<String>);
		def aliased_global = global_var;
		def CONST_ALIAS = MY_CONST;
		def @macro_alias = @a;
		fn void main() {`

		//		line := uint32(4)
		expectedKind := protocol.CompletionItemKindTypeParameter

		cases := []struct {
			input    string
			expected []protocol.CompletionItem
		}{
			{"Kil", []protocol.CompletionItem{
				createCompletionItemWithDoc("Kilo", protocol.CompletionItemKindTypeParameter, "Type alias for 'int'", protocol2.NewLSPRange(1, 0, 1, 3), "abc"),
				//{Label: "Kilo", Kind: &expectedKind, Detail: cast.ToPtr("Type"), Documentation: asMarkdown("abc")},
				{Label: "KiloPtr", Kind: &expectedKind, Detail: cast.ToPtr("Type alias for 'Kilo*'"), Documentation: nil},
			}},
			{"KiloP", []protocol.CompletionItem{
				{Label: "KiloPtr", Kind: &expectedKind, Detail: cast.ToPtr("Type alias for 'Kilo*'"), Documentation: nil},
			}},
			{"MyFunct", []protocol.CompletionItem{
				{Label: "MyFunction", Kind: &expectedKind, Detail: cast.ToPtr("Type alias for 'fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*)'"), Documentation: nil},
			}},
			{"MyMa", []protocol.CompletionItem{
				{Label: "MyMap", Kind: &expectedKind, Detail: cast.ToPtr("Type alias for 'HashMap(<String, Feature>)'"), Documentation: nil},
			}},
			{"Came", []protocol.CompletionItem{
				{Label: "Camera", Kind: &expectedKind, Detail: cast.ToPtr("Type alias for 'raylib::Camera'"), Documentation: nil},
			}},
			{"fun", []protocol.CompletionItem{
				{Label: "func", Kind: &expectedKind, Detail: cast.ToPtr("Alias for 'a(<String>)'"), Documentation: nil},
			}},
			{"aliased_g", []protocol.CompletionItem{
				{Label: "aliased_global", Kind: &expectedKind, Detail: cast.ToPtr("Alias for 'global_var'"), Documentation: nil},
			}},
			{"CONST_AL", []protocol.CompletionItem{
				{Label: "CONST_ALIAS", Kind: &expectedKind, Detail: cast.ToPtr("Alias for 'MY_CONST'"), Documentation: nil},
			}},
			{"@macro_alias", []protocol.CompletionItem{
				{Label: "@macro_alias", Kind: &expectedKind, Detail: cast.ToPtr("Alias for '@a'"), Documentation: nil},
			}},
		}

		for _, tt := range cases {
			t.Run(fmt.Sprintf("Case #%s", tt.input), func(t *testing.T) {

				completionList := getCompletionList(sourceStart + "\n" + tt.input + "|||\n}")

				assert.Equal(t, len(tt.expected), len(completionList))
				for idx, item := range completionList {
					assert.Equal(t, tt.expected[idx].Label, item.Label)
					assert.Equal(t, tt.expected[idx].Kind, item.Kind)
					//	assert.Equal(t, tt.expected[idx].TextEdit, item.TextEdit, "test edit is wrong")
					if tt.expected[idx].Documentation == nil {
						assert.Nil(t, item.Documentation)
					} else {
						assert.Equal(t, tt.expected[idx].Documentation, item.Documentation)
					}

					if item.Detail != nil {
						assert.Equal(t, *tt.expected[idx].Detail, *item.Detail, "Wrong Detail")
					} else {
						assert.Fail(t, "Detail is nil")
					}
				}
			})
		}
	})
}

func TestBuildCompletionList_should_suggest_nothing_when_on_literal(t *testing.T) {

}

func TestBuildCompletionList_suggests_C3_keywords(t *testing.T) {

}
