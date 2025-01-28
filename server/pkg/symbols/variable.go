package symbols

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type ArgInfo struct {
	// Whether this variable came from a vararg, that is, ...args
	VarArg bool

	// The default value for the argument that originated this variable, if any
	Default option.Option[string]
}

type Variable struct {
	Type Type
	Arg  ArgInfo
	BaseIndexable
}

func NewVariable(name string, variableType Type, module string, docId string, idRange Range, docRange Range) Variable {
	return Variable{
		Type: variableType,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindVariable,
		),
	}
}

func NewConstant(name string, variableType Type, module string, docId string, idRange Range, docRange Range) Variable {
	return Variable{
		Type: variableType,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindConstant,
		),
	}
}

func (v *Variable) GetArgInfo(arg ArgInfo) *ArgInfo {
	return &v.Arg
}

func (v *Variable) SetArgInfo(arg ArgInfo) {
	v.Arg = arg
}

func (v *Variable) GetType() *Type {
	return &v.Type
}

func (v Variable) IsConstant() bool {
	return v.Kind == protocol.CompletionItemKindConstant
}

func (v Variable) GetHoverInfo() string {
	return fmt.Sprintf("%s %s", v.GetType(), v.GetName())
}

func (v Variable) GetCompletionDetail() string {
	return v.GetType().String()
}
