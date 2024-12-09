package lsp

import protocol "github.com/tliron/glsp/protocol_3_16"

type Position struct {
	Line, Column uint
}

func NewLSPPosition(pos protocol.Position) Position {
	return Position{
		Line:   uint(pos.Line),
		Column: uint(pos.Character),
	}
}
