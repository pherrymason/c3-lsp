package protocol

import (
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func Lsp_NewRangeFromRange(idxRange symbols.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(idxRange.Start.Line), Character: protocol.UInteger(idxRange.Start.Character)},
		End:   protocol.Position{Line: protocol.UInteger(idxRange.End.Line), Character: protocol.UInteger(idxRange.End.Character)},
	}
}

func Lsp_NewPosition(line uint, char uint) protocol.Position {
	return protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(char)}
}
