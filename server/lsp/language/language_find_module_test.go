package language

import (
	"testing"

	"github.com/pherrymason/c3-lsp/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/stretchr/testify/assert"
)

func TestLanguage_findClosestSymbolDeclaration_in_same_or_submodules(t *testing.T) {
	language, documents := initTestEnv()

	t.Run("Find variable definition in same module, but different file", func(t *testing.T) {
		doc := documents["app.c3"]
		position := buildPosition(20, 5) // Cursor h|elpDisplayedTimes
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "helpDisplayedTimes", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
	})

	t.Run("Should find the right element when there is a different element with the same name up anywhere in the same module", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc(
			"a.c3",
			`module app;
			fn void main() {
				variable = 3;
			}
			
			fn void trap() {
				bool variable = false;
			}`,
		)
		state.registerDoc(
			"b.c3",
			`module app;
			int variable = 4;
			fn void foo() {
				char variable = 'c';
			}`,
		)

		doc := state.docs["a.c3"]
		position := buildPosition(3, 4) // Cursor v|variable
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, state.language.parsedModulesByDocument[doc.URI], position)

		symbolOption := state.language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "variable", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
		assert.Equal(t, idx.NewRange(1, 7, 1, 15), variable.GetIdRange())
		assert.Equal(t, "b.c3", variable.GetDocumentURI())
	})

	t.Run("resolve variable from implicit sub module", func(t *testing.T) {
		position := buildPosition(7, 21) // Cursor at BA|R_WEIGHT
		doc := documents["module_foo.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})
}

func TestLanguage_findClosestSymbolDeclaration_in_imported_modules(t *testing.T) {
	language, documents := initTestEnv()
	t.Run("resolve implicit variable from different module in different file", func(t *testing.T) {
		position := buildPosition(8, 21) // Cursor at BA|R_WEIGHT
		doc := documents["module_foo2.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("resolve explicit variable from explicit sub module", func(t *testing.T) {
		position := buildPosition(9, 28) // Cursor at foo::bar::D|EFAULT_BAR_COLOR;
		doc := documents["module_foo2.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "DEFAULT_BAR_COLOR", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("finds symbol in parent implicit module", func(t *testing.T) {
		position := buildPosition(6, 5) // Cursor at `B|ar`
		doc := documents["module_foo_bar_dashed.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "Bar", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("should not finds symbol in sibling implicit module", func(t *testing.T) {
		position := buildPosition(6, 5) // Cursor at `B|ar`
		doc := documents["module_foo_bar_dashed.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)
		//searchParams.SetSymbol("Circle")

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol should not be found")
	})

	t.Run("resolve properly when there are cyclic dependencies", func(t *testing.T) {
		// This test ask specifically for a symbol located in an imported module defined after another module that has a cyclic dependency.
		position := buildPosition(10, 6) // Cursor at `T|riangle`
		doc := documents["module_foo2.c3"]
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
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
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)

		symbolOption := language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "module_multiple_same_file.c3", symbol.GetDocumentURI())
		assert.Equal(t, "mario", symbol.GetModuleString())

		// Second search
		position = buildPosition(12, 12) // Cursor at `something(v|alue);`
		searchParams = search_params.BuildSearchBySymbolUnderCursor(&doc, language.parsedModulesByDocument[doc.URI], position)
		symbolOption = language.findClosestSymbolDeclaration(searchParams, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol = symbolOption.Get()
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "module_multiple_same_file.c3", symbol.GetDocumentURI())
		assert.Equal(t, "luigi", symbol.GetModuleString())
	})
}

func TestResolve_generic_module_parameters(t *testing.T) {
	state := NewTestState()

	state.registerDoc(
		"module.c3",
		`module foo_test(<Type1, Type2>);
		struct Foo
		{
			Type1 a;
		}
		fn Type2 test(Type2 b, Foo *foo)
		{
			return foo.a + b;
		}`,
	)

	position := buildPosition(6, 17) // Cursor at `fn Type2 test(T|ype2 b, Foo *foo)`
	doc := state.GetDoc("module.c3")
	searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, state.GetParsedModules(doc.URI), position)

	symbolOption := state.language.findClosestSymbolDeclaration(searchParams, debugger)

	assert.True(t, symbolOption.IsSome())

	genericParameter := symbolOption.Get()
	assert.Equal(t, "Type2", genericParameter.GetName())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), genericParameter.GetIdRange())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), genericParameter.GetDocumentRange())
}
