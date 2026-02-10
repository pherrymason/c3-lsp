package search_v2

import (
	"testing"

	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/stretchr/testify/assert"
)

// Helper to resolve access path using FindSymbolDeclarationInWorkspace
func ResolveAccessPath(t *testing.T, body string) option.Option[symbols.Indexable] {
	state := NewTestState()
	search := NewSearchV2WithoutLog()

	cursorlessBody, position := parseBodyWithCursor(body)
	state.RegisterDoc("app.c3", cursorlessBody)

	doc := state.GetDoc("app.c3")

	// Use the main entry point which handles both simple symbols and access paths
	result := search.FindSymbolDeclarationInWorkspace(doc.URI, position, &state.State)

	return result
}

func TestResolveAccessPath_BasicStructMembers(t *testing.T) {
	t.Run("Struct member access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Point {
				int x;
				int y;
			}
			fn void test() {
				Point p;
				p.x|||;
			}
		`)

		assert.True(t, result.IsSome(), "Should find struct member")
		if result.IsSome() {
			member := result.Get().(*symbols.StructMember)
			assert.Equal(t, "x", member.GetName())
			assert.Equal(t, "int", member.GetType().String())
		}
	})

	t.Run("Nested struct member access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Inner {
				int value;
			}
			struct Outer {
				Inner inner;
			}
			fn void test() {
				Outer o;
				o.inner.val|||ue;
			}
		`)

		assert.True(t, result.IsSome(), "Should find nested struct member")
		member := result.Get().(*symbols.StructMember)
		assert.Equal(t, "value", member.GetName())
	})

	t.Run("Three-level nested access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Level3 {
				int data;
			}
			struct Level2 {
				Level3 l3;
			}
			struct Level1 {
				Level2 l2;
			}
			fn void test() {
				Level1 obj;
				obj.l2.l3.da|||ta;
			}
		`)

		assert.True(t, result.IsSome(), "Should find deeply nested member")
		member := result.Get().(*symbols.StructMember)
		assert.Equal(t, "data", member.GetName())
	})
}

func TestResolveAccessPath_EnumAccess(t *testing.T) {
	t.Run("Enum variant access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			enum Color {
				RED,
				GREEN,
				BLUE
			}
			fn void test() {
				Color c = Color.RE|||D;
			}
		`)

		assert.True(t, result.IsSome(), "Should find enum variant")
		enumerator := result.Get().(*symbols.Enumerator)
		assert.Equal(t, "RED", enumerator.GetName())
	})

	t.Run("Enum with associated values", func(t *testing.T) {
		t.Skip("TODO: Enum associated values not yet tested in original search - needs C3 parser investigation")
		result := ResolveAccessPath(t, `
			module app;
			enum Result {
				OK(int value),
				ERROR
			}
			fn void test() {
				Result r = Result.OK;
				r.val|||ue;
			}
		`)

		assert.True(t, result.IsSome(), "Should find associated value")
		if result.IsSome() {
			assocValue := result.Get()
			assert.Equal(t, "value", assocValue.GetName())
		}
	})
}

func TestResolveAccessPath_DistinctTypes(t *testing.T) {
	t.Run("Inline distinct member access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Point { int x; int y; }
			typedef DistPoint = inline Point;
			fn void test() {
				DistPoint p;
				p.x|||;
			}
		`)

		assert.True(t, result.IsSome(), "Should find member through inline distinct")
		if result.IsSome() {
			member := result.Get().(*symbols.StructMember)
			assert.Equal(t, "x", member.GetName())
		}
	})

	t.Run("Non-inline distinct blocks member access", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Point { int x; int y; }
			typedef DistPoint = Point;
			fn void test() {
				DistPoint p;
				p.x|||;
			}
		`)

		// Non-inline distinct should block member access
		assert.False(t, result.IsSome(), "Should NOT find member through non-inline distinct")
	})
}

func TestResolveAccessPath_Methods(t *testing.T) {
	t.Run("Struct method", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Point {
				int x;
				int y;
			}
			fn void Point.reset(&self) {
				self.x = 0;
			}
			fn void test() {
				Point p;
				p.res|||et();
			}
		`)

		assert.True(t, result.IsSome(), "Should find struct method")
		if result.IsSome() {
			method := result.Get().(*symbols.Function)
			assert.Equal(t, "reset", method.GetMethodName())
		}
	})

	t.Run("Enum method", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			enum Color { RED, GREEN, BLUE }
			fn bool Color.is_primary(self) {
				return self == RED || self == BLUE || self == GREEN;
			}
			fn void test() {
				Color c = Color.RED;
				c.is_prim|||ary();
			}
		`)

		assert.True(t, result.IsSome(), "Should find enum method")
		if result.IsSome() {
			method := result.Get().(*symbols.Function)
			assert.Equal(t, "is_primary", method.GetMethodName())
		}
	})
}

func TestResolveAccessPath_ComplexChains(t *testing.T) {
	t.Run("Variable to type to member", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Config {
				int timeout;
			}
			struct App {
				Config config;
			}
			fn void test() {
				App app;
				app.config.time|||out;
			}
		`)

		assert.True(t, result.IsSome(), "Should find member through chain")
		member := result.Get().(*symbols.StructMember)
		assert.Equal(t, "timeout", member.GetName())
	})

	t.Run("Function return to member", func(t *testing.T) {
		result := ResolveAccessPath(t, `
			module app;
			struct Point { int x; }
			fn Point getPoint() {
				Point p;
				return p;
			}
			fn void test() {
				getPoint().x|||;
			}
		`)

		assert.True(t, result.IsSome(), "Should find member on function return")
		member := result.Get().(*symbols.StructMember)
		assert.Equal(t, "x", member.GetName())
	})
}

func TestResolveAccessPath_FaultTypes(t *testing.T) {
	t.Run("Fault constant access", func(t *testing.T) {
		t.Skip("TODO: Fault access path not tested in original search - C3 uses 'faultdef' syntax, needs investigation")
		result := ResolveAccessPath(t, `
			module app;
			fault MyError {
				ERROR_ONE,
				ERROR_TWO
			}
			fn void test() {
				MyError err = MyError.ERROR_ON|||E;
			}
		`)

		assert.True(t, result.IsSome(), "Should find fault constant")
		if result.IsSome() {
			constant := result.Get().(*symbols.FaultConstant)
			assert.Equal(t, "ERROR_ONE", constant.GetName())
		}
	})
}

func TestResolveAccessPath_EdgeCases(t *testing.T) {
	t.Run("Empty access path returns empty", func(t *testing.T) {
		state := NewTestState()
		search := NewSearchV2WithoutLog()

		// Create search params with empty access path
		searchParams := search_params.NewSearchParamsBuilder().
			WithText("test", symbols.NewRange(0, 0, 0, 0)).
			Build()

		result := search.ResolveAccessPath(searchParams, &state.State)
		assert.False(t, result.IsSome(), "Empty access path should return no result")
	})
}

// Benchmark to measure performance
func BenchmarkResolveAccessPath_DeepNesting(b *testing.B) {
	state := NewTestState()
	search := NewSearchV2WithoutLog()

	body := `
		module app;
		struct Level4 { int data; }
		struct Level3 { Level4 l4; }
		struct Level2 { Level3 l3; }
		struct Level1 { Level2 l2; }
		fn void test() {
			Level1 obj;
			obj.l2.l3.l4.data;
		}
	`
	cursorlessBody, position := parseBodyWithCursor(body)
	state.RegisterDoc("app.c3", cursorlessBody)

	doc := state.GetDoc("app.c3")
	searchParams := search_params.BuildSearchBySymbolUnderCursor(
		&doc,
		*state.State.GetUnitModulesByDoc(doc.URI),
		position,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		search.ResolveAccessPath(searchParams, &state.State)
	}
}
