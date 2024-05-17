package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type GenericParameter struct {
	BaseIndexable
}

func NewGenericParameter(name string, module string, docId string, idRange Range, docRange Range) *GenericParameter {
	return &GenericParameter{
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindTypeParameter,
		),
	}
}
