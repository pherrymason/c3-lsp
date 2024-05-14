package indexables

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Variable struct {
	Type Type
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

func (v Variable) GetType() Type {
	return v.Type
}

func (v Variable) IsConstant() bool {
	return v.Kind == protocol.CompletionItemKindConstant
}

func (v Variable) GetHoverInfo() string {
	return fmt.Sprintf("%s %s", v.GetType(), v.GetName())
}
