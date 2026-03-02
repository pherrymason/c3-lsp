package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type FaultBuilder struct {
	fault Fault
}

func NewFaultBuilder(name string, baseType string, module string, docId string) *FaultBuilder {
	return &FaultBuilder{
		fault: Fault{
			baseType: baseType,
			BaseIndexable: BaseIndexable{
				Name:         name,
				ModuleString: module,
				Module:       NewModulePathFromString(module),
				DocumentURI:  docId,
				Kind:         protocol.CompletionItemKindEnum,
			},
		},
	}
}

func (eb *FaultBuilder) WithoutSourceCode() *FaultBuilder {
	eb.fault.BaseIndexable.HasSourceCode_ = false
	return eb
}

func (eb *FaultBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithDocs(docs string) *FaultBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	eb.fault.BaseIndexable.DocComment = &docComment
	return eb
}

func (eb *FaultBuilder) WithConstant(constant *FaultConstant) *FaultBuilder {
	eb.fault.constants = append(eb.fault.constants, constant)

	return eb
}

func (eb *FaultBuilder) Build() *Fault {
	return &eb.fault
}

// FaultConstantBuilder
type FaultConstantBuilder struct {
	faultConstant FaultConstant
}

func NewFaultConstantBuilder(name string, module string, docId string) *FaultConstantBuilder {
	return &FaultConstantBuilder{
		faultConstant: FaultConstant{
			BaseIndexable: BaseIndexable{
				Name:         name,
				ModuleString: module,
				Module:       NewModulePathFromString(module),
				DocumentURI:  docId,
				Kind:         protocol.CompletionItemKindEnumMember,
			},
		},
	}
}

func (eb *FaultConstantBuilder) WithoutSourceCode() *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.HasSourceCode_ = false
	return eb
}

func (eb *FaultConstantBuilder) WithFaultName(faultName string) *FaultConstantBuilder {
	eb.faultConstant.faultName = faultName
	return eb
}

func (eb *FaultConstantBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) Build() *FaultConstant {
	return &eb.faultConstant
}
