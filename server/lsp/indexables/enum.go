package indexables

import (
	"fmt"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enum struct {
	name        string
	baseType    string
	enumerators []Enumerator
	BaseIndexable
}

func NewEnum(name string, baseType string, enumerators []Enumerator, identifierRangePosition Range, documentRangePosition Range, docId string) Enum {
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

func (e Enum) GetDocumentURI() string {
	return e.documentURI
}

func (e Enum) GetDeclarationRange() Range {
	return e.documentRange
}

func (e Enum) GetDocumentRange() Range {
	return e.identifierRange
}

func (e *Enum) AddEnumerators(enumerators []Enumerator) {
	e.enumerators = enumerators
}

func (e Enum) HasEnumerator(identifier string) bool {
	for _, enumerator := range e.enumerators {
		if enumerator.name == identifier {
			return true
		}
	}

	return false
}

func (e Enum) GetEnumerator(identifier string) Enumerator {
	for _, enumerator := range e.enumerators {
		if enumerator.name == identifier {
			return enumerator
		}
	}

	panic(fmt.Sprint(identifier, " enumerator not found"))
}
