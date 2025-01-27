package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enumerator struct {
	value            string
	AssociatedValues []Variable
	EnumName         string
	BaseIndexable
}

func NewEnumerator(name string, value string, associatedValues []Variable, enumName string, module string, idRange Range, docId string) *Enumerator {
	enumerator := &Enumerator{
		value:            value,
		AssociatedValues: associatedValues,
		EnumName:         enumName,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			NewRange(0, 0, 0, 0),
			protocol.CompletionItemKindEnumMember,
		),
	}

	for _, av := range associatedValues {
		enumerator.InsertNestedScope(&av)
	}

	return enumerator
}

func (e *Enumerator) GetEnumName() string {
	return e.EnumName
}

func (e *Enumerator) GetEnumFQN() string {
	return fmt.Sprintf("%s::%s", e.GetModule().GetName(), e.GetEnumName())
}

func (e Enumerator) GetHoverInfo() string {
	return fmt.Sprintf("%s: %s", e.name, e.value)
}

func (e Enumerator) GetCompletionDetail() string {
	return "Enum Value"
}
