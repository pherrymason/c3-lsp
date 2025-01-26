package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enum struct {
	baseType    string
	enumerators []*Enumerator
	BaseIndexable
}

func NewEnum(name string, baseType string, enumerators []*Enumerator, module string, docId string, idRange Range, docRange Range) Enum {
	return Enum{
		baseType:    baseType,
		enumerators: enumerators,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindEnum,
		),
	}
}

func (e Enum) GetType() string {
	return e.baseType
}

func (e *Enum) RegisterEnumerator(name string, value string, posRange Range) {
	enumerator := NewEnumerator(name, value, []Variable{}, "", posRange, e.documentURI)
	e.enumerators = append(e.enumerators, enumerator)
	e.Insert(enumerator)
}

func (e *Enum) AddEnumerators(enumerators []*Enumerator) {
	e.enumerators = enumerators
	e.children = []Indexable{}
	for _, en := range enumerators {
		e.Insert(en)
	}
}

func (e *Enum) HasEnumerator(identifier string) bool {
	for _, enumerator := range e.enumerators {
		if enumerator.name == identifier {
			return true
		}
	}

	return false
}

func (e *Enum) GetAssociatedValues() []Variable {
	if len(e.enumerators) > 0 {
		return e.enumerators[0].AssociatedValues
	} else {
		return []Variable{}
	}
}

func (e *Enum) GetEnumerator(identifier string) *Enumerator {
	for _, enumerator := range e.enumerators {
		if enumerator.name == identifier {
			return enumerator
		}
	}

	panic(fmt.Sprint(identifier, " enumerator not found"))
}

func (e *Enum) GetEnumerators() []*Enumerator {
	return e.enumerators
}

func (e Enum) GetHoverInfo() string {
	return e.name
}

func (e Enum) GetCompletionDetail() string {
	// While we could specify 'Type' here like in struct,
	// enums behave quite differently overall, especially
	// regarding instantiation
	return "Enum"
}
