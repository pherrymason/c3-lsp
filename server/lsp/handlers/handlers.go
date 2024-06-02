package handlers

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
	l "github.com/pherrymason/c3-lsp/lsp/language"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
)

type Handlers struct {
	documents *document.DocumentStore
	language  *l.Language
	parser    *p.Parser
}

func NewHandlers(documents *document.DocumentStore,
	language *l.Language, parser *p.Parser) Handlers {
	return Handlers{
		documents: documents,
		language:  language,
		parser:    parser,
	}
}
