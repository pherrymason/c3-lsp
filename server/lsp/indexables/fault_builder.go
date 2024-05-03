package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type FaultBuilder struct {
	fault Fault
}

func NewFaultBuilder(name string, baseType string, module string, docId string) *FaultBuilder {
	return &FaultBuilder{
		fault: Fault{
			name:     name,
			baseType: baseType,
			BaseIndexable: BaseIndexable{
				moduleString: module,
				module:       NewModulePathFromString(module),
				documentURI:  docId,
				Kind:         protocol.CompletionItemKindEnum,
			},
		},
	}
}

func (eb *FaultBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithConstant(constant FaultConstant) *FaultBuilder {
	eb.fault.constants = append(eb.fault.constants, constant)

	return eb
}

func (eb *FaultBuilder) Build() Fault {
	return eb.fault
}

// FaultConstantBuilder
type FaultConstantBuilder struct {
	faultConstant FaultConstant
}

func NewFaultConstantBuilder(name string, docId string) *FaultConstantBuilder {
	return &FaultConstantBuilder{
		faultConstant: FaultConstant{
			name: name,
			BaseIndexable: BaseIndexable{
				documentURI: docId,
				Kind:        protocol.CompletionItemKindEnumMember,
			},
		},
	}
}

func (eb *FaultConstantBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) Build() FaultConstant {
	return eb.faultConstant
}
