package lsp

import (
	idx "github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func lsp_NewRangeFromRange(idxRange idx.Range) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(idxRange.Start.Line), Character: protocol.UInteger(idxRange.Start.Character)},
		End:   protocol.Position{Line: protocol.UInteger(idxRange.End.Line), Character: protocol.UInteger(idxRange.End.Character)},
	}
}

func lsp_NewPosition(line uint, char uint) protocol.Position {
	return protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(char)}
}
