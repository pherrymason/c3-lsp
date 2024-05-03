package indexables

import protocol "github.com/tliron/glsp/protocol_3_16"

type DefBuilder struct {
	def Def
}

func NewDefBuilder(name string, module string, docId string) *DefBuilder {
	return &DefBuilder{
		def: Def{
			name: name,
			BaseIndexable: BaseIndexable{
				module:       NewModulePathFromString(module),
				moduleString: module,
				documentURI:  docId,
				Kind:         protocol.CompletionItemKindTypeParameter,
			},
		},
	}
}

func (d *DefBuilder) WithResolvesTo(resolvesTo string) *DefBuilder {
	d.def.resolvesTo = resolvesTo
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

func (d *DefBuilder) Build() Def {
	return d.def
}
