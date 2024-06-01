package symbols

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Def struct {
	resolvesTo     string
	resolvesToType option.Option[Type]
	BaseIndexable
}

func NewDef(name string, resolvesTo string, module string, docId string, idRange Range, docRange Range) Def {
	return Def{
		resolvesTo:     resolvesTo,
		resolvesToType: option.None[Type](),
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

func NewDefType(name string, resolvesTo Type, module string, docId string, idRange Range, docRange Range) Def {
	return Def{
		resolvesTo:     "",
		resolvesToType: option.Some(resolvesTo),
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

func (d Def) GetResolvesTo() string {
	return d.resolvesTo
}

func (d Def) GetHoverInfo() string {
	return fmt.Sprintf("def %s = %s", d.name, d.resolvesTo)
}
