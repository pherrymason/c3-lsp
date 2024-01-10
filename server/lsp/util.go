package lsp

import (
	sitter "github.com/smacker/go-tree-sitter"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func treeSitterPoint2Position(point sitter.Point) protocol.Position {
	return protocol.Position{Line: point.Row, Character: point.Column}
}

func treeSitterPoints2Range(start sitter.Point, end sitter.Point) protocol.Range {
	return protocol.Range{
		Start: treeSitterPoint2Position(start),
		End:   treeSitterPoint2Position(end),
	}
}

func NewPosition(line protocol.UInteger, char protocol.UInteger) protocol.Position {
	return protocol.Position{Line: line, Character: char}
}

func NewRange(startLine protocol.UInteger, startChar protocol.UInteger, endLine protocol.UInteger, endChar protocol.UInteger) protocol.Range {
	return protocol.Range{
		Start: NewPosition(startLine, startChar),
		End:   NewPosition(endLine, endChar),
	}
}
