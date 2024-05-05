package document

import (
	"github.com/pherrymason/c3-lsp/lsp/indexables"
)

type Token struct {
	Token string
	//position   protocol.Position
	TokenRange indexables.Range
}

func NewToken(text string, positionRange indexables.Range) Token {
	return Token{
		Token:      text,
		TokenRange: positionRange,
	}
}
