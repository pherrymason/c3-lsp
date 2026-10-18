package stdlib

import (
	"encoding/json"
	"testing"

	"github.com/pherrymason/c3-lsp/pkg/document"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/commonlog"
)

const cacheRoundTripSource = `module std::sample;
import std::io;

<*
 Prints a value.
 @param x : "The value to print"
*>
macro void printn(x = "") {}

fn int add(int a, int b = 2) { return a + b; }

fn void Point.move(&self, int dx) {}

struct Point
{
	int x;
	int y;
	struct inner { float z; }
}

union Number { int i; float f; }

bitstruct Flags : char
{
	bool a : 0;
	bool b : 1;
}

enum Color : int (String hex)
{
	RED = { "#f00" },
	GREEN = { "#0f0" },
}

constdef Opt : inline int
{
	ALIGNCENTER = 1 << 0,
	ALIGNRIGHT = 1 << 1,
}

faultdef NOT_FOUND, BROKEN;

interface Shape
{
	fn float area();
}

alias PointPtr = Point*;
typedef Meters = inline float;
const int MAX = 10;
int counter;
`

func parseRoundTripSource(t *testing.T) symbols_table.UnitModules {
	t.Helper()
	doc := document.NewDocumentFromString("sample.c3", cacheRoundTripSource)
	parser := p.NewParser(commonlog.MockLogger{})
	modules, _ := parser.ParseSymbols(&doc)
	return modules
}

func childNames(symbols []symbols.Indexable) []string {
	names := make([]string, 0, len(symbols))
	for _, s := range symbols {
		names = append(names, s.GetName())
	}
	return names
}

func TestStdlibCache_round_trip_preserves_symbols(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	logger := commonlog.MockLogger{}

	original := parseRoundTripSource(t)
	require.NoError(t, saveStdlibToCache(logger, "test", "fp", []symbols_table.UnitModules{original}))

	loaded, err := loadStdlibFromCache(logger, "test", "fp")
	require.NoError(t, err)
	require.Len(t, loaded, 1)

	want := original.Get("std::sample")
	got := loaded[0].Get("std::sample")
	require.NotNil(t, got, "module must be registered under its reconstructed name")

	t.Run("serialized data is identical", func(t *testing.T) {
		wantJSON, err := json.Marshal(want)
		require.NoError(t, err)
		gotJSON, err := json.Marshal(got)
		require.NoError(t, err)
		assert.JSONEq(t, string(wantJSON), string(gotJSON))
	})

	t.Run("derived indexes are rebuilt", func(t *testing.T) {
		assert.Equal(t, want.GetModule().GetName(), got.GetModule().GetName())
		assert.Equal(t, want.ChildrenNames(), got.ChildrenNames())
		assert.Equal(t, childNames(want.NestedScopes()), childNames(got.NestedScopes()))
	})

	t.Run("functions keep signature and docs", func(t *testing.T) {
		for i, fn := range want.ChildrenFunctions {
			assert.Equal(t, fn.GetHoverInfo(), got.ChildrenFunctions[i].GetHoverInfo())
			assert.Equal(t, fn.FunctionType(), got.ChildrenFunctions[i].FunctionType())
			assert.Equal(t, childNames(fn.Children()), childNames(got.ChildrenFunctions[i].Children()))
		}
		assert.Equal(t, "Prints a value.", got.ChildrenFunctions[0].GetDocComment().GetBody())
	})

	t.Run("nested symbols are rebuilt", func(t *testing.T) {
		assert.Equal(t, []string{"RED", "GREEN"}, childNames(got.Enums["Color"].Children()))
		assert.Equal(t, []string{"ALIGNCENTER", "ALIGNRIGHT"}, childNames(got.Enums["Opt"].Children()))
		assert.Equal(t, childNames(want.Structs["Point"].Children()), childNames(got.Structs["Point"].Children()))
		assert.True(t, got.Structs["Number"].IsUnion())
		assert.Equal(t, childNames(want.Bitstructs["Flags"].Children()), childNames(got.Bitstructs["Flags"].Children()))
		assert.Equal(t, childNames(want.Interfaces["Shape"].Children()), childNames(got.Interfaces["Shape"].Children()))
		assert.Equal(t, want.Defs["PointPtr"].GetHoverInfo(), got.Defs["PointPtr"].GetHoverInfo())
		assert.Equal(t, want.Distincts["Meters"].GetHoverInfo(), got.Distincts["Meters"].GetHoverInfo())
		assert.Equal(t, want.Variables["MAX"].GetType().String(), got.Variables["MAX"].GetType().String())
	})
}

func TestStdlibCache_rejects_cache_from_other_build(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	logger := commonlog.MockLogger{}

	require.NoError(t, saveStdlibToCache(logger, "test", "fp-old", []symbols_table.UnitModules{parseRoundTripSource(t)}))

	_, err := loadStdlibFromCache(logger, "test", "fp-new")
	assert.ErrorContains(t, err, "fingerprint mismatch")
}
