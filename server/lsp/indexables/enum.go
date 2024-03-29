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

func NewEnum(name string, baseType string, enumerators []Enumerator, module string, docId string, identifierRangePosition Range, documentRangePosition Range) Enum {
	return Enum{
		name:        name,
		baseType:    baseType,
		enumerators: enumerators,
		BaseIndexable: BaseIndexable{
			module:          module,
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

func (e Enum) GetModule() string {
	return e.module
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

func (e *Enum) RegisterEnumerator(name string, value string, posRange Range) {
	e.enumerators = append(e.enumerators,
		NewEnumerator(name, value, "", posRange, e.documentURI))
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

func (e Enum) GetHoverInfo() string {
	return e.name
}
