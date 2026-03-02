package symbols

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type VariableBuilder struct {
	variable Variable
}

// NewVariableBuilder
func NewVariableBuilder(name string, variableType Type, module string, docId string) *VariableBuilder {
	return &VariableBuilder{
		variable: Variable{
			Type: variableType,
			BaseIndexable: BaseIndexable{
				Name:         name,
				ModuleString: module,
				Module:       NewModulePathFromString(module),
				DocumentURI:  docId,
				Kind:         protocol.CompletionItemKindVariable,
			},
		},
	}
}

func (vb *VariableBuilder) WithoutSourceCode() *VariableBuilder {
	vb.variable.BaseIndexable.HasSourceCode_ = false
	return vb
}

func (vb *VariableBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *VariableBuilder {
	vb.variable.BaseIndexable.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return vb
}

func (vb *VariableBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *VariableBuilder {
	vb.variable.BaseIndexable.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return vb
}

func (vb *VariableBuilder) WithDocs(docs string) *VariableBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	vb.variable.BaseIndexable.DocComment = &docComment
	return vb
}

func (vb *VariableBuilder) IsVarArg() *VariableBuilder {
	vb.variable.Arg.VarArg = true
	return vb
}

func (vb *VariableBuilder) WithArgDefault(value string) *VariableBuilder {
	vb.variable.Arg.Default = option.Some(value)
	return vb
}

func (vb *VariableBuilder) Build() *Variable {
	return &vb.variable
}
