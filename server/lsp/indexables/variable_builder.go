package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type VariableBuilder struct {
	variable Variable
}

// NewVariableBuilder
func NewVariableBuilder(name string, variableType string, module string, docId string) *VariableBuilder {
	return &VariableBuilder{
		variable: Variable{
			Type: NewTypeFromString(variableType),
			BaseIndexable: BaseIndexable{
				name:         name,
				moduleString: module,
				module:       NewModulePathFromString(module),
				documentURI:  docId,
				Kind:         protocol.CompletionItemKindVariable,
			},
		},
	}
}

func (vb *VariableBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *VariableBuilder {
	vb.variable.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return vb
}

func (vb *VariableBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *VariableBuilder {
	vb.variable.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return vb
}

func (vb *VariableBuilder) Build() Variable {
	return vb.variable
}
