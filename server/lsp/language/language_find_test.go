package language

import (
	"fmt"
	"os"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}

func readC3File(filePath string) string {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error al leer el archivo: %v\n", err)
		return ""
	}

	// Convierte el slice de bytes a un string
	return string(contentBytes)
}

func docCpu() *document.Document {
	fileCpu := readC3File("./test_files/cpu.c3")
	doc := document.NewDocument("cpu.c3", "?", fileCpu)

	return &doc
}

func docCpuRegisters() *document.Document {
	fileCpuRegisters := readC3File("./test_files/cpu.registers.c3")
	doc := document.NewDocument("cpu.registers.c3", "?", fileCpuRegisters)
	return &doc
}

func docAudio() *document.Document {
	fileCpuBus := readC3File("./test_files/emu.c3")
	doc := document.NewDocument("audio.c3", "?", fileCpuBus)
	return &doc
}

func docEmu() *document.Document {
	fileCpuBus := readC3File("./test_files/emu.c3")
	doc := document.NewDocument("emu.c3", "?", fileCpuBus)
	return &doc
}

func initTestEnv() (*Language, map[string]*document.Document) {

	documents := make(map[string]*document.Document, 0)

	//documents["cpu"] = docCpu()
	//documents["cpu.registers"] = docCpuRegisters()
	//documents["audio"] = docAudio()
	documents["emu.c3"] = docEmu()
	fileCpuBus := readC3File("./test_files/definitions.c3")
	d := document.NewDocument("definitions.c3", "?", fileCpuBus)
	documents["definitions.c3"] = &d

	parser := createParser()
	language := NewLanguage()
	//language.RefreshDocumentIdentifiers(documents["cpu"], &parser)
	//language.RefreshDocumentIdentifiers(documents["cpu.registers"], &parser)
	language.RefreshDocumentIdentifiers(documents["definitions.c3"], &parser)
	language.RefreshDocumentIdentifiers(documents["emu.c3"], &parser)

	return &language, documents
}

func TestLanguage_findClosestSymbolDeclaration(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		searchParams := NewSearchParams("emulator", "emu.c3")
		position := protocol.Position{8, 9}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		searchParams := NewSearchParams("emulator", "emu.c3")
		position := protocol.Position{11, 10}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType())
	})

	t.Run("should find definition declaration in same scope of cursor", func(t *testing.T) {
		searchParams := NewSearchParams("Emu", "emu.c3")
		position := protocol.Position{8, 2}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("should find struct declaration in same scope of cursor", func(t *testing.T) {
		searchParams := NewSearchParams("Emu", "emu.c3")
		position := protocol.Position{8, 2}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Find local struct member variable definition", func(t *testing.T) {
		position := protocol.Position{9, 13}
		searchParams, _ := buildSearchParams(documents["emu.c3"], position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Find local enum variable definition", func(t *testing.T) {
		searchParams := NewSearchParams("status", "emu.c3")
		position := protocol.Position{15, 5}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "status", resolvedSymbol.GetName())
		assert.Equal(t, "WindowStatus", variable.GetType())
	})
	t.Run("Find local enumerator definition", func(t *testing.T) {
		searchParams := NewSearchParams("BACKGROUND", "emu.c3")
		position := protocol.Position{16, 12}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		assert.Equal(t, "BACKGROUND", resolvedSymbol.GetName())
	})

	t.Run("Find local definition definition", func(t *testing.T) {
		searchParams := NewSearchParams("Kilo", "definitions.c3")
		position := protocol.Position{1, 2}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Def not found")
		assert.Equal(t, "Kilo", resolvedSymbol.GetName())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		searchParams := NewSearchParams("tick", "emu.c3")
		position := protocol.Position{14, 4}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "tick", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType())
	})

	t.Run("Find local function definition", func(t *testing.T) {
		searchParams := NewSearchParams("run", "emu.c3")
		position := protocol.Position{10, 2}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Local function not found")

		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "run", fun.GetName())
		assert.Equal(t, "void", fun.GetReturnType())
	})

	t.Run("Asking the selectedSymbol information in the very same declaration, should resolve to the correct selectedSymbol. Even if there is another selectedSymbol with same name in a different file.", func(t *testing.T) {
		t.Skip()
		// Should only resolve in very same module, unless module B is imported.
		// ---------------------
		// module A has int out;
		// module B has int out;
		// asking info about B::out should resolve to B::out, and not A::out.

		// Other cases:
		// module A;
		// struct MyStruct{}
		// fn void MyStruct.search(&self) {}
		// fn void search() {}
		//
		// module B;
		// MyStruct object;
		// object.search();
	})

	t.Run("resolve fn correctly", func(t *testing.T) {
		t.Skip()
		// struct MyStruct{}
		// fn void MyStruct.search(&self) {}
		// fn void search() {}
		//
		// MyStruct object;
		// object.search();
	})

	t.Run("resolve variable from other module", func(t *testing.T) {
		t.Skip()
		// struct MyStruct{}
		// fn void MyStruct.search(&self) {}
		// fn void search() {}
		//
		// MyStruct object;
		// object.search();
	})
}

func TestLanguage_FindHoverInformation(t *testing.T) {
	language := NewLanguage()
	parser := createParser()

	doc := document.NewDocument("x", "", `
	int value = 1;
	fn void main() {
		char value = 3;
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{
				Line:      3,
				Character: 8,
			},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover, _ := language.FindHoverInformation(&doc, &params)

	expectedHover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("char value"),
		},
	}
	assert.Equal(t, expectedHover, hover)
}

func TestLanguage_FindHoverInformationFromDifferentFile(t *testing.T) {
	language := NewLanguage()
	parser := createParser()

	doc := document.NewDocument("x", "x", `
	fn void main() {
		importedMethod();
	}
`)
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc2 := document.NewDocument("y", "x", `
	fn void importedMethod() {}
	`)
	language.RefreshDocumentIdentifiers(&doc2, &parser)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: "x"},
			protocol.Position{Line: 2, Character: 8},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}

	hover, _ := language.FindHoverInformation(&doc, &params)

	expectedHover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("void importedMethod()"),
		},
	}
	assert.Equal(t, expectedHover, hover)
}

func newDeclarationParams(docId string, line protocol.UInteger, char protocol.UInteger) protocol.DeclarationParams {
	return protocol.DeclarationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			protocol.TextDocumentIdentifier{URI: docId},
			protocol.Position{line, char},
		},
		WorkDoneProgressParams: protocol.WorkDoneProgressParams{},
	}
}

func initLanguage(docId string, module string, source string) (Language, *document.Document, *p.Parser) {
	doc := document.NewDocument(docId, module, source)
	language := NewLanguage()
	parser := createParser()
	language.RefreshDocumentIdentifiers(&doc, &parser)

	return language, &doc, &parser
}

func TestLanguage_ShouldFindVariablesInSameScope(t *testing.T) {
	module := "mod"
	docId := "docId"

	t.Run("should find definition declaration in same scope of cursor", func(t *testing.T) {
		source := `def Kilo = int;Kilo value = 3;`
		language, doc, _ := initLanguage(docId, module, source)

		params := newDeclarationParams(docId, 0, 17)

		symbol, _ := language.FindSymbolDeclarationInWorkspace(doc, params.Position)

		expected := idx.NewDefBuilder("Kilo", docId).
			WithResolvesTo("int").
			WithIdentifierRange(0, 4, 0, 8).
			WithDocumentRange(0, 0, 0, 15).
			Build()
		assert.Equal(t, expected, symbol)
	})

}
