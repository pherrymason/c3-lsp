package parser

import (
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	idx "github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

func assertVariableFound(t *testing.T, name string, symbols idx.Function) {
	_, ok := symbols.Variables[name]
	assert.True(t, ok)
}

func TestExtractSymbols_find_variables(t *testing.T) {
	source := `
	<* docs *>
	int value = 1;
	char* character;
	<* multidocs *>
	int foo, foo2;
	char[] message;
	char[4] message2;
	fn void test() { int value = 1; }
	fn void test2() { int value, value2; }
	`
	docId := "x"
	doc := document.NewDocument(docId, source)
	parser := createParser()

	t.Run("finds global basic type variable declarations", func(t *testing.T) {
		symbols, pendingToResolve := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["value"]
		assert.Equal(t, "value", found.GetName(), "Variable name")
		assert.Equal(t, "int", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "int value = 1"), found.GetDocumentRange())
		assert.Equal(t, findRange(source, "value"), found.GetIdRange())
		assert.Equal(t, "docs", found.GetDocComment().GetBody(), "Variable docs")
		assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(docId)), "Basic types should not be registered as pending to resolve.")
	})

	t.Run("finds global pointer variable declarations", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["character"]
		assert.Equal(t, "character", found.GetName(), "Variable name")
		assert.Equal(t, "char*", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "char* character"), found.GetDocumentRange())
		assert.Equal(t, findRange(source, "character"), found.GetIdRange())
	})

	t.Run("finds global variable collection declarations", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["message"]
		assert.Equal(t, "message", found.GetName(), "Variable name")
		assert.Equal(t, "char[]", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "char[] message"), found.GetDocumentRange())
		assert.Equal(t, findRange(source, "message"), found.GetIdRange())
	})

	t.Run("finds global variable static collection declarations", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		found := symbols.Get("x").Variables["message2"]
		assert.Equal(t, "message2", found.GetName(), "Variable name")
		assert.Equal(t, "char[4]", found.GetType().String(), "Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "char[4] message2"), found.GetDocumentRange())
		assert.Equal(t, findRange(source, "message2"), found.GetIdRange())
	})

	t.Run("finds multiple global variables declared in single sentence", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		symbol := symbols.Get("x")
		if !assert.NotNil(t, symbol, "Symbol x not found") {
			return
		}
		found := symbol.Variables["foo"]
		if !assert.NotNil(t, found, "Variable foo not found") {
			return
		}
		assert.Equal(t, "foo", found.GetName(), "First Variable name")
		assert.Equal(t, "int", found.GetType().String(), "First Variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "foo"), found.GetIdRange(), "First variable identifier range")
		assert.Equal(t, findRange(source, "int foo, foo2"), found.GetDocumentRange(), "First variable declaration range")
		assert.Equal(t, "multidocs", found.GetDocComment().GetBody())

		found = symbols.Get("x").Variables["foo2"]
		assert.Equal(t, "foo2", found.GetName(), "Second variable name")
		assert.Equal(t, "int", found.GetType().String(), "Second variable type")
		assert.Equal(t, true, found.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, findRange(source, "foo2"), found.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, findRange(source, "int foo, foo2"), found.GetDocumentRange(), "Second variable declaration range")
		assert.Equal(t, "multidocs", found.GetDocComment().GetBody())
	})

	t.Run("finds variables declared inside function", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["value"]
		if !assert.NotNil(t, variable, "Couldnt find variable 'value' inside function") {
			return
		}
		assert.Equal(t, "value", variable.GetName(), "variable name")
		assert.Equal(t, "int", variable.GetType().String(), "variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(8, 22, 8, 27), variable.GetIdRange(), "variable identifier range")
		assert.Equal(t, idx.NewRange(8, 18, 8, 31), variable.GetDocumentRange(), "variable declaration range")
	})

	t.Run("finds multiple local variables declared in single sentence", func(t *testing.T) {
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test2")
		assert.True(t, function.IsSome())
		variable := function.Get().Variables["value"]
		assert.Equal(t, "value", variable.GetName(), "First Variable name")
		assert.Equal(t, "int", variable.GetType().String(), "First Variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(9, 23, 9, 28), variable.GetIdRange(), "First variable identifier range")
		assert.Equal(t, idx.NewRange(9, 19, 9, 36), variable.GetDocumentRange(), "First variable declaration range")

		variable = function.Get().Variables["value2"]
		assert.Equal(t, "value2", variable.GetName(), "Second variable name")
		assert.Equal(t, "int", variable.GetType().String(), "Second variable type")
		assert.Equal(t, true, variable.GetType().IsBaseTypeLanguage(), "Variable Type should be base type")
		assert.Equal(t, idx.NewRange(9, 30, 9, 36), variable.GetIdRange(), "Second variable identifier range")
		assert.Equal(t, idx.NewRange(9, 19, 9, 36), variable.GetDocumentRange(), "Second variable declaration range")
	})

	t.Run("finds local variable declarations inside nested blocks", func(t *testing.T) {
		source := `
		struct Value {
			int inner;
		}

		fn bool Value.to_bool(self) {
			return true;
		}

		fn void parse() {
			if (true) {
				Value val = {};
				val.to_bool();
			}
		}
		`

		doc := document.NewDocument("x", source)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("parse")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["val"]
		if !assert.NotNil(t, variable, "Could not find variable 'val' inside nested block") {
			return
		}

		assert.Equal(t, "val", variable.GetName())
		assert.Equal(t, "Value", variable.GetType().String())
	})

	t.Run("finds try-unwrap bound variables inside function", func(t *testing.T) {
		source := `
		fn uint parse_clients(String[] args, uint default_value = 32)
		{
			if (args.len < 2) return default_value;
			if (try try_n_bind = args[1].to_integer(uint, 10))
			{
				if (try_n_bind > 0) return try_n_bind;
			}
			return default_value;
		}
		`

		doc := document.NewDocument("x", source)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("parse_clients")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["try_n_bind"]
		if !assert.NotNil(t, variable, "Could not find variable 'try_n_bind' bound by try unwrap") {
			return
		}

		assert.Equal(t, "try_n_bind", variable.GetName())
		assert.Equal(t, "uint", variable.GetType().String())
		assert.Equal(t, findRange(source, "try_n_bind"), variable.GetIdRange())
	})

	t.Run("finds catch-unwrap bound variables as fault type", func(t *testing.T) {
		source := `
		fn int parse_port(String[] args, int default_value = 19080)
		{
			if (catch catch_reason = args[1].to_integer(uint, 10))
			{
				return default_value;
			}
			return default_value;
		}
		`

		doc := document.NewDocument("x", source)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("parse_port")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["catch_reason"]
		if !assert.NotNil(t, variable, "Could not find variable 'catch_reason' bound by catch unwrap") {
			return
		}

		assert.Equal(t, "catch_reason", variable.GetName())
		assert.Equal(t, "fault", variable.GetType().String())
		assert.Equal(t, findRange(source, "catch_reason"), variable.GetIdRange())
	})

	t.Run("finds catch-unwrap bound variables with explicit type", func(t *testing.T) {
		source := `
		fn int parse_port(String[] args, int default_value = 19080)
		{
			if (catch int catch_reason = args[1].to_integer(int, 10))
			{
				return default_value;
			}
			return catch_reason;
		}
		`

		doc := document.NewDocument("x", source)
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("parse_port")
		assert.True(t, function.IsSome())

		variable := function.Get().Variables["catch_reason"]
		if !assert.NotNil(t, variable, "Could not find variable 'catch_reason' bound by catch unwrap") {
			return
		}

		assert.Equal(t, "catch_reason", variable.GetName())
		assert.Equal(t, "int", variable.GetType().String())
		assert.Equal(t, findRange(source, "catch_reason"), variable.GetIdRange())
	})
}

func TestExtractSymbols_parses_fully_qualified_type_paths(t *testing.T) {
	source := `
	bindgen::bg::BGOptions opts = {};
	std::io::File file = std::io::stdout().file;
	`

	doc := document.NewDocument("docId", source)
	parser := createParser()
	parsed, _ := parser.ParseSymbols(&doc)
	module := parsed.Get("docid")
	if !assert.NotNil(t, module) {
		return
	}

	opts := module.Variables["opts"]
	if !assert.NotNil(t, opts) {
		return
	}
	assert.Equal(t, "BGOptions", opts.GetType().GetName())
	assert.Equal(t, "bindgen::bg", opts.GetType().GetModule())

	file := module.Variables["file"]
	if !assert.NotNil(t, file) {
		return
	}
	assert.Equal(t, "File", file.GetType().GetName())
	assert.Equal(t, "std::io", file.GetType().GetModule())
}

func TestExtractSymbols_find_variables_in_nested_blocks(t *testing.T) {
	t.Run("finds variable declared inside while block", func(t *testing.T) {
		source := `fn void test() {
	int top_level = 1;
	while (top_level > 0) {
		int inside_while = 2;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "top_level", *function.Get())
		assertVariableFound(t, "inside_while", *function.Get())
	})

	t.Run("finds variable declared inside for block", func(t *testing.T) {
		source := `fn void test() {
	for (int i = 0; i < 10; i++) {
		int inside_for = 3;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_for", *function.Get())
	})

	t.Run("finds variable declared inside if block", func(t *testing.T) {
		source := `fn void test() {
	if (true) {
		int inside_if = 4;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_if", *function.Get())
	})

	t.Run("finds variable declared inside if-else blocks", func(t *testing.T) {
		source := `fn void test() {
	if (true) {
		int in_if = 1;
	} else {
		int in_else = 2;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "in_if", *function.Get())
		assertVariableFound(t, "in_else", *function.Get())
	})

	t.Run("finds variable declared inside nested blocks (2 levels)", func(t *testing.T) {
		source := `fn void test() {
	while (true) {
		if (true) {
			int deeply_nested = 5;
		}
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "deeply_nested", *function.Get())
	})

	t.Run("finds variable declared inside nested blocks (3 levels)", func(t *testing.T) {
		source := `fn void test() {
	while (true) {
		for (int i = 0; i < 10; i++) {
			if (true) {
				int very_deep = 6;
			}
		}
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "very_deep", *function.Get())
	})

	t.Run("finds variable declared inside do-while block", func(t *testing.T) {
		source := `fn void test() {
	do {
		int inside_do = 7;
	} while (true);
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_do", *function.Get())
	})

	t.Run("finds variable declared inside defer block", func(t *testing.T) {
		source := `fn void test() {
	defer {
		int inside_defer = 8;
	};
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_defer", *function.Get())
	})

	t.Run("finds variable declared inside plain nested block", func(t *testing.T) {
		source := `fn void test() {
	{
		int inside_block = 9;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_block", *function.Get())
	})

	t.Run("finds variable declared inside switch case", func(t *testing.T) {
		source := `fn void test() {
	int x = 1;
	switch (x) {
		case 1:
			int inside_case = 10;
		default:
			int inside_default = 11;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "inside_case", *function.Get())
		assertVariableFound(t, "inside_default", *function.Get())
	})

	t.Run("finds foreach iterator variable", func(t *testing.T) {
		source := `struct Route { int id; }
fn void test() {
	Route[] routes;
	foreach (route : routes) {
		int inside_foreach = 12;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		assertVariableFound(t, "route", *function.Get())
		assertVariableFound(t, "inside_foreach", *function.Get())
	})

	t.Run("infers foreach iterator type from receiver collection member", func(t *testing.T) {
		source := `struct Route { String path; }
struct Router { List{Route} routes; }

fn void Router.match(&router)
{
	foreach (route : router.routes)
	{
		String p = route.path;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("Router.match")
		assert.True(t, function.IsSome())

		route := function.Get().Variables["route"]
		assert.Equal(t, "Route", route.GetType().String())
	})

	t.Run("infers foreach index and element types", func(t *testing.T) {
		source := `struct Route { String path; }
fn void test() {
	Route[] routes;
	foreach (i, route : routes) {
		String p = route.path;
		usz n = i;
	}
}`
		doc := document.NewDocument("x", source)
		parser := createParser()
		symbols, _ := parser.ParseSymbols(&doc)

		function := symbols.Get("x").GetChildrenFunctionByName("test")
		assert.True(t, function.IsSome())

		indexVar := function.Get().Variables["i"]
		assert.Equal(t, "usz", indexVar.GetType().String())

		route := function.Get().Variables["route"]
		assert.Equal(t, "Route", route.GetType().String())
	})
}

func TestExtractSymbols_find_constants(t *testing.T) {

	source := `<* docs *>
	const int A_VALUE = 12;`

	doc := document.NewDocument("docId", source)
	parser := createParser()

	symbols, _ := parser.ParseSymbols(&doc)

	found := symbols.Get("docid").Variables["A_VALUE"]
	assert.Equal(t, "A_VALUE", found.GetName(), "Variable name")
	assert.Equal(t, "int", found.GetType().String(), "Variable type")
	assert.True(t, found.IsConstant())
	assert.Equal(t, "12", found.GetConstantValue(), "Constant value")
	assert.Equal(t, idx.NewRange(1, 1, 1, 23), found.GetDocumentRange())
	assert.Equal(t, idx.NewRange(1, 11, 1, 18), found.GetIdRange())
	assert.Equal(t, "docs", found.GetDocComment().GetBody(), "Variable doc comment")
}

func TestExtractSymbols_find_variables_flag_pending_to_resolve(t *testing.T) {
	t.Run("resolves basic type declaration should not flag type as pending to be resolved", func(t *testing.T) {
		source := `int value = 1;`
		docId := "x"
		doc := document.NewDocument(docId, source)
		parser := createParser()
		symbols, pendingToResolve := parser.ParseSymbols(&doc)

		found := symbols.Get(docId).Variables["value"]
		assert.NotNil(t, found)

		assert.Equal(t, 0, len(pendingToResolve.GetTypesByModule(docId)), "Basic types should not be registered as pending to resolve.")
	})
}
