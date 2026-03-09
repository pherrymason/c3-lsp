package symbols

import (
	"github.com/pherrymason/c3-lsp/pkg/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type AliasBuilder struct {
	def Alias
}

func NewAliasBuilder(name string, module string, docId string) *AliasBuilder {
	return &AliasBuilder{
		def: Alias{
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

func (d *AliasBuilder) WithName(name string) *AliasBuilder {
	d.def.Name = name
	return d
}

func (d *AliasBuilder) WithResolvesTo(resolvesTo string) *AliasBuilder {
	d.def.resolvesTo = resolvesTo
	return d
}

func (d *AliasBuilder) WithResolvesToType(resolvesTo Type) *AliasBuilder {
	d.def.resolvesToType = option.Some(&resolvesTo)
	return d
}

func (d *AliasBuilder) WithoutSourceCode() *AliasBuilder {
	d.def.HasSourceCode_ = false
	return d
}

func (d *AliasBuilder) WithIdentifierRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *AliasBuilder {
	d.def.IdRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *AliasBuilder) WithDocumentRange(lineStart uint, CharStart uint, lineEnd uint, CharEnd uint) *AliasBuilder {
	d.def.DocRange = NewRange(lineStart, CharStart, lineEnd, CharEnd)
	return d
}

func (d *AliasBuilder) WithDocs(docs string) *AliasBuilder {
	// Only modules, functions and macros can have contracts, so a string is enough
	// Theoretically, there can be custom contracts here, but the stdlib shouldn't be creating them
	docComment := NewDocComment(docs)
	d.def.DocComment = &docComment
	return d
}

func (d *AliasBuilder) Build() *Alias {
	return &d.def
}
