package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enumerator struct {
	value            string
	AssociatedValues []Variable
	BaseIndexable
}

func NewEnumerator(name string, value string, associatedValues []Variable, module string, idRange Range, docId string) *Enumerator {
	enumerator := &Enumerator{
		value:            value,
		AssociatedValues: associatedValues,
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

func (e Enumerator) GetHoverInfo() string {
	return fmt.Sprintf("%s: %s", e.name, e.value)
}
