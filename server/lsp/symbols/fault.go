package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Fault struct {
	baseType  string
	constants []FaultConstant
	BaseIndexable
}

func NewFault(name string, baseType string, constants []FaultConstant, module string, docId string, idRange Range, docRange Range) Fault {
	return Fault{
		baseType:  baseType,
		constants: constants,
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

func (e Fault) GetType() string {
	return e.baseType
}

func (e *Fault) RegisterConstant(name string, value string, posRange Range) {
	e.constants = append(e.constants,
		FaultConstant{
			BaseIndexable: BaseIndexable{
				name:         name,
				moduleString: e.moduleString,
				documentURI:  e.documentURI,
				idRange:      posRange,
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

func (e Fault) GetConstants() []FaultConstant {
	return e.constants
}

func (e Fault) GetHoverInfo() string {
	return e.name
}

type FaultConstant struct {
	BaseIndexable
}

func (e FaultConstant) GetHoverInfo() string {
	return e.name
}

func NewFaultConstant(name string, idRange Range) FaultConstant {
	return FaultConstant{
		BaseIndexable: BaseIndexable{
			name:    name,
			idRange: idRange,
		},
	}
}
