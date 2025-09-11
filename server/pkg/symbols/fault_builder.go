package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type FaultBuilder struct {
	fault Fault
}

func NewFaultBuilder(name string, module string, docId string) *FaultBuilder {
	return &FaultBuilder{
		fault: Fault{
			BaseIndexable: BaseIndexable{
				name:         name,
				moduleString: module,
				module:       NewModulePathFromString(module),
				documentURI:  docId,
				Kind:         protocol.CompletionItemKindEnum,
			},
		},
	}
}

func (eb *FaultBuilder) WithoutSourceCode() *FaultBuilder {
	eb.fault.BaseIndexable.hasSourceCode = false
	return eb
}

func (eb *FaultBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultBuilder {
	eb.fault.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultBuilder) WithDocs(docs string) *FaultBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	eb.fault.BaseIndexable.docComment = &docComment
	return eb
}

func (eb *FaultBuilder) WithConstant(constant *FaultConstant) *FaultBuilder {
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
				name:         name,
				moduleString: module,
				module:       NewModulePathFromString(module),
				documentURI:  docId,
				Kind:         protocol.CompletionItemKindEnumMember,
			},
		},
	}
}

func (eb *FaultConstantBuilder) WithoutSourceCode() *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.hasSourceCode = false
	return eb
}

func (eb *FaultConstantBuilder) WithFaultName(faultName string) *FaultConstantBuilder {
	//eb.faultConstant.faultName = faultName
	return eb
}

func (eb *FaultConstantBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) Build() *FaultConstant {
	return &eb.faultConstant
}
