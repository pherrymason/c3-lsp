package document

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (d *Document) SymbolInPosition(position protocol.Position) (Token, error) {
	index := position.IndexIn(d.Content)
	return d.symbolInIndex(index)
}

func (d *Document) ParentSymbolInPosition(position protocol.Position) (Token, error) {
	if !d.HasPointInFrontSymbol(position) {
		return Token{}, errors.New("No previous '.' found.")
	}

	index := position.IndexIn(d.Content)
	start, _, errRange := d.getSymbolRangeIndexesAtIndex(index)
	if errRange != nil {
		return Token{}, errRange
	}

	index = start - 2
	foundPreviousChar := false
	for {
		if index == 0 {
			break
		}
		r := rune(d.Content[index])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			foundPreviousChar = true
			break
		}
		index -= 1
	}

	if foundPreviousChar {
		parentSymbol, errSymbol := d.symbolInIndex(index)

		return parentSymbol, errSymbol
	}

	return Token{}, errors.New("No previous symbol found")
}

func (d *Document) symbolInIndex(index int) (Token, error) {
	start, end, err := d.getSymbolRangeIndexesAtIndex(index)

	if err != nil {
		posRange := indexables.Range{
			Start: d.indexToPosition(index),
			End:   d.indexToPosition(index + 1),
		}
		return NewToken(d.Content[index:index+1], posRange), err
	}

	posRange := indexables.Range{
		Start: d.indexToPosition(start),
		End:   d.indexToPosition(end + 1),
	}
	return NewToken(d.Content[start:end+1], posRange), nil
}

func (d *Document) indexToPosition(index int) indexables.Position {
	character := 0
	line := 0

	for i := 0; i < len(d.Content); {
		r, size := utf8.DecodeRuneInString(d.Content[i:])
		if i == index {
			// We've reached the wanted position skip and build position
			break
		}

		if r == '\n' {
			// We've found a new line
			line++
			character = 0
		} else {
			character++
		}

		// Advance the correct number of bytes
		i += size
	}

	return indexables.Position{
		Line:      uint(line),
		Character: uint(character),
	}
}
