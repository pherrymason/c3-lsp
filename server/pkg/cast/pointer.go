package cast

import protocol "github.com/tliron/glsp/protocol_3_16"

func BoolPtr(v bool) *bool {
	b := v
	return &b
}

func StrPtr(v string) *string {
	return &v
}

// protocol
func CompletionItemKindPtr(kind protocol.CompletionItemKind) *protocol.CompletionItemKind {
	return &kind
}
