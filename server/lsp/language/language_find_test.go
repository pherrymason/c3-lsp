package language

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/document"
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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

func installDocuments(language *Language, parser *p.Parser) map[string]document.Document {
	var fileContent string

	filenames := []string{"app.c3", "app_helper.c3", "emu.c3", "definitions.c3", "cpu.c3",

		// Module related sources
		"module_foo.c3",
		"module_foo_bar.c3",
		"module_foo_circle.c3",
		"module_foo2.c3",
	}
	baseDir := "./test_files/"
	documents := make(map[string]document.Document, 0)

	for _, filename := range filenames {
		// Construir la ruta completa al archivo
		fullPath := filepath.Join(baseDir, filename)
		fileContent = readC3File(fullPath)
		documents[filename] = document.NewDocument(filename, "?", fileContent)
		doc := documents[filename]
		language.RefreshDocumentIdentifiers(&doc, parser)
	}

	return documents
}

func initTestEnv() (*Language, map[string]document.Document) {
	parser := createParser()
	language := NewLanguage()

	documents := installDocuments(&language, &parser)

	return &language, documents
}

func buildPosition(line protocol.UInteger, character protocol.UInteger) protocol.Position {
	return protocol.Position{Line: line - 1, Character: character}
}

func TestLanguage_findClosestSymbolDeclaration_in_same_scope(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("resolve fn correctly", func(t *testing.T) {
		t.Skip()
		// struct MyStruct{}
		// fn void MyStruct.search(&self) {}
		// fn void search() {}
		//
		// MyStruct object;
		// object.search();
	})

	t.Run("resolve implicit variable from module", func(t *testing.T) {
		position := buildPosition(6, 21) // Cursor at BAR_WEIGHT
		doc := documents["module_foo2.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModule())
	})

	t.Run("resolve explicit variable from sub module", func(t *testing.T) {
		t.Skip()
		position := buildPosition(7, 28) // Cursor at foo::bar::DEFAULT_BAR_COLOR;
		doc := documents["module_foo2.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "DEFAULT_BAR_COLOR", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModule())
	})

	t.Run("resolve variable from sub module", func(t *testing.T) {
		t.Skip()
		position := buildPosition(7, 21) // Cursor at BAR_WEIGHT
		doc := documents["module_foo.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModule())
	})
}

func TestLanguage_findClosestSymbolDeclaration_variables(t *testing.T) {
	language, _ := initTestEnv()

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		position := buildPosition(23, 9)
		searchParams := NewSearchParams("emulator", position, "emu.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Symbol not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		position := buildPosition(24, 10)
		searchParams := NewSearchParams("emulator", position, "emu.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Should find the right element when there is a different element with the same name up in the scope", func(t *testing.T) {
		position := buildPosition(16, 9)
		searchParams := NewSearchParams("ambiguousVariable", position, "app.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "ambiguousVariable", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find variable definition in same module, but different file", func(t *testing.T) {
		position := buildPosition(2, 2)
		searchParams := NewSearchParams("helpDisplayedTimes", position, "app.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "helpDisplayedTimes", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})
}

func TestLanguage_findClosestSymbolDeclaration_structs(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Should find struct declaration in variable declaration", func(t *testing.T) {
		position := buildPosition(10, 2)
		searchParams := NewSearchParams("Emu", position, "emu.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find struct declaration in function return type", func(t *testing.T) {
		position := buildPosition(9, 4)
		searchParams := NewSearchParams("Emu", position, "emu.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find local struct member variable definition", func(t *testing.T) {

		position := buildPosition(27, 11) // Cursor at emulator.o|n = true
		doc := documents["emu.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should find local struct member variable definition when struct is a pointer", func(t *testing.T) {

		position := buildPosition(36, 9)
		doc := documents["emu.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "cpu", resolvedSymbol.GetName())
		assert.Equal(t, "Cpu", variable.GetType())
	})

	t.Run("Should find struct method", func(t *testing.T) {
		doc := documents["emu.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, buildPosition(32, 10))

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)
		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "tick", fun.GetName())
		assert.Equal(t, "Emu.tick", fun.GetFullName())
	})

	t.Run("Should find local struct method when there are N nested structs", func(t *testing.T) {
		// No pointer scenario
		position := buildPosition(26, 20)
		doc := documents["emu.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct method not found")

		fun, ok := resolvedSymbol.(*idx.Function)
		assert.True(t, ok, "Struct method not found")
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should find local struct method definition", func(t *testing.T) {
		position := buildPosition(27, 11)
		doc := documents["emu.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should not find local struct method definition", func(t *testing.T) {
		doc := documents["emu.c3"]
		// Cursor is at emu.audio.u|nknown
		position := buildPosition(38, 16)
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.Nil(t, resolvedSymbol, "Struct method not found")
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

	t.Run("Should find interface struct is implementing", func(t *testing.T) {
		position := buildPosition(8, 14)
		searchParams := NewSearchParams("EmulatorConsole", position, "emu.c3")

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_interface, ok := resolvedSymbol.(idx.Interface)

		assert.True(t, ok, "Element found should be an Interface")
		assert.Equal(t, "EmulatorConsole", _interface.GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_enums(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find local enum variable definition", func(t *testing.T) {
		position := buildPosition(11, 18)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "status", resolvedSymbol.GetName())
		assert.Equal(t, "WindowStatus", variable.GetType().String())
	})

	t.Run("Should find enum definition", func(t *testing.T) {
		position := buildPosition(11, 5)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		enum := resolvedSymbol.(idx.Enum)
		assert.Equal(t, "WindowStatus", enum.GetName())
	})

	t.Run("Should find local enumerator definition", func(t *testing.T) {
		position := buildPosition(12, 27)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_, ok := resolvedSymbol.(idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(resolvedSymbol)))
		assert.Equal(t, "BACKGROUND", resolvedSymbol.GetName())
	})

	t.Run("Should find enum method definition", func(t *testing.T) {
		t.Skip()
	})
}

func TestLanguage_findClosestSymbolDeclaration_faults(t *testing.T) {
	language, docs := initTestEnv()

	t.Run("Find local fault variable definition", func(t *testing.T) {
		position := buildPosition(17, 5)
		doc := docs["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Fault not found")

		fault := resolvedSymbol.(idx.Fault)
		assert.Equal(t, "WindowError", fault.GetName())
	})

	t.Run("Should find fault constant definition", func(t *testing.T) {
		position := buildPosition(17, 37)
		doc := docs["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")
		_, ok := resolvedSymbol.(idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(resolvedSymbol)))
		assert.Equal(t, "SOMETHING_HAPPENED", resolvedSymbol.GetName())
	})
}

func TestLanguage_findClosestSymbolDeclaration_def(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find local definition definition", func(t *testing.T) {
		position := buildPosition(2, 2)
		doc := documents["definitions.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Def not found")
		assert.Equal(t, "Kilo", resolvedSymbol.GetName())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		position := buildPosition(10, 4)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "tick", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})
}

func TestLanguage_findClosestSymbolDeclaration_functions(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find local function definition", func(t *testing.T) {
		position := buildPosition(21, 5)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Local function not found")

		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "run", fun.GetName())
		assert.Equal(t, "void", fun.GetReturnType())
	})

	t.Run("Should not confuse function with virtual root scope function", func(t *testing.T) {

		position := buildPosition(25, 5)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams)

		assert.NotNil(t, resolvedSymbol, "Local function not found")

		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "main", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})
}
