package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

type FaultDefBuilder struct {
	fault FaultDef
}

func NewFaultDefBuilder(name string, baseType string, module string, docId string) *FaultDefBuilder {
	return &FaultDefBuilder{
		fault: FaultDef{
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

func (eb *FaultDefBuilder) WithoutSourceCode() *FaultDefBuilder {
	eb.fault.HasSourceCode_ = false
	return eb
}

func (eb *FaultDefBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultDefBuilder {
	eb.fault.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultDefBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultDefBuilder {
	eb.fault.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultDefBuilder) WithDocs(docs string) *FaultDefBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	eb.fault.DocComment = &docComment
	return eb
}

func (eb *FaultDefBuilder) WithConstant(constant *FaultConstant) *FaultDefBuilder {
	eb.fault.constants = append(eb.fault.constants, constant)

	return eb
}

func (eb *FaultDefBuilder) Build() *FaultDef {
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
	eb.faultConstant.HasSourceCode_ = false
	return eb
}

func (eb *FaultConstantBuilder) WithFaultName(faultName string) *FaultConstantBuilder {
	eb.faultConstant.faultName = faultName
	return eb
}

func (eb *FaultConstantBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *FaultConstantBuilder {
	eb.faultConstant.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return eb
}

func (eb *FaultConstantBuilder) Build() *FaultConstant {
	return &eb.faultConstant
}
