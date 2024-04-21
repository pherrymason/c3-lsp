package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func createParser() Parser {
	logger := &commonlog.MockLogger{}
	return NewParser(logger)
}

func TestFindsTypedEnums(t *testing.T) {
	module := "x"
	docId := "doc"
	source := `enum Colors:int { RED, BLUE, GREEN };`
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("finds Colors enum identifier", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		expectedEnum := idx.NewEnumBuilder("Colors", "int", module, docId).
			Build()

		assert.NotNil(t, symbols.Enums["Colors"])
		assert.Equal(t, expectedEnum.GetName(), symbols.Enums["Colors"].GetName())
		assert.Equal(t, expectedEnum.GetType(), symbols.Enums["Colors"].GetType())
	})

	t.Run("reads ranges for enum", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		enum := symbols.Enums["Colors"]
		assert.Equal(t, idx.NewRange(0, 0, 0, 36), enum.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(0, 5, 0, 11), enum.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds defined enumerators", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		enum := symbols.Enums["Colors"]
		e := enum.GetEnumerator("RED")
		assert.Equal(t, "RED", e.GetName())
		assert.Equal(t, idx.NewRange(0, 18, 0, 21), e.GetIdRange())

		e = enum.GetEnumerator("BLUE")
		assert.Equal(t, "BLUE", e.GetName())
		assert.Equal(t, idx.NewRange(0, 23, 0, 27), e.GetIdRange())

		e = enum.GetEnumerator("GREEN")
		assert.Equal(t, "GREEN", e.GetName())
		assert.Equal(t, idx.NewRange(0, 29, 0, 34), e.GetIdRange())
	})
}

func TestFindsUnTypedEnums(t *testing.T) {
	module := "x"
	docId := "doc"
	source := `enum Colors { RED, BLUE, GREEN };`
	doc := document.NewDocument(docId, module, source)
	parser := createParser()

	t.Run("finds Colors enum identifier", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		expectedEnum := idx.NewEnumBuilder("Colors", "", module, docId).
			Build()

		assert.NotNil(t, symbols.Enums["Colors"])
		assert.Equal(t, expectedEnum.GetName(), symbols.Enums["Colors"].GetName())
		assert.Equal(t, expectedEnum.GetType(), symbols.Enums["Colors"].GetType())
	})

	t.Run("reads ranges for enum", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		enum := symbols.Enums["Colors"]
		assert.Equal(t, idx.NewRange(0, 0, 0, 32), enum.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(0, 5, 0, 11), enum.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds defined enumerators", func(t *testing.T) {
		symbols := parser.ExtractSymbols(&doc)

		enum := symbols.Enums["Colors"]
		e := enum.GetEnumerator("RED")
		assert.Equal(t, "RED", e.GetName())
		assert.Equal(t, idx.NewRange(0, 14, 0, 17), e.GetIdRange())

		e = enum.GetEnumerator("BLUE")
		assert.Equal(t, "BLUE", e.GetName())
		assert.Equal(t, idx.NewRange(0, 19, 0, 23), e.GetIdRange())

		e = enum.GetEnumerator("GREEN")
		assert.Equal(t, "GREEN", e.GetName())
		assert.Equal(t, idx.NewRange(0, 25, 0, 30), e.GetIdRange())
	})
}

func TestExtractSymbols_finds_definition(t *testing.T) {
	source := `
	def Kilo = int;
	def KiloPtr = Kilo*;
	def MyFunction = fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*);
	def MyMap = HashMap(<String, Feature>);
	`
	// TODO: Missing def different definition examples. See parser.nodeToDef
	doc := document.NewDocument("x", "x", source)
	parser := createParser()

	symbols := parser.ExtractSymbols(&doc)

	expectedDefKilo := idx.NewDefBuilder("Kilo", "x").
		WithResolvesTo("int").
		WithIdentifierRange(1, 5, 1, 9).
		WithDocumentRange(1, 1, 1, 16).
		Build()

	expectedDefKiloPtr := idx.NewDefBuilder("KiloPtr", "x").
		WithResolvesTo("Kilo*").
		WithIdentifierRange(2, 5, 2, 12).
		WithDocumentRange(2, 1, 2, 21).
		Build()

	expectedDefFunction := idx.NewDefBuilder("MyFunction", "x").
		WithResolvesTo("fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*)").
		WithIdentifierRange(3, 5, 3, 15).
		WithDocumentRange(3, 1, 3, 74).
		Build()

	expectedDefTypeWithGenerics := idx.NewDefBuilder("MyMap", "x").
		WithResolvesTo("HashMap(<String, Feature>)").
		WithIdentifierRange(4, 5, 4, 10).
		WithDocumentRange(4, 1, 4, 40).
		Build()

	assert.Equal(t, expectedDefKilo, symbols.Defs["Kilo"])
	assert.Equal(t, expectedDefKiloPtr, symbols.Defs["KiloPtr"])
	assert.Equal(t, expectedDefFunction, symbols.Defs["MyFunction"])

	assert.Equal(t, expectedDefTypeWithGenerics, symbols.Defs["MyMap"])
}

func TestExtractSymbols_find_macro(t *testing.T) {
	if true {
		t.Skip("Incomplete until defining macros in grammar.js")
	}

	sourceCode := `
	macro void log(LogLevel $level, String format, args...) {
		if (log_level != OFF && $level <= log_level) {
			io::fprintf(&log_file, "[%s] ", $level)!!;
			io::fprintfn(&log_file, format, ...args)!!;
		}
	}`

	_ = document.NewDocument("x", "x", sourceCode)
	//	parser := createParser()
	//	tree := parser.ExtractSymbols(&doc)

	assert.Equal(t, true, true)
}

func dfs(n *sitter.Node, level int) {
	if n == nil {
		return
	}

	// Procesa el nodo actual (puedes imprimir, almacenar en un slice, etc.)
	tabs := strings.Repeat("\t", level)
	fmt.Printf("%sNode", tabs)
	//fmt.Printf("%sPos: %d - %d -> ", tabs, n.StartPoint().Row, n.StartPoint().Column)
	//fmt.Printf("%d - %d\n", n.EndPoint().Row, n.EndPoint().Column)
	fmt.Printf("%sType: %s", tabs, n.Type())
	//fmt.Printf("\tContent: %s", n.C)
	fmt.Printf("\n")

	// Llama recursivamente a DFS para los nodos hijos
	//fmt.Printf("~inside~")
	for i := uint32(0); i < n.ChildCount(); i++ {
		child := n.Child(int(i))
		dfs(child, level+1)
	}
}
