package parser

import (
	"fmt"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/commonlog"
)

func createParser() Parser {
	logger := &commonlog.MockLogger{}
	return NewParser(logger)
}

func TestParses_empty_document(t *testing.T) {
	doc := document.NewDocument("empty", "")
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	assert.Equal(t, 0, len(symbols.ModuleIds()))
}

func TestParses_TypedEnums(t *testing.T) {
	docId := "doc"
	source := `
	<* abc *>
	enum Colors:int { RED, BLUE, GREEN }
	fn bool Colors.hasRed(Colors color)
	{}
	`
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds Colors enum identifier", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		module := symbols.Get("doc")

		assert.NotNil(t, module.Enums["Colors"])
		assert.Equal(t, "Colors", module.Enums["Colors"].GetName())
		assert.Equal(t, "int", module.Enums["Colors"].GetType())
		assert.Same(t, module.Enums["Colors"], module.Children()[0])
	})

	t.Run("reads ranges for enum", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("doc")
		enum := scope.Enums["Colors"]

		assert.Equal(t, idx.NewRange(2, 1, 2, 37), enum.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(2, 6, 2, 12), enum.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds doc comment", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("doc")
		enum := scope.Enums["Colors"]

		assert.Equal(t, "abc", enum.GetDocComment().GetBody())
	})

	t.Run("finds defined enumerators", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		enum := symbols.Get("doc").Enums["Colors"]
		e := enum.GetEnumerator("RED")
		assert.Equal(t, "RED", e.GetName())
		assert.Equal(t, idx.NewRange(2, 19, 2, 22), e.GetIdRange())
		assert.Equal(t, "Colors", e.GetEnumName())
		assert.Same(t, enum.Children()[0], e)

		e = enum.GetEnumerator("BLUE")
		assert.Equal(t, "BLUE", e.GetName())
		assert.Equal(t, idx.NewRange(2, 24, 2, 28), e.GetIdRange())
		assert.Equal(t, "Colors", e.GetEnumName())
		assert.Same(t, enum.Children()[1], e)

		e = enum.GetEnumerator("GREEN")
		assert.Equal(t, "GREEN", e.GetName())
		assert.Equal(t, idx.NewRange(2, 30, 2, 35), e.GetIdRange())
		assert.Equal(t, "Colors", e.GetEnumName())
		assert.Same(t, enum.Children()[2], e)
	})

	t.Run("associate values >= v0.6.0", func(t *testing.T) {
		source := `
		enum State : int (String state_desc, bool active)
		{
			PENDING = {"pending start", false},
			RUNNING = {"running", true},
			TERMINATED = {"ended", false}
		}`
		doc := document.NewDocument("ass.c3", source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("ass")
		assert.NotNil(t, scope.Enums["State"])
		enum := scope.Enums["State"]
		enumerators := enum.GetEnumerators()

		assert.Len(t, enumerators, 3, "Missing enumerators")

		expectedAssocValues := []struct {
			type_ string
			name  string
		}{
			{
				type_: "String",
				name:  "state_desc",
			},
			{
				type_: "bool",
				name:  "active",
			},
		}

		t.Run("GetAssociatedValues", func(t *testing.T) {
			assocs := enum.GetAssociatedValues()
			assert.Equal(t, len(expectedAssocValues), len(assocs))
			for i, assoc := range assocs {
				assocIndex := fmt.Sprintf("Associated value #%d", i)
				expected := expectedAssocValues[i]
				assert.Equal(t, expected.name, assoc.GetName(), assocIndex+" didn't match")
				assert.Equal(t, expected.type_, assoc.GetType().GetName(), assocIndex+" didn't match")
			}
		})

		for enum_i, enumerator := range enumerators {
			t.Run(fmt.Sprintf("Enumerator #%d", enum_i), func(t *testing.T) {
				assert.Equal(t, len(expectedAssocValues), len(enumerator.AssociatedValues))
				for i, assoc := range enumerator.AssociatedValues {
					assocIndex := fmt.Sprintf("Associated value #%d", i)
					expected := expectedAssocValues[i]
					assert.Equal(t, expected.name, assoc.GetName(), assocIndex+" didn't match")
					assert.Equal(t, expected.type_, assoc.GetType().GetName(), assocIndex+" didn't match")
				}
			})
		}
	})

	t.Run("associated values >= 0.6.0 without backing type", func(t *testing.T) {
		source := `
		enum State : (String state_desc, bool active)
		{
			PENDING = {"pending start", false},
			RUNNING = {"running", true},
			TERMINATED = {"ended", false}
		}`
		doc := document.NewDocument("ass.c3", source)
		parser := createParser()

		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("ass")
		assert.NotNil(t, scope.Enums["State"])
		enum := scope.Enums["State"]
		enumerators := enum.GetEnumerators()

		assert.Len(t, enumerators, 3, "Missing enumerators")

		expectedAssocValues := []struct {
			type_ string
			name  string
		}{
			{
				type_: "String",
				name:  "state_desc",
			},
			{
				type_: "bool",
				name:  "active",
			},
		}

		t.Run("GetAssociatedValues", func(t *testing.T) {
			assocs := enum.GetAssociatedValues()
			assert.Equal(t, len(expectedAssocValues), len(assocs))
			for i, assoc := range assocs {
				assocIndex := fmt.Sprintf("Associated value #%d", i)
				expected := expectedAssocValues[i]
				assert.Equal(t, expected.name, assoc.GetName(), assocIndex+" didn't match")
				assert.Equal(t, expected.type_, assoc.GetType().GetName(), assocIndex+" didn't match")
			}
		})

		for enum_i, enumerator := range enumerators {
			t.Run(fmt.Sprintf("Enumerator #%d", enum_i), func(t *testing.T) {
				assert.Equal(t, len(expectedAssocValues), len(enumerator.AssociatedValues))
				for i, assoc := range enumerator.AssociatedValues {
					assocIndex := fmt.Sprintf("Associated value #%d", i)
					expected := expectedAssocValues[i]
					assert.Equal(t, expected.name, assoc.GetName(), assocIndex+" didn't match")
					assert.Equal(t, expected.type_, assoc.GetType().GetName(), assocIndex+" didn't match")
				}
			})
		}
	})

	t.Run("finds enum method", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		f := symbols.Get("doc").GetChildrenFunctionByName("Colors.hasRed")
		assert.True(t, f.IsSome())
	})
}

func TestParses_UnTypedEnums(t *testing.T) {
	docId := "doc"
	source := `<*
		abc
	*>
	enum Colors { RED, BLUE, GREEN };`
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds Colors enum identifier", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("doc")

		assert.NotNil(t, scope.Enums["Colors"])
		assert.Equal(t, "Colors", scope.Enums["Colors"].GetName())
		assert.Equal(t, "", scope.Enums["Colors"].GetType())
		assert.Same(t, scope.Children()[0], scope.Enums["Colors"])
	})

	t.Run("reads ranges for enum", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		enum := symbols.Get("doc").Enums["Colors"]
		assert.Equal(t, idx.NewRange(3, 1, 3, 33), enum.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(3, 6, 3, 12), enum.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds doc comment", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("doc")
		enum := scope.Enums["Colors"]

		assert.Equal(t, "abc", enum.GetDocComment().GetBody())
	})

	t.Run("finds defined enumerators", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		enum := symbols.Get("doc").Enums["Colors"]
		e := enum.GetEnumerator("RED")
		assert.Equal(t, "RED", e.GetName())
		assert.Equal(t, idx.NewRange(3, 15, 3, 18), e.GetIdRange())

		e = enum.GetEnumerator("BLUE")
		assert.Equal(t, "BLUE", e.GetName())
		assert.Equal(t, idx.NewRange(3, 20, 3, 24), e.GetIdRange())

		e = enum.GetEnumerator("GREEN")
		assert.Equal(t, "GREEN", e.GetName())
		assert.Equal(t, idx.NewRange(3, 26, 3, 31), e.GetIdRange())
	})
}

func TestParse_fault(t *testing.T) {
	docId := "doc"
	source := `<* docs *>
	fault IOResult
	{
	  IO_ERROR,
	  PARSE_ERROR
	};`

	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds Fault identifier", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		scope := symbols.Get("doc")
		assert.NotNil(t, scope.Faults["IOResult"])
		assert.Equal(t, "IOResult", scope.Faults["IOResult"].GetName())
		assert.Equal(t, "", scope.Faults["IOResult"].GetType())
		assert.Same(t, scope.Children()[0], scope.Faults["IOResult"])
	})

	t.Run("reads ranges for fault", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("doc").Faults["IOResult"]
		assert.Equal(t, idx.NewRange(1, 1, 5, 2), found.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(1, 7, 1, 15), found.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds doc comment", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fault := symbols.Get("doc").Faults["IOResult"]

		assert.Equal(t, "docs", fault.GetDocComment().GetBody())
	})

	t.Run("finds defined fault constants", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		fault := symbols.Get("doc").Faults["IOResult"]
		e := fault.GetConstant("IO_ERROR")
		assert.Equal(t, "IO_ERROR", e.GetName())
		assert.Equal(t, idx.NewRange(3, 3, 3, 11), e.GetIdRange())
		assert.Equal(t, idx.NewRange(3, 3, 3, 11), e.GetDocumentRange())
		assert.Equal(t, "IOResult", e.GetFaultName())
		assert.Same(t, fault.Children()[0], e)

		e = fault.GetConstant("PARSE_ERROR")
		assert.Equal(t, "PARSE_ERROR", e.GetName())
		assert.Equal(t, idx.NewRange(4, 3, 4, 14), e.GetIdRange())
		assert.Equal(t, idx.NewRange(4, 3, 4, 14), e.GetDocumentRange())
		assert.Equal(t, "IOResult", e.GetFaultName())
		assert.Same(t, fault.Children()[1], e)
	})
}

func TestParse_interface(t *testing.T) {
	module := "x"
	docId := "doc"
	source := `<* docs *>
	interface MyName
	{
		fn String method();
	};`

	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds interface", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		expected := idx.NewInterfaceBuilder("MyName", module, doc.URI).
			Build()

		module := symbols.Get("doc")
		interfac := module.Interfaces["MyName"]
		assert.NotNil(t, interfac)
		assert.Same(t, module.Children()[0], interfac)

		assert.Equal(t, expected.GetName(), interfac.GetName())
	})

	t.Run("reads ranges for interface", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("doc").Interfaces["MyName"]
		assert.Equal(t, idx.NewRange(1, 1, 4, 2), found.GetDocumentRange(), "Wrong document rage")
		assert.Equal(t, idx.NewRange(1, 11, 1, 17), found.GetIdRange(), "Wrong identifier range")
	})

	t.Run("finds doc comment", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("doc").Interfaces["MyName"]
		assert.Equal(t, "docs", found.GetDocComment().GetBody())
	})

	t.Run("finds defined methods in interface", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		module := symbols.Get("doc")
		_interface := module.Interfaces["MyName"]
		m := _interface.GetMethod("method")
		assert.Equal(t, "method", m.GetName())
		assert.Equal(t, "String", m.GetReturnType().GetName())
		assert.Equal(t, idx.NewRange(3, 12, 3, 18), m.GetIdRange())
		assert.Equal(t, module.Children()[0], _interface)
	})
}

func TestExtractSymbols_finds_definition(t *testing.T) {
	source := `module mod;
	<* docs *>
	def Kilo = int;
	def KiloPtr = Kilo*;
	def MyFunction = fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*);
	def MyMap = HashMap(<String, Feature>);
	def Camera = raylib::Camera;

	int global_var = 10;
	const int MY_CONST = 5;
	macro @ad(; @body) { @body(); }
	fn void a() {}

	def func = a(<String>);
	def aliased_global = global_var;
	def CONST_ALIAS = MY_CONST;
	def @macro_alias = @a;
	`
	// TODO: Missing def different definition examples. See parser.nodeToDef
	mod := "mod"
	doc := document.NewDocument("x", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)
	module := symbols.Get(mod)

	expectedDefKilo := idx.NewDefBuilder("Kilo", mod, doc.URI).
		WithResolvesToType(
			idx.NewType(true, "int", 0, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(2, 5, 2, 9).
		WithDocumentRange(2, 1, 2, 16).
		Build()
	expectedDefKilo.SetDocComment(cast.ToPtr(idx.NewDocComment("docs")))
	assert.Equal(t, expectedDefKilo, module.Defs["Kilo"])
	assert.Same(t, module.Children()[0], module.Defs["Kilo"])

	expectedDefKiloPtr := idx.NewDefBuilder("KiloPtr", mod, doc.URI).
		WithResolvesToType(
			idx.NewType(false, "Kilo", 1, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(3, 5, 3, 12).
		WithDocumentRange(3, 1, 3, 21).
		Build()
	assert.Equal(t, expectedDefKiloPtr, module.Defs["KiloPtr"])
	assert.Same(t, module.Children()[1], module.Defs["KiloPtr"])

	expectedDefFunction := idx.NewDefBuilder("MyFunction", mod, doc.URI).
		WithResolvesTo("fn void (Allocator*, JSONRPCRequest*, JSONRPCResponse*)").
		WithIdentifierRange(4, 5, 4, 15).
		WithDocumentRange(4, 1, 4, 74).
		Build()

	assert.Equal(t, expectedDefFunction, module.Defs["MyFunction"])
	assert.Same(t, module.Children()[2], module.Defs["MyFunction"])

	expectedDefTypeWithGenerics := idx.NewDefBuilder("MyMap", mod, doc.URI).
		WithResolvesToType(
			idx.NewTypeWithGeneric(
				false,
				false,
				"HashMap",
				0,
				[]idx.Type{
					idx.NewType(false, "String", 0, false, false, option.None[int](), "mod"),
					idx.NewType(false, "Feature", 0, false, false, option.None[int](), "mod"),
				}, "mod"),
		).
		WithIdentifierRange(5, 5, 5, 10).
		WithDocumentRange(5, 1, 5, 40).
		Build()

	assert.Equal(t, expectedDefTypeWithGenerics, module.Defs["MyMap"])
	assert.Same(t, module.Children()[3], module.Defs["MyMap"])

	expectedDefTypeWithModulePath := idx.NewDefBuilder("Camera", mod, doc.URI).
		WithResolvesToType(
			idx.NewType(false, "Camera", 0, false, false, option.None[int](), "raylib"),
		).
		WithIdentifierRange(6, 5, 6, 11).
		WithDocumentRange(6, 1, 6, 29).
		Build()

	assert.Equal(t, expectedDefTypeWithModulePath, module.Defs["Camera"])
	assert.Same(t, module.Children()[4], module.Defs["Camera"])

	expectedDefTypeAliasingToFunc := idx.NewDefBuilder("func", mod, doc.URI).
		WithResolvesTo("a(<String>)").
		WithIdentifierRange(13, 5, 13, 9).
		WithDocumentRange(13, 1, 13, 24).
		Build()

	assert.Equal(t, expectedDefTypeAliasingToFunc, module.Defs["func"])
	assert.Same(t, module.Children()[7], module.Defs["func"])

	expectedDefTypeAliasingToGlobalVar := idx.NewDefBuilder("aliased_global", mod, doc.URI).
		WithResolvesTo("global_var").
		WithIdentifierRange(14, 5, 14, 19).
		WithDocumentRange(14, 1, 14, 33).
		Build()

	assert.Equal(t, expectedDefTypeAliasingToGlobalVar, module.Defs["aliased_global"])
	assert.Same(t, module.Children()[8], module.Defs["aliased_global"])

	expectedDefTypeAliasingToConst := idx.NewDefBuilder("CONST_ALIAS", mod, doc.URI).
		WithResolvesTo("MY_CONST").
		WithIdentifierRange(15, 5, 15, 16).
		WithDocumentRange(15, 1, 15, 28).
		Build()

	assert.Equal(t, expectedDefTypeAliasingToConst, module.Defs["CONST_ALIAS"])
	assert.Same(t, module.Children()[9], module.Defs["CONST_ALIAS"])

	expectedDefTypeAliasingToMacro := idx.NewDefBuilder("@macro_alias", mod, doc.URI).
		WithResolvesTo("@a").
		WithIdentifierRange(16, 5, 16, 17).
		WithDocumentRange(16, 1, 16, 23).
		Build()

	assert.Equal(t, expectedDefTypeAliasingToMacro, module.Defs["@macro_alias"])
	assert.Same(t, module.Children()[10], module.Defs["@macro_alias"])
}

func TestExtractSymbols_finds_distinct(t *testing.T) {
	source := `module mod;
	<* docs *>
	distinct Kilo = int;
	distinct KiloPtr = Kilo*;
	distinct MyMap = HashMap(<String, Feature>);
	distinct Camera = raylib::Camera;
	`
	mod := "mod"
	doc := document.NewDocument("x", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)
	module := symbols.Get(mod)

	expectedDistinctKilo := idx.NewDistinctBuilder("Kilo", mod, doc.URI).
		WithInline(false).
		WithBaseType(
			idx.NewType(true, "int", 0, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(2, 10, 2, 14).
		WithDocumentRange(2, 1, 2, 21).
		Build()
	expectedDistinctKilo.SetDocComment(cast.ToPtr(idx.NewDocComment("docs")))
	assert.Equal(t, expectedDistinctKilo, module.Distincts["Kilo"])
	assert.Same(t, module.Children()[0], module.Distincts["Kilo"])

	expectedDistinctKiloPtr := idx.NewDistinctBuilder("KiloPtr", mod, doc.URI).
		WithInline(false).
		WithBaseType(
			idx.NewType(false, "Kilo", 1, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(3, 10, 3, 17).
		WithDocumentRange(3, 1, 3, 26).
		Build()
	assert.Equal(t, expectedDistinctKiloPtr, module.Distincts["KiloPtr"])
	assert.Same(t, module.Children()[1], module.Distincts["KiloPtr"])

	expectedDistinctTypeWithGenerics := idx.NewDistinctBuilder("MyMap", mod, doc.URI).
		WithInline(false).
		WithBaseType(
			idx.NewTypeWithGeneric(
				false,
				false,
				"HashMap",
				0,
				[]idx.Type{
					idx.NewType(false, "String", 0, false, false, option.None[int](), "mod"),
					idx.NewType(false, "Feature", 0, false, false, option.None[int](), "mod"),
				}, "mod"),
		).
		WithIdentifierRange(4, 10, 4, 15).
		WithDocumentRange(4, 1, 4, 45).
		Build()

	assert.Equal(t, expectedDistinctTypeWithGenerics, module.Distincts["MyMap"])
	assert.Same(t, module.Children()[2], module.Distincts["MyMap"])

	expectedDistinctTypeWithModulePath := idx.NewDistinctBuilder("Camera", mod, doc.URI).
		WithInline(false).
		WithBaseType(
			idx.NewType(false, "Camera", 0, false, false, option.None[int](), "raylib"),
		).
		WithIdentifierRange(5, 10, 5, 16).
		WithDocumentRange(5, 1, 5, 34).
		Build()

	assert.Equal(t, expectedDistinctTypeWithModulePath, module.Distincts["Camera"])
	assert.Same(t, module.Children()[3], module.Distincts["Camera"])
}

func TestExtractSymbols_finds_distinct_with_inline(t *testing.T) {
	source := `module mod;
	<* docs *>
	distinct Kilo = inline int;
	distinct KiloPtr = inline Kilo*;
	distinct MyMap = inline HashMap(<String, Feature>);
	distinct Camera = inline raylib::Camera;
	`
	mod := "mod"
	doc := document.NewDocument("x", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)
	module := symbols.Get(mod)

	expectedDistinctKilo := idx.NewDistinctBuilder("Kilo", mod, doc.URI).
		WithInline(true).
		WithBaseType(
			idx.NewType(true, "int", 0, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(2, 10, 2, 14).
		WithDocumentRange(2, 1, 2, 28).
		Build()
	expectedDistinctKilo.SetDocComment(cast.ToPtr(idx.NewDocComment("docs")))
	assert.Equal(t, expectedDistinctKilo, module.Distincts["Kilo"])
	assert.Same(t, module.Children()[0], module.Distincts["Kilo"])

	expectedDistinctKiloPtr := idx.NewDistinctBuilder("KiloPtr", mod, doc.URI).
		WithInline(true).
		WithBaseType(
			idx.NewType(false, "Kilo", 1, false, false, option.None[int](), "mod"),
		).
		WithIdentifierRange(3, 10, 3, 17).
		WithDocumentRange(3, 1, 3, 33).
		Build()
	assert.Equal(t, expectedDistinctKiloPtr, module.Distincts["KiloPtr"])
	assert.Same(t, module.Children()[1], module.Distincts["KiloPtr"])

	expectedDistinctTypeWithGenerics := idx.NewDistinctBuilder("MyMap", mod, doc.URI).
		WithInline(true).
		WithBaseType(
			idx.NewTypeWithGeneric(
				false,
				false,
				"HashMap",
				0,
				[]idx.Type{
					idx.NewType(false, "String", 0, false, false, option.None[int](), "mod"),
					idx.NewType(false, "Feature", 0, false, false, option.None[int](), "mod"),
				}, "mod"),
		).
		WithIdentifierRange(4, 10, 4, 15).
		WithDocumentRange(4, 1, 4, 52).
		Build()

	assert.Equal(t, expectedDistinctTypeWithGenerics, module.Distincts["MyMap"])
	assert.Same(t, module.Children()[2], module.Distincts["MyMap"])

	expectedDistinctTypeWithModulePath := idx.NewDistinctBuilder("Camera", mod, doc.URI).
		WithInline(true).
		WithBaseType(
			idx.NewType(false, "Camera", 0, false, false, option.None[int](), "raylib"),
		).
		WithIdentifierRange(5, 10, 5, 16).
		WithDocumentRange(5, 1, 5, 41).
		Build()

	assert.Equal(t, expectedDistinctTypeWithModulePath, module.Distincts["Camera"])
	assert.Same(t, module.Children()[3], module.Distincts["Camera"])
}

func TestExtractSymbols_find_macro(t *testing.T) {
	/*
		sourceCode := `
		macro void log(LogLevel $level, String format, args...) {
			if (log_level != OFF && $level <= log_level) {
				io::fprintf(&log_file, "[%s] ", $level)!!;
				io::fprintfn(&log_file, format, ...args)!!;
			}
		}`*/
	source := `
	<* docs *>
	macro m(x) {
    	return x + 2;
	}`

	doc := document.NewDocument("docId", source)
	parser := createParser()
	symbols, _ := parser.ParseSymbols(&doc)

	module := symbols.Get("docid")
	fn := module.GetChildrenFunctionByName("m")
	assert.True(t, fn.IsSome())
	assert.Equal(t, "macro m(x)", fn.Get().GetHoverInfo())
	assert.Equal(t, "m", fn.Get().GetName())
	assert.Equal(t, "", fn.Get().GetReturnType().String())
	assert.Equal(t, "x", fn.Get().Variables["x"].GetName())
	assert.Equal(t, "", fn.Get().Variables["x"].GetType().String())
	assert.Equal(t, "docs", fn.Get().GetDocComment().GetBody())
	assert.Same(t, module.NestedScopes()[0], fn.Get())
}

func TestExtractSymbols_find_macro_with_return_type(t *testing.T) {
	/*
		sourceCode := `
		macro void log(LogLevel $level, String format, args...) {
			if (log_level != OFF && $level <= log_level) {
				io::fprintf(&log_file, "[%s] ", $level)!!;
				io::fprintfn(&log_file, format, ...args)!!;
			}
		}`*/
	source := `
	<* docs *>
	macro int m(int x) {
    	return x + 2;
	}`

	doc := document.NewDocument("docId", source)
	parser := createParser()
	symbols, _ := parser.ParseSymbols(&doc)

	module := symbols.Get("docid")
	fn := module.GetChildrenFunctionByName("m")
	assert.True(t, fn.IsSome())
	assert.Equal(t, "macro int m(int x)", fn.Get().GetHoverInfo())
	assert.Equal(t, "m", fn.Get().GetName())
	assert.Equal(t, "int", fn.Get().GetReturnType().String())
	assert.Equal(t, "x", fn.Get().Variables["x"].GetName())
	assert.Equal(t, "int", fn.Get().Variables["x"].GetType().String())
	assert.Equal(t, "docs", fn.Get().GetDocComment().GetBody())
	assert.Same(t, module.NestedScopes()[0], fn.Get())
}

func TestExtractSymbols_handle_invalid_macro_signature(t *testing.T) {
	source := `
	<* docs *>
	macro fn void scary() {

	}`

	doc := document.NewDocument("docId", source)
	parser := createParser()
	symbols, _ := parser.ParseSymbols(&doc)

	module := symbols.Get("docid")
	assert.Empty(t, module.ChildrenFunctions)
	assert.True(t, cast.ToPtr(module.GetChildrenFunctionByName("scary")).IsNone())
	assert.True(t, cast.ToPtr(module.GetChildrenFunctionByName("void")).IsNone())
	assert.True(t, cast.ToPtr(module.GetChildrenFunctionByName("fn")).IsNone())
}

func TestExtractSymbols_find_module(t *testing.T) {
	t.Run("finds anonymous module", func(t *testing.T) {
		source := `int value = 1;`

		doc := document.NewDocument("file name.c3", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)
		fn := symbols.Get("file_name")

		assert.Equal(t, "file_name", fn.GetModuleString(), "Function module is wrong")
	})

	t.Run("finds single module in single file", func(t *testing.T) {
		source := `
	<* docs *>
	module foo;
	int value = 1;
	`

		doc := document.NewDocument("docId", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		module := symbols.Get("foo")
		assert.Equal(t, "foo", module.GetModuleString(), "module name is wrong")
		assert.Equal(t, "docs", module.GetDocComment().GetBody(), "module doc comment is wrong")
	})

	t.Run("finds different modules defined in single file", func(t *testing.T) {
		source := `
	<* docs foo *>
	module foo;
	int value = 1;

	<* docs foo2 *>
	module foo2;
	int value = 2;`

		doc := document.NewDocument("docid", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		module := symbols.Get("foo")
		assert.Equal(t, "foo", module.GetModuleString(), "module name is wrong")
		assert.Equal(t, "foo", module.GetName(), "module name is wrong")
		assert.Equal(t, "docs foo", module.GetDocComment().GetBody(), "module doc comment is wrong")
		assert.Equal(t, idx.NewRange(2, 1, 3, 15), module.GetDocumentRange(), "Wrong range for foo module")

		module = symbols.Get("foo2")
		assert.Equal(t, "foo2", module.GetModuleString(), "module name is wrong")
		assert.Equal(t, "foo2", module.GetName(), "module name is wrong")
		assert.Equal(t, "docs foo2", module.GetDocComment().GetBody(), "module doc comment is wrong")
		assert.Equal(t, idx.NewRange(6, 1, 7, 15), module.GetDocumentRange(), "Wrong range for foo2 module")
	})

	t.Run("finds named module with attributes", func(t *testing.T) {
		source := `module std::core @if(env::DARWIN) @private;`

		doc := document.NewDocument("filename.c3", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)
		module := symbols.Get("std::core")

		assert.Equal(t, "std::core", module.GetName(), "module name is wrong")
		assert.Equal(t, true, module.IsPrivate(), "module should be marked as private")
	})
}

func TestExtractSymbols_find_imports(t *testing.T) {
	source := `
	module foo;
	import some, other, foo::bar::final;
	import another;
	import another2;
	int value = 1;
	`

	doc := document.NewDocument("docid", source)
	parser := createParser()
	symbols, _ := parser.ParseSymbols(&doc)

	assert.Equal(t, []string{"some", "other", "foo::bar::final", "another", "another2"}, symbols.Get("foo").Imports)
}

func TestExtractSymbols_module_with_generics(t *testing.T) {

	//module std::atomic::types(<Type>);
	source := `module foo_test(<Type1, Type2>);
		struct Foo
		{
			Type1 a;
		}
		fn Type2 test(Type2 b, Foo *foo)
		{
			return foo.a + b;
		}

		module foo::another::deep(<Type>);
		int bar = 0;`

	doc := document.NewDocument("docid", source)
	parser := createParser()
	symbols, _ := parser.ParseSymbols(&doc)

	module := symbols.Get("foo_test")
	assert.Equal(t, "foo_test", module.GetName())

	// Generic parameter was found
	generic, ok := module.GenericParameters["Type1"]
	assert.True(t, ok)
	assert.Equal(t, "Type1", generic.GetName())
	assert.Equal(t, idx.NewRange(0, 17, 0, 22), generic.GetIdRange())
	assert.Equal(t, idx.NewRange(0, 17, 0, 22), generic.GetDocumentRange())

	// Generic parameter was found
	generic, ok = module.GenericParameters["Type2"]
	assert.True(t, ok)
	assert.Equal(t, "Type2", generic.GetName())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), generic.GetIdRange())
	assert.Equal(t, idx.NewRange(0, 24, 0, 29), generic.GetDocumentRange())

	// Usages of generic parameters are flagged as such
	strukt := module.Structs["Foo"]
	sms := strukt.GetMembers()
	assert.Equal(t, true, sms[0].GetType().IsGenericArgument())

	module = symbols.Get("foo::another::deep")
	assert.Equal(t, "foo::another::deep", module.GetName())
}
