package lsp

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
