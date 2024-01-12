package indexables

import "github.com/tliron/glsp/protocol_3_16"

type Enumerator struct {
	name  string
	value string
	BaseIndexable
}

func NewEnumerator(name string, value string, identifierPosition Range) Enumerator {
	return Enumerator{
		name:  name,
		value: value,
		BaseIndexable: BaseIndexable{
			identifierRange: identifierPosition,
			Kind:            protocol.CompletionItemKindEnumMember,
		},
	}
}

func (e Enumerator) GetName() string {
	return e.name
}

func (e Enumerator) GetKind() protocol.CompletionItemKind {
	return e.Kind
}

func (e Enumerator) GetDocumentURI() string {
	return e.documentURI
}

func (e Enumerator) GetDeclarationRange() Range {
	return e.identifierRange
}

func (e Enumerator) GetDocumentRange() Range {
	return e.documentRange
}
