package symbols

import (
	"encoding/json"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Range struct {
	Start Position
	End   Position
}

// MarshalJSON serializes Range as [[startLine, startChar], [endLine, endChar]]
func (r Range) MarshalJSON() ([]byte, error) {
	compact := [2][2]uint{
		{r.Start.Line, r.Start.Character},
		{r.End.Line, r.End.Character},
	}
	return json.Marshal(compact)
}

// UnmarshalJSON deserializes Range from [[startLine, startChar], [endLine, endChar]]
func (r *Range) UnmarshalJSON(data []byte) error {
	var compact [2][2]uint
	if err := json.Unmarshal(data, &compact); err != nil {
		return fmt.Errorf("failed to unmarshal range: %w", err)
	}

	r.Start = Position{Line: compact[0][0], Character: compact[0][1]}
	r.End = Position{Line: compact[1][0], Character: compact[1][1]}
	return nil
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
