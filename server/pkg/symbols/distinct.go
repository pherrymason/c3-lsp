package symbols

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type TypeDef struct {
	baseType *Type
	inline   bool
	BaseIndexable
}

func NewTypeDef(name string, baseType *Type, inline bool, resolvesTo string, module string, docId string, idRange Range, docRange Range) TypeDef {
	return TypeDef{
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

func (d *TypeDef) GetBaseType() *Type {
	if d.baseType == nil {
		empty := Type{}
		return &empty
	}

	return d.baseType
}

func (d *TypeDef) IsInline() bool {
	return d.inline
}

func (d *TypeDef) SetInline(inline bool) {
	d.inline = inline
}

func (d TypeDef) GetHoverInfo() string {
	if d.baseType == nil {
		if d.inline {
			return fmt.Sprintf("distinct %s = inline ?", d.Name)
		}

		return fmt.Sprintf("distinct %s = ?", d.Name)
	}

	baseType := d.baseType.String()

	inline := ""
	if d.inline {
		inline = "inline "
	}

	return fmt.Sprintf("distinct %s = %s%s", d.Name, inline, baseType)
}

func (d TypeDef) GetCompletionDetail() string {
	return "Type"
}
