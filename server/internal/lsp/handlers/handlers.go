package handlers

import (
	l "github.com/pherrymason/c3-lsp/internal/lsp/language"
	"github.com/pherrymason/c3-lsp/pkg/document"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
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
