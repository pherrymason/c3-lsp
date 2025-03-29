package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Fault struct {
	//baseType  string
	//constants []*FaultConstant
	BaseIndexable
}

func NewFault(name string, module string, docId string, idRange Range, docRange Range) Fault {
	fault := Fault{
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindEnum,
		),
	}
	return fault
}

func (e Fault) GetType() string {
	return ""
}

func (e *Fault) RegisterConstant(name string, posRange Range) {
	constant := &FaultConstant{
		BaseIndexable: BaseIndexable{
			name:         name,
			moduleString: e.moduleString,
			module:       e.module,
			documentURI:  e.documentURI,
			idRange:      posRange,
			docRange:     posRange,
		},
	}
	//e.constants = append(e.constants, constant)
	e.Insert(constant)
}

func (e *Fault) AddConstants(constants []*FaultConstant) {
	/*e.constants = constants
	for _, constant := range constants {
		e.Insert(constant)
	}*/
}

func (e Fault) HasConstant(identifier string) bool {
	/*for _, constant := range e.constants {
		if constant.name == identifier {
			return true
		}
	}
	*/
	return false
}

func (e Fault) GetConstant(identifier string) *FaultConstant {
	/*for _, constant := range e.constants {
		if constant.name == identifier {
			return constant
		}
	}*/

	panic(fmt.Sprint(identifier, " enumerator not found"))
}

func (e Fault) GetHoverInfo() string {
	return e.name
}

func (e Fault) GetCompletionDetail() string {
	return "Fault"
}

type FaultConstant struct {
	BaseIndexable
}

func (e *FaultConstant) GetFaultName() string {
	return ""
}

func (e *FaultConstant) GetFaultFQN() string {
	return fmt.Sprintf("%s::%s", e.GetModule().GetName(), e.GetFaultName())
}

func (e FaultConstant) GetHoverInfo() string {
	return e.name
}

func (e FaultConstant) GetCompletionDetail() string {
	return "Fault Constant"
}

func NewFaultConstant(name string, module string, docId string, idRange Range, docRange Range) *FaultConstant {
	return &FaultConstant{
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindEnumMember,
		),
	}
}
