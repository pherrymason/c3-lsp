package document

import (
	"context"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/cst"
	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const STRUCT_SEPARATOR = '.'
const MODULE_SEPARATOR = ':'

type Document struct {
	URI string
	// Monotonic version from LSP text document sync.
	Version           int32
	ContextSyntaxTree *sitter.Tree
	SourceCode        code.SourceCode
	// parser is kept alive across edits to enable incremental re-parsing.
	// Must be closed via Close() when the document is no longer needed.
	parser *sitter.Parser
}

func NewDocument(docId string, sourceCode string) Document {
	p := cst.NewSitterParser()
	tree, _ := p.ParseCtx(context.Background(), nil, []byte(cst.NormalizeSource(sourceCode)))
	return Document{
		URI:               docId,
		Version:           0,
		ContextSyntaxTree: tree,
		SourceCode:        code.NewSourceCode(sourceCode),
		parser:            p,
	}
}

func NewDocumentFromString(docId string, sourceCode string) Document {
	return NewDocument(docId, sourceCode)
}

func NewDocumentFromDocURI(docURI string, sourceCode string, docVersion int32) *Document {
	normalizedPath := utils.NormalizePath(docURI)
	doc := NewDocumentFromString(normalizedPath, sourceCode)
	doc.Version = docVersion

	return &doc
}

// Close releases the parser held by this document.
// It must be called when the document is no longer in use to avoid resource leaks.
func (d *Document) Close() {
	if d.parser != nil {
		d.parser.Close()
		d.parser = nil
	}
}

// ApplyChanges updates the content of the Document from LSP textDocument/didChange events.
// For incremental changes (with a Range), it uses tree-sitter's incremental parsing to
// reparse only the affected region, which is significantly faster than a full reparse.
func (d *Document) ApplyChanges(changes []interface{}) {
	for _, change := range changes {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEvent:
			d.applyIncrementalChange(c)
		case protocol.TextDocumentContentChangeEventWhole:
			// Full document replacement — reparse from scratch.
			d.SourceCode.Text = c.Text
			d.reparseFromScratch()
		}
	}
}

// applyIncrementalChange applies a single ranged LSP change using tree-sitter's
// incremental parsing. This avoids reparsing the entire document.
func (d *Document) applyIncrementalChange(c protocol.TextDocumentContentChangeEvent) {
	oldText := d.SourceCode.Text

	// 1. Compute byte offsets for the edited range in the old text.
	startByte := uint32(c.Range.Start.IndexIn(oldText))
	oldEndByte := uint32(c.Range.End.IndexIn(oldText))

	// 2. Compute the new end byte after the replacement text is inserted.
	newTextBytes := []byte(c.Text)
	newEndByte := startByte + uint32(len(newTextBytes))

	// 3. Compute Points (row, byte-column) for EditInput.
	//    tree-sitter Point.Column is a byte offset from the start of the line.
	startPoint := lspPositionToPoint(c.Range.Start, oldText)
	oldEndPoint := lspPositionToPoint(c.Range.End, oldText)
	newEndPoint := computeNewEndPoint(c.Range.Start, c.Text, oldText)

	// 4. Apply text change to SourceCode.
	startIndex := int(startByte)
	oldEndIndex := int(oldEndByte)
	d.SourceCode.Text = d.SourceCode.Text[:startIndex] + c.Text + d.SourceCode.Text[oldEndIndex:]

	// 5. Notify the old tree of the edit so it can update node positions.
	if d.ContextSyntaxTree != nil {
		d.ContextSyntaxTree.Edit(sitter.EditInput{
			StartIndex:  startByte,
			OldEndIndex: oldEndByte,
			NewEndIndex: newEndByte,
			StartPoint:  startPoint,
			OldEndPoint: oldEndPoint,
			NewEndPoint: newEndPoint,
		})
	}

	// 6. Incrementally reparse — tree-sitter reuses unchanged nodes.
	// NormalizeSource is byte-offset-preserving, so CST positions remain valid
	// against the original SourceCode.Text.
	newTree, err := d.parser.ParseCtx(context.Background(), d.ContextSyntaxTree, []byte(cst.NormalizeSource(d.SourceCode.Text)))
	if err != nil {
		// On parse error fall back to a full reparse.
		d.reparseFromScratch()
		return
	}

	d.ContextSyntaxTree = newTree
}

// reparseFromScratch performs a full reparse of the current SourceCode.Text.
// Used for whole-document replacements and error recovery.
func (d *Document) reparseFromScratch() {
	tree, err := d.parser.ParseCtx(context.Background(), nil, []byte(cst.NormalizeSource(d.SourceCode.Text)))
	if err != nil {
		// Extremely unlikely (empty document parse failure). Keep stale tree.
		return
	}
	d.ContextSyntaxTree = tree
}

// lspPositionToPoint converts an LSP Position to a tree-sitter Point.
// tree-sitter expects Column as a byte offset from the start of the line,
// not a UTF-16 code unit count as LSP uses.
func lspPositionToPoint(pos protocol.Position, text string) sitter.Point {
	// Walk line-by-line to find the byte offset of the start of the line,
	// then advance pos.Character UTF-16 code units to get the byte column.
	lineStartByte := lineStartByteOffset(text, uint32(pos.Line))
	colByte := utf16UnitsToByteOffset(text[lineStartByte:], uint32(pos.Character))

	return sitter.Point{
		Row:    uint32(pos.Line),
		Column: colByte,
	}
}

// computeNewEndPoint calculates the tree-sitter Point that corresponds to the
// end of the newly inserted text.
func computeNewEndPoint(startPos protocol.Position, insertedText string, docText string) sitter.Point {
	if insertedText == "" {
		// Pure deletion: new end == start.
		return lspPositionToPoint(startPos, docText)
	}

	lines := strings.Split(insertedText, "\n")
	lastLine := lines[len(lines)-1]

	if len(lines) == 1 {
		// Insertion stays on the same line.
		startPoint := lspPositionToPoint(startPos, docText)
		return sitter.Point{
			Row:    startPoint.Row,
			Column: startPoint.Column + uint32(len([]byte(lastLine))),
		}
	}

	// Multi-line insertion: new row = startRow + number of newlines added.
	newRow := uint32(startPos.Line) + uint32(len(lines)-1)
	// New column is the byte length of the last line of the inserted text.
	newCol := uint32(len([]byte(lastLine)))
	return sitter.Point{Row: newRow, Column: newCol}
}

// lineStartByteOffset returns the byte offset of the beginning of the given
// (0-indexed) line within text. Returns len(text) if line is out of bounds.
func lineStartByteOffset(text string, line uint32) uint32 {
	offset := uint32(0)
	for row := uint32(0); row < line; row++ {
		next := strings.Index(text[offset:], "\n")
		if next == -1 {
			return uint32(len(text))
		}
		offset += uint32(next) + 1
	}
	return offset
}

// utf16UnitsToByteOffset advances nUnits UTF-16 code units into the string s
// and returns the corresponding byte offset. Stops at end-of-line or string.
func utf16UnitsToByteOffset(s string, nUnits uint32) uint32 {
	byteOffset := uint32(0)
	remaining := uint32(0)
	for _, r := range s {
		if r == '\n' || r == '\r' {
			break
		}
		if remaining >= nUnits {
			break
		}
		w := uint32(runeUTF8Len(r))
		u := uint32(1)
		if r >= 0x10000 {
			u = 2 // Supplementary plane: 2 UTF-16 code units.
		}
		byteOffset += w
		remaining += u
	}
	return byteOffset
}

// runeUTF8Len returns the number of bytes needed to encode r in UTF-8.
func runeUTF8Len(r rune) int {
	switch {
	case r < 0x80:
		return 1
	case r < 0x800:
		return 2
	case r < 0x10000:
		return 3
	default:
		return 4
	}
}

func (d *Document) HasPointInFrontSymbol(position symbols.Position) bool {
	index := position.IndexIn(d.SourceCode.Text)
	start, _, _ := d.getWordIndexLimits(index)

	if start == 0 {
		return false
	}

	if start-1 < len(d.SourceCode.Text) && rune(d.SourceCode.Text[start-1]) == STRUCT_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) HasModuleSeparatorInFrontSymbol(position symbols.Position) bool {
	index := position.IndexIn(d.SourceCode.Text)
	start, _, _ := d.getWordIndexLimits(index)

	if start < 2 {
		return false
	}

	if start-1 < len(d.SourceCode.Text) && start-2 < len(d.SourceCode.Text) &&
		rune(d.SourceCode.Text[start-1]) == MODULE_SEPARATOR && rune(d.SourceCode.Text[start-2]) == MODULE_SEPARATOR {
		return true
	}

	return false
}

func (d *Document) GetSymbolPositionAtPosition(position symbols.Position) (symbols.Position, error) {
	index := position.IndexIn(d.SourceCode.Text)
	startIndex, _, _error := d.getWordIndexLimits(index)

	symbolStartPosition := d.indexToPosition(startIndex)

	return symbolStartPosition, _error
}

// GetLineLength returns the character count (in UTF-16 code units) of the given line.
// Lines are 0-indexed. Returns 0 if line number is out of bounds.
func (d *Document) GetLineLength(line uint) uint {
	lines := strings.Split(d.SourceCode.Text, "\n")

	if int(line) >= len(lines) {
		return 0
	}

	// Count UTF-16 code units for LSP protocol compatibility
	lineText := lines[line]
	count := uint(0)
	for _, r := range lineText {
		if r < 0x10000 {
			count++ // BMP character - 1 code unit
		} else {
			count += 2 // Supplementary character - 2 code units (surrogate pair)
		}
	}

	return count
}

// RewindPosition moves the position back by one character.
// If at the start of a line, moves to the end of the previous line.
// If at the start of the document, returns the position unchanged.
// This fixes the bug in Position.RewindCharacter() which didn't handle line boundaries.
func (d *Document) RewindPosition(p symbols.Position) symbols.Position {
	// Not at line start - simply decrement character
	if p.Character > 0 {
		return symbols.NewPosition(p.Line, p.Character-1)
	}

	// At line start - need to go to previous line
	if p.Line > 0 {
		prevLineLength := d.GetLineLength(p.Line - 1)
		return symbols.NewPosition(p.Line-1, prevLineLength)
	}

	// At document start - can't rewind further
	return p
}
