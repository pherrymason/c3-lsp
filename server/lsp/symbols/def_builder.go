package symbols

import protocol "github.com/tliron/glsp/protocol_3_16"

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

func (d *DefBuilder) WithResolvesTo(resolvesTo string) *DefBuilder {
	d.def.resolvesTo = resolvesTo
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
