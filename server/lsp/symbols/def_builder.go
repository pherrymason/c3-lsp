package symbols

import (
	"github.com/pherrymason/c3-lsp/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type DefBuilder struct {
	def Def
}

func NewDefBuilder(name string, module string, docId string) *DefBuilder {
	return &DefBuilder{
		def: Def{
			BaseIndexable: NewBaseIndexable(
				name,
				module,
				docId,
				NewRange(0, 0, 0, 0),
				NewRange(0, 0, 0, 0),
				protocol.CompletionItemKindTypeParameter,
			),
		},
	}
}

func (d *DefBuilder) WithName(name string) *DefBuilder {
	d.def.name = name
	return d
}

func (d *DefBuilder) WithResolvesTo(resolvesTo string) *DefBuilder {
	d.def.resolvesTo = resolvesTo
	return d
}

func (d *DefBuilder) WithResolvesToType(resolvesTo Type) *DefBuilder {
	d.def.resolvesToType = option.Some(&resolvesTo)
	return d
}

func (d *DefBuilder) WithoutSourceCode() *DefBuilder {
	d.def.BaseIndexable.hasSourceCode = false
	return d
}

func (d *DefBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *DefBuilder {
	d.def.BaseIndexable.idRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *DefBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *DefBuilder {
	d.def.BaseIndexable.docRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *DefBuilder) Build() *Def {
	return &d.def
}
