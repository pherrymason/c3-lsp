package indexables

import "github.com/tliron/glsp/protocol_3_16"

type Enumerator struct {
	name  string
	value string
	BaseIndexable
}

func NewEnumerator(name string, value string, identifierPosition protocol.Range) Enumerator {
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

func (e Enumerator) GetDocumentURI() protocol.DocumentUri {
	return e.documentURI
}

func (e Enumerator) GetDeclarationRange() protocol.Range {
	return e.identifierRange
}

func (e Enumerator) GetDocumentRange() protocol.Range {
	return e.documentRange
}
