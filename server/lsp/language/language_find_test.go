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

	filenames := []string{
		"app.c3",
		"app_helper.c3",
		"emu.c3",
		"definitions.c3",
		"cpu.c3",
		// Structs related
		"structs.c3",

		// Module related sources
		"module_foo.c3",
		"module_foo_bar.c3",
		"module_foo_bar_dashed.c3",
		"module_foo_circle.c3",
		"module_foo2.c3",
		"module_cyclic.c3",
		"module_foo_triangle.c3",
		"module_multiple_same_file.c3",
	}
	baseDir := "./../../test_files/"
	documents := make(map[string]document.Document, 0)

	for _, filename := range filenames {
		fullPath := filepath.Join(baseDir, filename)
		fileContent = readC3File(fullPath)
		documents[filename] = document.NewDocument(filename, "?", fileContent)
		doc := documents[filename]
		language.RefreshDocumentIdentifiers(&doc, parser)
	}

	return documents
}

func createParser() p.Parser {
	logger := &commonlog.MockLogger{}
	return p.NewParser(logger)
}

func initTestEnv() (*Language, map[string]document.Document) {
	parser := createParser()
	language := NewLanguage(commonlog.MockLogger{})

	documents := installDocuments(&language, &parser)

	return &language, documents
}

func buildPosition(line protocol.UInteger, character protocol.UInteger) protocol.Position {
	return protocol.Position{Line: line - 1, Character: character}
}

func find(language *Language, searchParams SearchParams) idx.Indexable {
	return language.findClosestSymbolDeclaration(searchParams, NewFindDebugger(true))
}

func TestLanguage_findClosestSymbolDeclaration_ignores_keywords(t *testing.T) {
	cases := []struct {
		source string
	}{
		{"void"}, {"bool"}, {"char"}, {"double"},
		{"float"}, {"float16"}, {"int128"}, {"ichar"},
		{"int"}, {"iptr"}, {"isz"}, {"long"},
		{"short"}, {"uint128"}, {"uint"}, {"ulong"},
		{"uptr"}, {"ushort"}, {"usz"}, {"float128"},
		{"any"}, {"anyfault"}, {"typeid"}, {"assert"},
		{"asm"}, {"bitstruct"}, {"break"}, {"case"},
		{"catch"}, {"const"}, {"continue"}, {"def"},
		{"default"}, {"defer"}, {"distinct"}, {"do"},
		{"else"}, {"enum"}, {"extern"}, {"false"},
		{"fault"}, {"for"}, {"foreach"}, {"foreach_r"},
		{"fn"}, {"tlocal"}, {"if"}, {"inline"},
		{"import"}, {"macro"}, {"module"}, {"nextcase"},
		{"null"}, {"return"}, {"static"}, {"struct"},
		{"switch"}, {"true"}, {"try"}, {"union"},
		{"var"}, {"while"},
		{"$alignof"}, {"$assert"}, {"$case"}, {"$default"},
		{"$defined"}, {"$echo"}, {"$embed"}, {"$exec"},
		{"$else"}, {"$endfor"}, {"$endforeach"}, {"$endif"},
		{"$endswitch"}, {"$eval"}, {"$evaltype"}, {"$error"},
		{"$extnameof"}, {"$for"}, {"$foreach"}, {"$if"},
		{"$include"}, {"$nameof"}, {"$offsetof"}, {"$qnameof"},
		{"$sizeof"}, {"$stringify"}, {"$switch"}, {"$typefrom"},
		{"$typeof"}, {"$vacount"}, {"$vatype"}, {"$vaconst"},
		{"$varef"}, {"$vaarg"}, {"$vaexpr"}, {"$vasplat"},
	}
	parser := createParser()
	logger := &MockLogger{
		tracker: make(map[string][]string),
	}
	language := NewLanguage(logger)
	doc := document.NewDocument("x", "?", "module foo;")
	language.RefreshDocumentIdentifiers(&doc, &parser)

	doc = document.NewDocument("z", "?", "module bar;import foo;")
	language.RefreshDocumentIdentifiers(&doc, &parser)

	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			logger.tracker = make(map[string][]string)
			doc := document.NewDocument("y", "?", "module foo;"+tt.source)
			language.RefreshDocumentIdentifiers(&doc, &parser)
			position := buildPosition(1, 12) // Cursor at BA|R_WEIGHT
			searchParams, _ := NewSearchParamsFromPosition(&doc, position)

			debugger := NewFindDebugger(true)
			symbol := language.findClosestSymbolDeclaration(searchParams, debugger)

			assert.Nil(t, symbol, "Symbol should not be found")
			assert.Equal(t, 1, len(logger.tracker["debug"]))
			assert.Equal(t, "| Ignore because C3 keyword", logger.tracker["debug"][0])
		})
	}
}

func TestLanguage_findClosestSymbolDeclaration_in_same_scope(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("resolve implicit variable from different module in different file", func(t *testing.T) {
		position := buildPosition(8, 21) // Cursor at BA|R_WEIGHT
		doc := documents["module_foo2.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("resolve explicit variable from explicit sub module", func(t *testing.T) {
		position := buildPosition(9, 28) // Cursor at foo::bar::D|EFAULT_BAR_COLOR;
		doc := documents["module_foo2.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "DEFAULT_BAR_COLOR", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("resolve variable from implicit sub module", func(t *testing.T) {
		position := buildPosition(7, 21) // Cursor at BA|R_WEIGHT
		doc := documents["module_foo.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("finds symbol in parent implicit module", func(t *testing.T) {
		position := buildPosition(6, 5) // Cursor at `B|ar`
		doc := documents["module_foo_bar_dashed.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "Bar", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("should not finds symbol in sibling implicit module", func(t *testing.T) {
		position := buildPosition(6, 5) // Cursor at `B|ar`
		doc := documents["module_foo_bar_dashed.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)
		searchParams.selectedSymbol.Token = "Circle"

		symbol := find(language, searchParams)

		assert.Nil(t, symbol, "Symbol should not be found")
	})

	t.Run("resolve properly when there are cyclic dependencies", func(t *testing.T) {
		// This test ask specifically for a symbol located in an imported module defined after another module that has a cyclic dependency.
		position := buildPosition(10, 6) // Cursor at `T|riangle`
		doc := documents["module_foo2.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "Triangle", symbol.GetName())
		assert.Equal(t, "module_foo_triangle.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::triangle", symbol.GetModuleString())
	})

	t.Run("resolve properly when there are cyclic dependencies in parent modules", func(t *testing.T) {
		t.Skip()
	})

	t.Run("resolve properly when file_contains_multiple_modules", func(t *testing.T) {
		// This test ask specifically for a symbol located in an imported module defined after another module that has a cyclic dependency.
		position := buildPosition(6, 16) // Cursor at `something(v|alue);`
		doc := documents["module_multiple_same_file.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		symbol := find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "module_multiple_same_file.c3", symbol.GetDocumentURI())
		assert.Equal(t, "mario", symbol.GetModuleString())

		// Second search
		position = buildPosition(12, 12) // Cursor at `something(v|alue);`
		searchParams, _ = NewSearchParamsFromPosition(&doc, position)
		symbol = find(language, searchParams)

		assert.NotNil(t, symbol, "Symbol not found")
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "module_multiple_same_file.c3", symbol.GetDocumentURI())
		assert.Equal(t, "luigi", symbol.GetModuleString())

	})

}

func TestLanguage_findClosestSymbolDeclaration_variables(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find local variable definition, with cursor in same declaration", func(t *testing.T) {
		doc := documents["emu.c3"]
		position := buildPosition(24, 9) // Cursor at `e|mulator`
		//searchParams := NewSearchParams("emulator", position, "emu.c3")
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Symbol not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Find local variable definition from usage", func(t *testing.T) {
		doc := documents["emu.c3"]
		position := buildPosition(24, 10)
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "emulator", resolvedSymbol.GetName())
		assert.Equal(t, "Emu", variable.GetType().String())
	})

	t.Run("Should find the right element when there is a different element with the same name up in the scope", func(t *testing.T) {
		doc := documents["app.c3"]
		position := buildPosition(16, 9) // Cursor a|mbiguousVariable
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "ambiguousVariable", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Find variable definition in same module, but different file", func(t *testing.T) {
		doc := documents["app.c3"]
		position := buildPosition(20, 5) // Cursor h|elpDisplayedTimes
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "helpDisplayedTimes", resolvedSymbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})
}

// Tests related to structs:
func TestLanguage_findClosestSymbolDeclaration_structs(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Should find struct declaration in variable declaration", func(t *testing.T) {
		position := buildPosition(18, 4) // Cursor at `Emu e|mulator`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find struct declaration in function return type", func(t *testing.T) {
		position := buildPosition(17, 4) // Cursor at `fn E|mu NewEmu() {`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		_struct := resolvedSymbol.(idx.Struct)
		assert.Equal(t, "Emu", _struct.GetName())
	})

	t.Run("Should find local struct member variable definition", func(t *testing.T) {
		position := buildPosition(19, 14) // Cursor at `emulator.o|n = true`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should find local struct member variable definition when struct is a pointer", func(t *testing.T) {

		position := buildPosition(24, 14) // Cursor at emulator.o|n = true
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "on", resolvedSymbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should find same struct member declaration, when cursor is already in member declaration", func(t *testing.T) {
		position := buildPosition(12, 8) // Cursor at `emulator.o|n = true`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct member not found")

		variable := resolvedSymbol.(idx.StructMember)
		assert.Equal(t, "cpu", resolvedSymbol.GetName())
		assert.Equal(t, "Cpu", variable.GetType())
	})

	t.Run("Should find struct method", func(t *testing.T) {
		// Cursor at `emulator.i|nit();`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, buildPosition(38, 14))

		resolvedSymbol := find(language, searchParams)
		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable", func(t *testing.T) {
		// Cursor at `Emu.i|nit(&emulator);`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, buildPosition(39, 9))

		resolvedSymbol := find(language, searchParams)
		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method when cursor is already in method declaration", func(t *testing.T) {
		// Cursor at `Emu.i|nit();`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, buildPosition(28, 13))

		resolvedSymbol := find(language, searchParams)
		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find local struct method when there are N nested structs", func(t *testing.T) {
		position := buildPosition(30, 14) // Cursor at `emu.audio.i|nit();``
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Struct method not found")

		fun, ok := resolvedSymbol.(*idx.Function)
		assert.True(t, ok, "Struct method not found")
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable when there are N nested structs", func(t *testing.T) {
		// Cursor at `Audio.i|nit(&emu.audio);`
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, buildPosition(32, 11))

		resolvedSymbol := find(language, searchParams)
		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should not find local struct method definition", func(t *testing.T) {
		doc := documents["structs.c3"]
		position := buildPosition(31, 16) // Cursor is at emu.audio.u|nknown
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

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
		position := buildPosition(11, 14) // Cursor is at struct Emu (E|mulatorConsole) {
		doc := documents["structs.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

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

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		variable := resolvedSymbol.(idx.Variable)
		assert.Equal(t, "status", resolvedSymbol.GetName())
		assert.Equal(t, "WindowStatus", variable.GetType().String())
	})

	t.Run("Should find enum definition", func(t *testing.T) {
		position := buildPosition(11, 5)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Element not found")

		enum := resolvedSymbol.(idx.Enum)
		assert.Equal(t, "WindowStatus", enum.GetName())
	})

	t.Run("Should find local enumerator definition", func(t *testing.T) {
		position := buildPosition(12, 27)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

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

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Fault not found")

		fault := resolvedSymbol.(idx.Fault)
		assert.Equal(t, "WindowError", fault.GetName())
	})

	t.Run("Should find fault constant definition", func(t *testing.T) {
		position := buildPosition(17, 37)
		doc := docs["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

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

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Def not found")
		assert.Equal(t, "Kilo", resolvedSymbol.GetName())
	})

	t.Run("Find local variable definition in function arguments", func(t *testing.T) {
		position := buildPosition(10, 4)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

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

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Local function not found")

		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "run", fun.GetName())
		assert.Equal(t, "void", fun.GetReturnType())
	})

	t.Run("Should not confuse function with virtual root scope function", func(t *testing.T) {

		position := buildPosition(25, 5)
		doc := documents["app.c3"]
		searchParams, _ := NewSearchParamsFromPosition(&doc, position)

		resolvedSymbol := find(language, searchParams)

		assert.NotNil(t, resolvedSymbol, "Local function not found")

		fun := resolvedSymbol.(*idx.Function)
		assert.Equal(t, "main", fun.GetName())
		assert.Equal(t, idx.FunctionType(idx.UserDefined), fun.FunctionType())
	})
}
