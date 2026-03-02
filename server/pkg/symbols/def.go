package symbols

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Def struct {
	resolvesTo     string
	resolvesToType option.Option[*Type]
	BaseIndexable
}

func NewDef(name string, resolvesTo string, module string, docId string, idRange Range, docRange Range) Def {
	return Def{
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

func NewDefType(name string, resolvesTo Type, module string, docId string, idRange Range, docRange Range) Def {
	return Def{
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

func (d Def) GetResolvesTo() string {
	return d.resolvesTo
}

func (d Def) GetHoverInfo() string {
	if d.resolvesToType.IsNone() {
		return fmt.Sprintf("def %s = %s", d.Name, d.resolvesTo)
	}

	resolvesTo := d.resolvesToType.Get().String()

	// TODO: Maybe this should be moved to 'String()' or some other type method
	if len(d.resolvesToType.Get().genericArguments) > 0 {
		genericNames := []string{}
		for _, generic := range d.resolvesToType.Get().genericArguments {
			genericNames = append(genericNames, generic.String())
		}
		resolvesTo += "(<" + strings.Join(genericNames, ", ") + ">)"
	}

	return fmt.Sprintf("def %s = %s", d.Name, resolvesTo)
}

func (d Def) GetCompletionDetail() string {
	if d.resolvesToType.IsSome() {
		return "Type"
	} else if strings.HasPrefix(d.Name, "@") {
		return "Alias for macro '" + d.resolvesTo + "'"
	} else {
		// No semantic information
		// TODO: Resolve the identifier and display its information?
		return "Alias for '" + d.resolvesTo + "'"
	}
}

func (d Def) ResolvesToType() bool {
	return d.resolvesToType.IsSome()
}

func (d *Def) ResolvedType() *Type {
	return d.resolvesToType.Get()
}
