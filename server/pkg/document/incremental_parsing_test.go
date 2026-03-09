package document

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeRange(startLine, startChar, endLine, endChar uint32) *protocol.Range {
	return &protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startChar},
		End:   protocol.Position{Line: endLine, Character: endChar},
	}
}

func incrementalChange(startLine, startChar, endLine, endChar uint32, text string) interface{} {
	return protocol.TextDocumentContentChangeEvent{
		Range: makeRange(startLine, startChar, endLine, endChar),
		Text:  text,
	}
}

func wholeChange(text string) interface{} {
	return protocol.TextDocumentContentChangeEventWhole{Text: text}
}

// ---------------------------------------------------------------------------
// Text content correctness
// ---------------------------------------------------------------------------

func TestApplyChanges_single_char_insert(t *testing.T) {
	doc := NewDocument("x", "hello world")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 5, 0, 5, " there"),
	})
	assert.Equal(t, "hello there world", doc.SourceCode.Text)
}

func TestApplyChanges_single_char_delete(t *testing.T) {
	doc := NewDocument("x", "hello world")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 5, 0, 11, ""),
	})
	assert.Equal(t, "hello", doc.SourceCode.Text)
}

func TestApplyChanges_replace_word(t *testing.T) {
	doc := NewDocument("x", "hello world")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 6, 0, 11, "Go"),
	})
	assert.Equal(t, "hello Go", doc.SourceCode.Text)
}

func TestApplyChanges_multiline_insert(t *testing.T) {
	doc := NewDocument("x", "line1\nline2\nline3")
	doc.ApplyChanges([]interface{}{
		incrementalChange(1, 5, 1, 5, "\nnew"),
	})
	assert.Equal(t, "line1\nline2\nnew\nline3", doc.SourceCode.Text)
}

func TestApplyChanges_multiline_delete(t *testing.T) {
	doc := NewDocument("x", "line1\nline2\nline3")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 5, 1, 5, ""),
	})
	assert.Equal(t, "line1\nline3", doc.SourceCode.Text)
}

func TestApplyChanges_whole_document_replacement(t *testing.T) {
	doc := NewDocument("x", "old content")
	doc.ApplyChanges([]interface{}{
		wholeChange("brand new content"),
	})
	assert.Equal(t, "brand new content", doc.SourceCode.Text)
}

func TestApplyChanges_multiple_sequential_changes(t *testing.T) {
	doc := NewDocument("x", "abc\ndef\n")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 1, 0, 3, "BC"),
		incrementalChange(1, 0, 1, 2, "DE"),
	})
	assert.Equal(t, "aBC\nDEf\n", doc.SourceCode.Text)
}

func TestApplyChanges_deterministic(t *testing.T) {
	changes := []interface{}{
		incrementalChange(0, 1, 0, 3, "BC"),
		incrementalChange(1, 0, 1, 2, "DE"),
	}

	docA := NewDocument("x", "abc\ndef\n")
	docA.ApplyChanges(changes)

	docB := NewDocument("x", "abc\ndef\n")
	docB.ApplyChanges(changes)

	assert.Equal(t, docA.SourceCode.Text, docB.SourceCode.Text)
}

// ---------------------------------------------------------------------------
// Tree validity after incremental parse
// ---------------------------------------------------------------------------

func TestApplyChanges_tree_is_non_nil_after_incremental_change(t *testing.T) {
	doc := NewDocument("x", "module foo; fn void main() {}")
	require.NotNil(t, doc.ContextSyntaxTree)

	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 28, 0, 28, " int x = 1;"),
	})
	assert.NotNil(t, doc.ContextSyntaxTree)
}

func TestApplyChanges_tree_root_not_nil_after_whole_replacement(t *testing.T) {
	doc := NewDocument("x", "module foo;")
	doc.ApplyChanges([]interface{}{
		wholeChange("module bar; fn void other() {}"),
	})
	require.NotNil(t, doc.ContextSyntaxTree)
	assert.NotNil(t, doc.ContextSyntaxTree.RootNode())
}

func TestApplyChanges_tree_root_node_reflects_new_content(t *testing.T) {
	doc := NewDocument("x", "module foo;")
	require.NotNil(t, doc.ContextSyntaxTree)

	// Insert a function declaration after the module declaration.
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 11, 0, 11, " fn void bar() {}"),
	})

	require.NotNil(t, doc.ContextSyntaxTree)
	root := doc.ContextSyntaxTree.RootNode()
	require.NotNil(t, root)
	// The root spans the entire source now.
	assert.Equal(t, uint32(0), root.StartByte())
	assert.Equal(t, uint32(len(doc.SourceCode.Text)), root.EndByte())
}

// ---------------------------------------------------------------------------
// Unicode / UTF-16 edge cases
// ---------------------------------------------------------------------------

func TestApplyChanges_unicode_bmp_character(t *testing.T) {
	// "é" is U+00E9, one UTF-16 code unit, but two UTF-8 bytes.
	doc := NewDocument("x", "café latte")
	// Replace "latte" (character 5..10) with "au lait".
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 5, 0, 10, "au lait"),
	})
	assert.Equal(t, "café au lait", doc.SourceCode.Text)
}

func TestApplyChanges_unicode_supplementary_plane(t *testing.T) {
	// 🎉 is U+1F389, one code point, 4 UTF-8 bytes, 2 UTF-16 code units.
	// "🎉 party" — character 0 occupies 2 UTF-16 code units.
	// To replace "party" (starting at UTF-16 character 3), use character offset 3.
	doc := NewDocument("x", "🎉 party")
	doc.ApplyChanges([]interface{}{
		incrementalChange(0, 3, 0, 8, "time"),
	})
	assert.Equal(t, "🎉 time", doc.SourceCode.Text)
}

// ---------------------------------------------------------------------------
// Helper function unit tests
// ---------------------------------------------------------------------------

func TestLineStartByteOffset(t *testing.T) {
	text := "abc\ndef\nghi"
	cases := []struct {
		line     uint32
		expected uint32
	}{
		{0, 0},
		{1, 4},
		{2, 8},
		{99, uint32(len(text))}, // out-of-bounds clamps to end
	}
	for _, tt := range cases {
		assert.Equal(t, tt.expected, lineStartByteOffset(text, tt.line), "line=%d", tt.line)
	}
}

func TestUtf16UnitsToByteOffset(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		nUnits   uint32
		expected uint32
	}{
		{"ascii", "hello", 3, 3},
		{"bmp unicode", "café", 3, 3},               // "caf" = 3 UTF-16 units = 3 bytes; stops before é
		{"supplementary plane emoji", "🎉 hi", 1, 4}, // 🎉 = 2 UTF-16 units, 4 UTF-8 bytes; 1 unit = 4 bytes
		{"zero units", "hello", 0, 0},
		{"past end", "hi", 100, 2}, // clamp to end of string
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := utf16UnitsToByteOffset(tt.s, tt.nUnits)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestComputeNewEndPoint_single_line(t *testing.T) {
	docText := "hello world"
	startPos := protocol.Position{Line: 0, Character: 6}
	pt := computeNewEndPoint(startPos, "Go", docText)
	assert.Equal(t, uint32(0), pt.Row)
	// "Go" = 2 bytes; start byte-col = 6
	assert.Equal(t, uint32(8), pt.Column)
}

func TestComputeNewEndPoint_multiline(t *testing.T) {
	docText := "hello\nworld"
	startPos := protocol.Position{Line: 0, Character: 5}
	pt := computeNewEndPoint(startPos, "\nline2\nline3", docText)
	// 2 newlines → row = 0 + 2 = 2
	assert.Equal(t, uint32(2), pt.Row)
	// Last line "line3" = 5 bytes
	assert.Equal(t, uint32(5), pt.Column)
}

func TestComputeNewEndPoint_empty_insertion(t *testing.T) {
	docText := "hello"
	startPos := protocol.Position{Line: 0, Character: 3}
	pt := computeNewEndPoint(startPos, "", docText)
	// Empty insertion: new end == start point
	assert.Equal(t, uint32(0), pt.Row)
	assert.Equal(t, uint32(3), pt.Column)
}

// ---------------------------------------------------------------------------
// Close / resource cleanup
// ---------------------------------------------------------------------------

func TestDocument_Close_sets_parser_to_nil(t *testing.T) {
	doc := NewDocument("x", "module foo;")
	require.NotNil(t, doc.parser)

	doc.Close()
	assert.Nil(t, doc.parser)
}

func TestDocument_Close_is_idempotent(t *testing.T) {
	doc := NewDocument("x", "module foo;")
	doc.Close()
	// Second close must not panic.
	assert.NotPanics(t, func() { doc.Close() })
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// generateLargeDocument creates a C3 source file with ~n lines.
func generateLargeDocument(n int) string {
	var sb strings.Builder
	sb.WriteString("module bench;\n")
	for i := 0; i < n; i++ {
		sb.WriteString("fn void f")
		sb.WriteString(strings.Repeat("x", 5))
		sb.WriteString("() { int x = ")
		sb.WriteString("1")
		sb.WriteString("; }\n")
	}
	return sb.String()
}

// benchmarkIncrementalEdit benchmarks incremental reparsing for a single
// keystroke on a long-lived document of nLines lines.
func benchmarkIncrementalEdit(b *testing.B, nLines int) {
	b.Helper()
	source := generateLargeDocument(nLines)
	midLine := uint32(nLines / 2)
	doc := NewDocument("bench", source)
	insert := true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if insert {
			doc.ApplyChanges([]interface{}{
				incrementalChange(midLine, 20, midLine, 20, "x"),
			})
		} else {
			doc.ApplyChanges([]interface{}{
				incrementalChange(midLine, 20, midLine, 21, ""),
			})
		}
		insert = !insert
	}
}

// benchmarkScratchEdit benchmarks full-reparse for a single keystroke,
// simulating the old implementation's behavior on every edit.
func benchmarkScratchEdit(b *testing.B, nLines int) {
	b.Helper()
	source := generateLargeDocument(nLines)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		midpoint := len(source) / 2
		newSource := source[:midpoint] + "x" + source[midpoint:]
		b.StartTimer()
		doc := NewDocument("bench", newSource)
		_ = doc.ContextSyntaxTree
	}
}

func BenchmarkApplyChanges_Incremental_500(b *testing.B)  { benchmarkIncrementalEdit(b, 500) }
func BenchmarkApplyChanges_Incremental_2000(b *testing.B) { benchmarkIncrementalEdit(b, 2000) }
func BenchmarkApplyChanges_Scratch_500(b *testing.B)      { benchmarkScratchEdit(b, 500) }
func BenchmarkApplyChanges_Scratch_2000(b *testing.B)     { benchmarkScratchEdit(b, 2000) }
