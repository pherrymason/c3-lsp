package document

import (
	"github.com/pherrymason/c3-lsp/lsp/symbols"
)

type Token struct {
	Token      string
	TokenRange symbols.Range
}

func NewToken(text string, positionRange symbols.Range) Token {
	return Token{
		Token:      text,
		TokenRange: positionRange,
	}
}
