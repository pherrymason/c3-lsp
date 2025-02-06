package lsp

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"strings"
	"unicode/utf8"
)

type Position struct {
	Line, Column uint
}

func (p Position) ToProtocol() protocol.Position {
	return protocol.Position{
		Line:      protocol.UInteger(p.Line),
		Character: protocol.UInteger(p.Column),
	}
}

func (p Position) IsAfter(other Position) bool {
	return p.Line > other.Line || (p.Line == other.Line && p.Column >= other.Column)
}

func (p Position) IsBefore(other Position) bool {
	return p.Line < other.Line || (p.Line == other.Line && p.Column <= other.Column)
}

func (p Position) IndexIn(content string) int {
	// This code is modified from the gopls implementation found:
	// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.1.5:internal/span/utf16.go;l=70

	// In accordance with the LSP Spec:
	// https://microsoft.github.io/language-server-protocol/specifications/specification-3-16#textDocuments
	// self.Column represents utf-16 code units, not bytes and so we need to
	// convert utf-16 code units to a byte offset.

	// Find the byte offset for the line
	index := 0
	for row := uint(0); row < p.Line; row++ {
		content_ := content[index:]
		if next := strings.Index(content_, "\n"); next != -1 {
			index += next + 1
		} else {
			panic("Position.Line is past content")
			return 0
		}
	}

	// The index represents the byte offset from the beginning of the line
	// count self.Column utf-16 code units from the index byte offset.

	byteOffset := index
	remains := content[index:]
	chr := int(p.Column)

	for count := 1; count <= chr; count++ {

		if len(remains) <= 0 {
			// char goes past content
			// this a error
			panic("Position.Column is past content")
			return 0
		}

		r, w := utf8.DecodeRuneInString(remains)
		if r == '\n' {
			// Per the LSP spec:
			//
			// > If the Column value is greater than the line length it
			// > defaults back to the line length.
			break
		}

		remains = remains[w:]
		if r >= 0x10000 {
			// a two point rune
			count++
			// if we finished in a two point rune, do not advance past the first
			if count > chr {
				break
			}
		}
		byteOffset += w

	}

	return byteOffset
}

func NewPosition(line uint, char uint) Position {
	return Position{
		Line:   line,
		Column: char,
	}
}
func NewPositionFromProtocol(pos protocol.Position) Position {
	return Position{
		Line:   uint(pos.Line),
		Column: uint(pos.Character),
	}
}

func NewPositionFromIndex(index int, content string) Position {
	character := 0
	line := 0

	for i := 0; i < len(content); {
		r, size := utf8.DecodeRuneInString(content[i:])
		if i == index {
			// We've reached the wanted position skip and build position
			break
		}

		if r == '\n' {
			// We've found a new line
			line++
			character = 0
		} else {
			character++
		}

		// Advance the correct number of bytes
		i += size
	}

	return Position{
		Line:   uint(line),
		Column: uint(character),
	}
}

type Range struct {
	Start Position
	End   Position
}

func NewRange(startLine uint, startColumn uint, endLine uint, endColumn uint) Range {
	return Range{
		Start: Position{Line: startLine, Column: startColumn},
		End:   Position{Line: endLine, Column: endColumn},
	}
}

func NewRangeFromSitterNode(node *sitter.Node) Range {
	start := node.StartPoint()
	end := node.EndPoint()
	return Range{
		Start: Position{
			Column: uint(start.Column),
			Line:   uint(start.Row),
		},
		End: Position{
			Column: uint(end.Column),
			Line:   uint(end.Row),
		},
	}
}

func (r Range) HasPosition(position Position) bool {
	line := position.Line
	ch := position.Column

	if line >= r.Start.Line && line <= r.End.Line {
		// Exactly same line
		if line == r.Start.Line && line == r.End.Line {
			// Must be inside character ranges
			if ch >= r.Start.Column && ch <= r.End.Column {
				return true
			}
		} else {
			return true
		}
	}

	return false
}

func (r Range) IsInside(or Range) bool {
	return r.Start.IsAfter(or.Start) && r.End.IsBefore(or.End)
}

func (r Range) ToProtocol() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(r.Start.Line), Character: protocol.UInteger(r.Start.Column)},
		End:   protocol.Position{Line: protocol.UInteger(r.End.Line), Character: protocol.UInteger(r.End.Column)},
	}
}
