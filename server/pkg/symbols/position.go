package symbols

import (
	"strings"
	"unicode/utf8"

	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Position struct {
	Line      uint
	Character uint
}

func NewPosition(line uint, char uint) Position {
	return Position{Line: line, Character: char}
}

func NewPositionFromTreeSitterPoint(position sitter.Point) Position {
	return NewPosition(uint(position.Row), uint(position.Column))
}

func NewPositionFromLSPPosition(position protocol.Position) Position {
	return Position{
		Line:      uint(position.Line),
		Character: uint(position.Character),
	}
}

func (p Position) ToLSPPosition() protocol.Position {
	return protocol.Position{
		Line:      uint32(p.Line),
		Character: uint32(p.Character),
	}
}

func (p Position) IndexIn(content string) int {
	// This code is modified from the gopls implementation found:
	// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.1.5:internal/span/utf16.go;l=70

	// In accordance with the LSP Spec:
	// https://microsoft.github.io/language-server-protocol/specifications/specification-3-16#textDocuments
	// p.Character represents utf-16 code units, not bytes and so we need to
	// convert utf-16 code units to a byte offset.

	// Find the byte offset for the line
	index := 0
	for row := uint(0); row < p.Line; row++ {
		content_ := content[index:]
		if next := strings.Index(content_, "\n"); next != -1 {
			index += next + 1
		} else {
			return len(content)
		}
	}

	// The index represents the byte offset from the beginning of the line
	// count p.Character utf-16 code units from the index byte offset.

	byteOffset := index
	remains := content[index:]
	chr := int(p.Character)

	for count := 1; count <= chr; count++ {

		if len(remains) <= 0 {
			// Per LSP behavior, clamp to the end of line/content.
			return byteOffset
		}

		r, w := utf8.DecodeRuneInString(remains)
		if r == '\n' {
			// Per the LSP spec:
			//
			// > If the character value is greater than the line length it
			// > defaults back to the line length.
			break
		}
		if r == '\r' && strings.HasPrefix(remains[w:], "\n") {
			// CRLF line ending: treat as end-of-line for character clamping,
			// same behavior as '\n'.
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
