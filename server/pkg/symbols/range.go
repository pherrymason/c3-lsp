package symbols

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Range struct {
	Start Position
	End   Position
}

func (r Range) HasPosition(position Position) bool {
	line := uint(position.Line)
	ch := uint(position.Character)

	if line >= r.Start.Line && line <= r.End.Line {
		// Exactly same line
		if line == r.Start.Line && line == r.End.Line {
			// Must be inside character ranges
			if ch >= r.Start.Character && ch <= r.End.Character {
				return true
			}
		} else {
			return true
		}
	}

	return false
}

func (r Range) IsBeforePosition(position Position) bool {
	if r.Start.Line > position.Line ||
		(r.Start.Line == position.Line && r.Start.Character > position.Character) {
		return true
	}

	return false
}

func (r Range) IsAfterPosition(position Position) bool {
	if r.End.Line < position.Line ||
		(r.End.Line == position.Line && r.End.Character < position.Character) {
		return true
	}

	return false
}

func (r Range) IsAfter(crange Range) bool {
	if r.End.Line > crange.End.Line {
		return true
	}

	if r.End.Line == crange.End.Line && r.End.Character > crange.End.Character {
		return true
	}

	return false
}

func (r Range) ToLSP() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: uint32(r.Start.Line), Character: uint32(r.Start.Character)},
		End:   protocol.Position{Line: uint32(r.End.Line), Character: uint32(r.End.Character)},
	}
}

func NewRange(startLine uint, startChar uint, endLine uint, endChar uint) Range {
	return Range{
		Start: NewPosition(startLine, startChar),
		End:   NewPosition(endLine, endChar),
	}
}

func NewRangeFromTreeSitterPositions(start sitter.Point, end sitter.Point) Range {
	return Range{
		Start: NewPositionFromTreeSitterPoint(start),
		End:   NewPositionFromTreeSitterPoint(end),
	}
}
