package symbols

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Alias struct {
	resolvesTo     string
	resolvesToType option.Option[*Type]
	BaseIndexable
}

func NewAlias(name string, resolvesTo string, module string, docId string, idRange Range, docRange Range) Alias {
	return Alias{
		resolvesTo:     resolvesTo,
		resolvesToType: option.None[*Type](),
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

func NewAliasType(name string, resolvesTo Type, module string, docId string, idRange Range, docRange Range) Alias {
	return Alias{
		resolvesTo:     "",
		resolvesToType: option.Some(&resolvesTo),
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

func (d Alias) GetResolvesTo() string {
	return d.resolvesTo
}

func (d Alias) GetHoverInfo() string {
	if d.resolvesToType.IsNone() {
		return fmt.Sprintf("def %s = %s", d.Name, d.resolvesTo)
	}

	return fmt.Sprintf("def %s = %s", d.Name, d.resolvesToType.Get().String())
}

func (d Alias) GetCompletionDetail() string {
	if d.resolvesToType.IsSome() {
		return "Type"
	} else if strings.HasPrefix(d.Name, "@") {
		return "Alias for macro '" + d.resolvesTo + "'"
	} else {
		return "Alias for '" + d.resolvesTo + "'"
	}
}

func (d Alias) ResolvesToType() bool {
	return d.resolvesToType.IsSome()
}

func (d *Alias) ResolvedType() *Type {
	return d.resolvesToType.Get()
}
