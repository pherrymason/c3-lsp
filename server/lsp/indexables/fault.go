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
			moduleString: module,
			module:       NewModulePathFromString(module),
			documentURI:  docId,
			idRange:      idRange,
			docRange:     docRange,
			Kind:         protocol.CompletionItemKindEnum,
		},
	}
}

func (e Fault) GetName() string {
	return e.name
}

func (e Fault) GetType() string {
	return e.baseType
}

func (e Fault) GetModuleString() string {
	return e.moduleString
}

func (f Fault) GetModule() ModulePath {
	return f.module
}

func (f Fault) IsSubModuleOf(module ModulePath) bool {
	return f.module.IsSubModuleOf(module)
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
	name string
	BaseIndexable
}

func (f FaultConstant) GetName() string {
	return f.name
}

func (f FaultConstant) GetIdRange() Range {
	return f.idRange
}

func (f FaultConstant) GetDocumentRange() Range {
	return f.docRange
}
func (f FaultConstant) GetDocumentURI() string {
	return f.documentURI
}
func (e FaultConstant) GetHoverInfo() string {
	return e.name
}
func (e FaultConstant) GetKind() protocol.CompletionItemKind {
	return e.Kind
}
func (e FaultConstant) GetModuleString() string {
	return e.moduleString
}

func (f FaultConstant) GetModule() ModulePath {
	return f.module
}

func (f FaultConstant) IsSubModuleOf(module ModulePath) bool {
	return f.module.IsSubModuleOf(module)
}

func NewFaultConstant(name string, idRange Range) FaultConstant {
	return FaultConstant{
		name: name,
		BaseIndexable: BaseIndexable{
			idRange: idRange,
		},
	}
}
