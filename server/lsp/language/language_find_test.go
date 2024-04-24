package language

import (
	"fmt"
	"os"
	"path/filepath"
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

func installDocuments(language *Language, parser *p.Parser) map[string]document.Document {
	var fileContent string

	filenames := []string{"app.c3", "app_helper.c3", "emu.c3", "definitions.c3"}
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
	/*
		fileContent = readC3File("./test_files/app.c3")
		documents["app.c3"] = document.NewDocument("app.c3", "?", fileContent)

		fileContent = readC3File("./test_files/emu.c3")
		documents["emu.c3"] = document.NewDocument("emu.c3", "?", fileContent)

		fileContent = readC3File("./test_files/definitions.c3")
		documents["definitions.c3"] = document.NewDocument("definitions.c3", "?", fileContent)

		for _, value := range documents {
			language.RefreshDocumentIdentifiers(&value, parser)
		}
	*/
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

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		searchParams := NewSearchParams("emulator", "emu.c3")
		position := buildPosition(10, 9)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		searchParams := NewSearchParams("emulator", "emu.c3")
		position := buildPosition(12, 10)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType())
	})

	t.Run("should find struct declaration in variable declaration", func(t *testing.T) {
		searchParams := NewSearchParams("Emu", "emu.c3")
		position := buildPosition(10, 2)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("should find struct declaration in function return type", func(t *testing.T) {
		searchParams := NewSearchParams("Emu", "emu.c3")
		position := buildPosition(9, 4)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Find local struct member variable definition", func(t *testing.T) {

		position := buildPosition(11, 11)
		doc := documents["emu.c3"]
		// Note: Here we use buildSearchParams instead of NewSearchParams because buildSearchParams has some logic to identify that the searchTerm has a '.'.
		searchParams, _ := buildSearchParams(&doc, position)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Find local enum variable definition", func(t *testing.T) {
		searchParams := NewSearchParams("status", "app.c3")
		position := buildPosition(8, 5)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "status", resolvedSymbol.GetName())
		assert.Equal(t, "WindowStatus", variable.GetType())
	})
	t.Run("Find local enumerator definition", func(t *testing.T) {
		searchParams := NewSearchParams("BACKGROUND", "app.c3")
		position := buildPosition(8, 17)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		assert.Equal(t, "BACKGROUND", resolvedSymbol.GetName())
	})

	t.Run("Find local definition definition", func(t *testing.T) {
		searchParams := NewSearchParams("Kilo", "definitions.c3")
		position := buildPosition(2, 2)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Def not found")
		assert.Equal(t, "Kilo", resolvedSymbol.GetName())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		searchParams := NewSearchParams("tick", "app.c3")
		position := buildPosition(6, 4)

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "tick", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType())
	})

	t.Run("Find local function definition", func(t *testing.T) {
		searchParams := NewSearchParams("run", "app.c3")
		position := buildPosition(13, 5)

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

func TestLanguage_findClosestSymbolDeclaration_in_same_module(t *testing.T) {
	language, _ := initTestEnv()

	t.Run("Find variable definition in same module", func(t *testing.T) {
		searchParams := NewSearchParams("Cpu", "emu.c3")
		position := protocol.Position{2, 2}

		resolvedSymbol := language.findClosestSymbolDeclaration(searchParams, position)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType())
	})
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
