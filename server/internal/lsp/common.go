package lsp

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Position struct {
	Line, Column uint
}

func (pos Position) ToProtocol() protocol.Position {
	return protocol.Position{
		Line:      protocol.UInteger(pos.Line),
		Character: protocol.UInteger(pos.Column),
	}
}

func NewLSPPosition(pos protocol.Position) Position {
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
