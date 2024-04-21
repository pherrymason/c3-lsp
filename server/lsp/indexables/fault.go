package indexables

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Fault struct {
	name      string
	baseType  string
	constants []FaultConstant
	BaseIndexable
}

func NewFault(name string, baseType string, constants []FaultConstant, module string, docId string, idRange Range, docRange Range) Fault {
	return Fault{
		name:      name,
		baseType:  baseType,
		constants: constants,
		BaseIndexable: BaseIndexable{
			module:      module,
			documentURI: docId,
			idRange:     idRange,
			docRange:    docRange,
			Kind:        protocol.CompletionItemKindEnum,
		},
	}
}

func (e Fault) GetName() string {
	return e.name
}

func (e Fault) GetType() string {
	return e.baseType
}

func (e Fault) GetModule() string {
	return e.module
}

func (e Fault) GetKind() protocol.CompletionItemKind {
	return e.Kind
}

func (e Fault) GetDocumentURI() string {
	return e.documentURI
}

func (e Fault) GetIdRange() Range {
	return e.idRange
}

func (e Fault) GetDocumentRange() Range {
	return e.docRange
}

func (e *Fault) RegisterConstant(name string, value string, posRange Range) {
	e.constants = append(e.constants,
		FaultConstant{
			name: name,
			BaseIndexable: BaseIndexable{
				module:      e.module,
				documentURI: e.documentURI,
				idRange:     posRange,
			},
		})
}

func (e *Fault) AddConstant(constants []FaultConstant) {
	e.constants = constants
}

func (e Fault) HasConstant(identifier string) bool {
	for _, constant := range e.constants {
		if constant.name == identifier {
			return true
		}
	}

	return false
}

func (e Fault) GetConstant(identifier string) FaultConstant {
	for _, constant := range e.constants {
		if constant.name == identifier {
			return constant
		}
	}

	panic(fmt.Sprint(identifier, " enumerator not found"))
}

func (e Fault) GetHoverInfo() string {
	return e.name
}

type FaultConstant struct {
	name string
	BaseIndexable
}

func (f FaultConstant) GetName() string {
	return f.name
}

func (f FaultConstant) GetIdRange() Range {
	return f.idRange
}

func NewFaultConstant(name string, idRange Range) FaultConstant {
	return FaultConstant{
		name: name,
		BaseIndexable: BaseIndexable{
			idRange: idRange,
		},
	}
}
