package symbols

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type FaultDef struct {
	baseType  string
	constants []*FaultConstant
	BaseIndexable
}

func NewFaultDef(name string, baseType string, constants []*FaultConstant, module string, docId string, idRange Range, docRange Range) FaultDef {
	fault := FaultDef{
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

	fault.AddConstants(constants)

	return fault
}

func (e FaultDef) GetType() string {
	return e.baseType
}

func (e *FaultDef) RegisterConstant(name string, value string, posRange Range) {
	constant := &FaultConstant{
		faultName: e.GetName(),
		BaseIndexable: BaseIndexable{
			Name:         name,
			ModuleString: e.ModuleString,
			Module:       e.Module,
			DocumentURI:  e.DocumentURI,
			IdRange:      posRange,
			DocRange:     posRange,
		},
	}
	e.constants = append(e.constants, constant)
	e.Insert(constant)
}

func (e *FaultDef) AddConstants(constants []*FaultConstant) {
	e.constants = constants
	for _, constant := range constants {
		e.Insert(constant)
	}
}

func (e FaultDef) HasConstant(identifier string) bool {
	for _, constant := range e.constants {
		if constant.Name == identifier {
			return true
		}
	}

	return false
}

func (e FaultDef) GetConstant(identifier string) option.Option[*FaultConstant] {
	for _, constant := range e.constants {
		if constant.Name == identifier {
			return option.Some(constant)
		}
	}

	return option.None[*FaultConstant]()
}

func (e FaultDef) GetConstants() []*FaultConstant {
	return e.constants
}

func (e FaultDef) GetHoverInfo() string {
	return e.Name
}

func (e FaultDef) GetCompletionDetail() string {
	return "Fault"
}

type FaultConstant struct {
	faultName string
	BaseIndexable
}

func (e *FaultConstant) GetFaultName() string {
	return e.faultName
}

func (e *FaultConstant) GetFaultFQN() string {
	return fmt.Sprintf("%s::%s", e.GetModule().GetName(), e.GetFaultName())
}

func (e FaultConstant) GetHoverInfo() string {
	return e.Name
}

func (e FaultConstant) GetCompletionDetail() string {
	return "Fault Constant"
}

func NewFaultConstant(name string, faultName string, module string, docId string, idRange Range, docRange Range) *FaultConstant {
	return &FaultConstant{
		faultName: faultName,
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
