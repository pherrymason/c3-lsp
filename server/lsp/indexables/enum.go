package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type Enum struct {
	name        string
	baseType    string
	enumerators []Enumerator
	BaseIndexable
}

func NewEnum(name string, baseType string, enumerators []Enumerator, identifierRangePosition protocol.Range, documentRangePosition protocol.Range, docId protocol.DocumentUri) Enum {
	return Enum{
		name:        name,
		baseType:    baseType,
		enumerators: enumerators,
		BaseIndexable: BaseIndexable{
			documentURI:     docId,
			identifierRange: identifierRangePosition,
			documentRange:   documentRangePosition,
			Kind:            protocol.CompletionItemKindEnum,
		},
	}
}

func (e Enum) GetName() string {
	return e.name
}

func (e Enum) GetKind() protocol.CompletionItemKind {
	return e.Kind
}

func (e Enum) GetDocumentURI() protocol.DocumentUri {
	return e.documentURI
}

func (e Enum) GetDeclarationRange() protocol.Range {
	return e.documentRange
}

func (e Enum) GetDocumentRange() protocol.Range {
	return e.identifierRange
}

func (e *Enum) AddEnumerators(enumerators []Enumerator) {
	e.enumerators = enumerators
}
