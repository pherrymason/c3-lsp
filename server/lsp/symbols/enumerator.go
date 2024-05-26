package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enumerator struct {
	value            string
	associatedValues []Variable
	BaseIndexable
}

func NewEnumerator(name string, value string, associatedValues []Variable, module string, idRange Range, docId string) *Enumerator {
	return &Enumerator{
		value:            value,
		associatedValues: associatedValues,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			NewRange(0, 0, 0, 0),
			protocol.CompletionItemKindEnumMember,
		),
	}
}

func (e Enumerator) GetAssociatedValues() []Variable {
	return e.associatedValues
}

func (e Enumerator) GetHoverInfo() string {
	return fmt.Sprintf("%s: %s", e.name, e.value)
}
