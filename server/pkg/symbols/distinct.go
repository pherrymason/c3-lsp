package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Distinct struct {
	baseType *Type
	inline   bool
	BaseIndexable
}

func NewDistinct(name string, baseType *Type, inline bool, resolvesTo string, module string, docId string, idRange Range, docRange Range) Distinct {
	return Distinct{
		baseType: baseType,
		inline:   inline,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			docRange,
			protocol.CompletionItemKindTypeParameter,
		),
	}
}

func (d *Distinct) GetBaseType() *Type {
	return d.baseType
}

func (d *Distinct) IsInline() bool {
	return d.inline
}

func (d *Distinct) SetInline(inline bool) {
	d.inline = inline
}

func (d Distinct) GetHoverInfo() string {
	baseType := d.baseType.String()

	inline := ""
	if d.inline {
		inline = "inline "
	}

	return fmt.Sprintf("distinct %s = %s%s", d.Name, inline, baseType)
}

func (d Distinct) GetCompletionDetail() string {
	return "Type"
}
