package language

import (
	"fmt"
	"reflect"
	"testing"

	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/stretchr/testify/assert"
)

func TestLanguage_findClosestSymbolDeclaration_access_path(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Should find enumerator with path definition", func(t *testing.T) {
		position := buildPosition(5, 38)
		doc := documents["enums.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(idx.Enumerator)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an enumerator, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "OPEN", symbolOption.Get().GetName())
	})

	t.Run("Should find fault constant definition with path definition", func(t *testing.T) {
		position := buildPosition(4, 37) // Cursor at `WindowError error = WindowError.S|OMETHING_HAPPENED;`
		doc := documents["faults.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Element not found")
		_, ok := symbolOption.Get().(idx.FaultConstant)
		assert.Equal(t, true, ok, fmt.Sprintf("The symbol is not an fault constant, %s was found", reflect.TypeOf(symbolOption.Get())))
		assert.Equal(t, "SOMETHING_HAPPENED", symbolOption.Get().GetName())
	})

	t.Run("Should find local struct member variable definition", func(t *testing.T) {
		position := buildPosition(19, 14) // Cursor at `emulator.o|n = true`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should find local struct member variable definition when struct is a pointer", func(t *testing.T) {

		position := buildPosition(24, 14) // Cursor at emulator.o|n = true
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(idx.StructMember)
		assert.Equal(t, "on", symbol.GetName())
		assert.Equal(t, "bool", variable.GetType())
	})

	t.Run("Should find same struct member declaration, when cursor is already in member declaration", func(t *testing.T) {
		position := buildPosition(12, 8) // Cursor at `bool o|n;`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(idx.StructMember)
		assert.Equal(t, "cpu", symbol.GetName())
		assert.Equal(t, "Cpu", variable.GetType())
	})

	t.Run("Should find struct method", func(t *testing.T) {
		// Cursor at `emulator.i|nit();`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(38, 14))

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable", func(t *testing.T) {
		// Cursor at `Emu.i|nit(&emulator);`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(39, 9))

		resolvedSymbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method when cursor is already in method declaration", func(t *testing.T) {
		// Cursor at `Emu.i|nit();`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(28, 13))

		resolvedSymbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct method when cursor is on chained returned from function", func(t *testing.T) {
		// Cursor at `newEmu().i|nit();`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(40, 15))

		resolvedSymbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		fun := resolvedSymbolOption.Get().(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Emu.init", fun.GetFullName())
	})

	t.Run("Should find struct member when cursor is on chained returned from function", func(t *testing.T) {
		// Cursor at `newEmu().i|nit();`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(41, 14))

		resolvedSymbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		member := resolvedSymbolOption.Get().(idx.StructMember)
		assert.Equal(t, "on", member.GetName())
	})

	t.Run("Should find local struct method when there are N nested structs", func(t *testing.T) {
		position := buildPosition(30, 14) // Cursor at `emu.audio.i|nit();``
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		resolvedSymbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, resolvedSymbolOption.IsNone(), "Struct method not found")

		fun, ok := resolvedSymbolOption.Get().(*idx.Function)
		assert.True(t, ok, "Struct method not found")
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should find struct method on alternative callable when there are N nested structs", func(t *testing.T) {
		// Cursor at `Audio.i|nit(&emu.audio);`
		doc := documents["structs.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], buildPosition(32, 11))

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)
		fun := symbolOption.Get().(*idx.Function)
		assert.Equal(t, "init", fun.GetName())
		assert.Equal(t, "Audio.init", fun.GetFullName())
	})

	t.Run("Should not find local struct method definition", func(t *testing.T) {
		doc := documents["structs.c3"]
		position := buildPosition(31, 16) // Cursor is at emu.audio.u|nknown
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.functionTreeByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.True(t, symbolOption.IsNone(), "Struct method not found")
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
}
