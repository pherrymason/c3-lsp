package search

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func TestProjectState_findClosestSymbolDeclaration_should_find_module(t *testing.T) {
	state := NewTestState()
	state.registerDoc(
		"origin.c3",
		`import other;
		import trap;`,
	)
	state.registerDoc(
		"other.c3",
		`module other;`,
	)
	state.registerDoc(
		"traps.c3",
		`module trap;
		fn void foo(bool other) {}
		struct Cpu {
			System* other;
		}`,
	)

	search := NewSearchWithoutLog()

	position := buildPosition(1, 8) // Cursor at `o|ther`
	doc := state.GetDoc("origin.c3")
	searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

	symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, FindDebugger{enabled: false, depth: 0})

	assert.True(t, symbolOption.IsSome())
	found := symbolOption.Get()
	module, ok := found.(*idx.Module)
	assert.True(t, ok, "Unexpected symbol resolved.")
	assert.Equal(t, "other", module.GetName())

}

func TestProjectState_findClosestSymbolDeclaration_in_same_or_submodules(t *testing.T) {
	search := NewSearchWithoutLog()

	t.Run("Find variable definition in same module, but different file", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc(
			"a.c3",
			`module app;
			fn void main() {
				helpDisplayedTimes = 1;
			}`,
		)
		state.registerDoc(
			"b.c3",
			`module app;
			int helpDisplayedTimes = 0;`,
		)

		//doc := documents["app.c3"]
		position := buildPosition(3, 5) // Cursor h|elpDisplayedTimes
		doc := state.GetDoc("a.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc("a.c3"), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, FindDebugger{enabled: false, depth: 0})

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
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.Variable)
		assert.Equal(t, "variable", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().String())
		assert.Equal(t, idx.NewRange(1, 7, 1, 15), variable.GetIdRange())
		assert.Equal(t, "b.c3", variable.GetDocumentURI())
	})

	t.Run("Should find the right element when there is a different element with the same name up anywhere in the same parent module", func(t *testing.T) {
		state := NewTestState()

		state.registerDoc(
			"libc.c3",
			`module std::libc;
			fn void prinf(char *msg, int args) {}`,
		)
		state.registerDoc(
			"io.c3",
			`module std::io;
			fn void printf(char* msg) {}`,
		)
		state.registerDoc(
			"user.c3",
			`module app;
			import std::io;
			fn void main() {
				printf("blabalbal");
			}
			`,
		)

		doc := state.docs["user.c3"]
		position := buildPosition(4, 5) // Cursor p|rintf("blabalbal");
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		fun := symbol.(*idx.Function)
		assert.Equal(t, "printf", symbol.GetName())
		assert.Equal(t, "std::io", fun.GetModuleString())
		assert.Equal(t, "io.c3", fun.GetDocumentURI())
	})

	t.Run("resolve variable from implicit sub module", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc("module_foo.c3",
			`module foo;

		int value = 1;
		
		fn void shapes() {
			Bar mybar;
			mybar.weight = BAR_WEIGHT;
			mybar.color = foo::bar::DEFAULT_BAR_COLOR;
			Circle mycircle;
		}`)
		state.registerDoc(
			"module_foo_bar.c3",
			`module foo::bar;

		const int BAR_WEIGHT = 1;
		const int DEFAULT_BAR_COLOR = 0;
		struct Bar {
			int width;
			int weight;
			int color;
		}`)

		position := buildPosition(7, 20) // Cursor at BA|R_WEIGHT
		doc := state.docs["module_foo.c3"]

		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	// This test is testing both accessPath and module path
	t.Run("resolve struct member from implicit sub module", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc("raylib.c3",
			`struct Camera3D {
			int target;
		}
		def Camera = Camera3D;`)
		state.registerDoc(
			"structs.c3",
			`module structs;
			import raylib;
			struct Widget {
				int count;
				raylib::Camera3D camera;
			}
			
			Widget view = {};
			view.camera.target = 3;
			`,
		)
		position := buildPosition(9, 16) // Cursor at `view.camera.t|arget = 3;`
		doc := state.GetDoc("structs.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()

		variable := symbol.(*idx.StructMember)
		assert.Equal(t, "target", symbol.GetName())
		assert.Equal(t, "int", variable.GetType().GetName())
	})
}

func TestProjectState_findClosestSymbolDeclaration_should_find_types_referenced_implicitly_from_imported_modules(t *testing.T) {
	search := NewSearchWithoutLog()

	t.Run("resolves struct member type", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc(
			"external.c3",
			`module external;
			def Color = int;`,
		)
		state.registerDoc(
			"main.c3",
			`module main;
			import external;
			struct MyStruct {
				Color color;
			}`,
		)
		position := buildPosition(4, 11) // Cursor at Color c|olor;
		doc := state.GetDoc("main.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get().(*idx.StructMember)
		assert.Equal(t, "color", symbol.GetName())
		assert.Equal(t, "external::Color", symbol.GetType().GetFullQualifiedName(), "Color module was not properly infered.")
	})

	t.Run("resolves struct member type even when reference was registered later", func(t *testing.T) {
		state := NewTestState()
		state.registerDoc(
			"main.c3",
			`module main;
			import external;
			struct MyStruct {
				Color color;
			}`,
		)
		state.registerDoc(
			"external.c3",
			`module external;
			def Color = int;`,
		)
		position := buildPosition(4, 11) // Cursor at Color c|olor;
		doc := state.GetDoc("main.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get().(*idx.StructMember)
		assert.Equal(t, "color", symbol.GetName())
		assert.Equal(t, "external::Color", symbol.GetType().GetFullQualifiedName(), "Color module was not properly infered.")
	})
}

func TestProjectState_findClosestSymbolDeclaration_in_imported_modules(t *testing.T) {
	search := NewSearchWithoutLog()
	state := NewTestState()

	t.Run("resolve implicit variable from different module in different file", func(t *testing.T) {
		state.registerDoc(
			"app.c3",
			`import foo::bar;
			fn void main() {
				Bar mybar;
				mybar.weight = BAR_WEIGHT;
			}`,
		)
		state.registerDoc(
			"foobar.c3",
			`module foo::bar;
			const int BAR_WEIGHT = 1;`,
		)

		position := buildPosition(4, 21) // Cursor at BA|R_WEIGHT
		doc := state.state.GetDocument("app.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "BAR_WEIGHT", symbol.GetName())
		assert.Equal(t, "foobar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("resolve explicit variable from explicit sub module", func(t *testing.T) {
		state.registerDoc(
			"origin.c3",
			`import foo::bar;
			fn void main() {
				Bar mybar;
				mybar.color = foo::bar::DEFAULT_BAR_COLOR;
			}`,
		)
		state.registerDoc(
			"module_foo_bar.c3",
			`module foo::bar;
			const int DEFAULT_BAR_COLOR = 1;`,
		)

		position := buildPosition(4, 28) // Cursor at foo::bar::D|EFAULT_BAR_COLOR;
		doc := state.state.GetDocument("origin.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "DEFAULT_BAR_COLOR", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("finds symbol in parent implicit module", func(t *testing.T) {
		state.registerDoc(
			"dashed.c3",
			`module foo::bar::dashed;

			const int BAR_DASHED_WEIGHT = 1;
			const int DEFAULT_BAR_DASHED_COLOR = 0;
			struct Bardashed {
				Bar bar;
				int interspace;
			}`,
		)
		state.registerDoc(
			"module_foo_bar.c3",
			`module foo::bar;
			struct Bar {
				int width;
				int weight;
				int color;
			}`,
		)

		position := buildPosition(6, 5) // Cursor at `B|ar`
		doc := state.state.GetDocument("dashed.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "Bar", symbol.GetName())
		assert.Equal(t, "module_foo_bar.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::bar", symbol.GetModuleString())
	})

	t.Run("resolve properly when there are cyclic dependencies", func(t *testing.T) {
		state.registerDoc(
			"foobar2.c3",
			`module foo2;
			import foo::bar;
			import cyclic;
			import foo::triangle;

			fn void main() {
				Triangle triangle;
			}`,
		)
		state.registerDoc("cyclic.c3",
			`module cyclic;
			import foo2;

			struct Cyclic {}`,
		)
		state.registerDoc("triangle.c3",
			`module foo::triangle;
			// This module is to cause cyclic dependencies.
			struct Triangle {}`,
		)

		// This test ask specifically for a symbol located in an imported module defined after another module that has a cyclic dependency.
		position := buildPosition(7, 6) // Cursor at `T|riangle`
		doc := state.state.GetDocument("foobar2.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.False(t, symbolOption.IsNone(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "Triangle", symbol.GetName())
		assert.Equal(t, "triangle.c3", symbol.GetDocumentURI())
		assert.Equal(t, "foo::triangle", symbol.GetModuleString())
	})

	t.Run("resolve properly when there are cyclic dependencies in parent modules", func(t *testing.T) {
		t.Skip()
	})

	t.Run("resolve properly when file_contains_multiple_modules", func(t *testing.T) {
		state.registerDoc(
			"multiple.c3",
			`module mario;
			int value = 1;

			fn void shapes() {
				something(value);
			}

			module luigi;
			int value = 2;
			something(value);`,
		)

		// This test ask specifically for a symbol located in an imported module defined after another module that has a cyclic dependency.
		position := buildPosition(5, 16) // Cursor at `something(v|alue);`
		doc := state.state.GetDocument("multiple.c3")
		searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

		symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.True(t, symbolOption.IsSome(), "Symbol not found")
		symbol := symbolOption.Get()
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "multiple.c3", symbol.GetDocumentURI())
		assert.Equal(t, "mario", symbol.GetModuleString())

		// Second search
		position = buildPosition(10, 14) // Cursor at `something(v|alue);`
		searchParams = search_params.BuildSearchBySymbolUnderCursor(doc, *state.state.GetUnitModulesByDoc(doc.URI), position)
		symbolOption = search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

		assert.True(t, symbolOption.IsSome(), "Symbol not found")
		symbol = symbolOption.Get()
		assert.Equal(t, "value", symbol.GetName())
		assert.Equal(t, "multiple.c3", symbol.GetDocumentURI())
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
	searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

	search := NewSearchWithoutLog()
	symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

	assert.True(t, symbolOption.IsSome())

	genericParameter := symbolOption.Get()
	assert.Equal(t, "Type2", genericParameter.GetName())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), genericParameter.GetIdRange())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), genericParameter.GetDocumentRange())
}

func TestProjectState_findClosestSymbolDeclaration_should_find_right_module_when_partial_name_of_module_is_used(t *testing.T) {
	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("C3-LSP.parser")
	state := NewTestState(logger)

	state.registerDoc(
		"app.c3",
		`import mystd::io;
		io::printf("Hello world");
		`,
	)
	state.registerDoc(
		"trap.c3",
		`module io;
		fn void printf(*char input) {}
		`,
	)
	state.registerDoc(
		"io.c3",
		`module mystd::io;
		fn void printf(*char input) {}
		`,
	)

	position := buildPosition(1, 15) // Cursor at import mystd::i|o;
	doc := state.docs["app.c3"]
	searchParams := search_params.BuildSearchBySymbolUnderCursor(&doc, *state.state.GetUnitModulesByDoc(doc.URI), position)

	search := NewSearchWithoutLog()
	symbolOption := search.findClosestSymbolDeclaration(searchParams, &state.state, debugger)

	assert.True(t, symbolOption.IsSome(), "Element not found")

	mod := symbolOption.Get().(*idx.Module)
	assert.Equal(t, "mystd::io", mod.GetName())
	//assert.Equal(t, idx.FunctionType(idx.UserDefined), mod.FunctionType())
}
