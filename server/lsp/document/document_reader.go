package document

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	"github.com/pherrymason/c3-lsp/option"
)

func (d *Document) SymbolInPosition2(position symbols.Position) option.Option[Token] {
	index := position.IndexIn(d.Content)
	return d.symbolInIndex2(index)
}

func (d *Document) symbolInIndex2(index int) option.Option[Token] {
	start, end, err := d.getSymbolRangeIndexesAtIndex(index)

	if err != nil {
		// Why is this logic here??
		// This causes problems, index+1 might be out of bounds!
		posRange := symbols.Range{
			Start: d.indexToPosition(index),
			End:   d.indexToPosition(index + 1),
		}
		return option.Some(NewToken(d.Content[index:index+1], posRange))
	}

	posRange := symbols.Range{
		Start: d.indexToPosition(start),
		End:   d.indexToPosition(end + 1),
	}
	return option.Some(NewToken(d.Content[start:end+1], posRange))
}

// Retrieves symbol present in current cursor position
// Details: It will grab only the symbol until the next `.` or `:`
// If you want to grab full chain of symbols present un current cursor, use FullSymbolInPosition
func (d *Document) SymbolInPosition(position symbols.Position) (Token, error) {
	index := position.IndexIn(d.Content)
	return d.symbolInIndex(index)
}

// Retrieves text from previous space until cursor position.
func (d *Document) SymbolBeforeCursor(position symbols.Position) (Token, error) {
	index := uint(position.IndexIn(d.Content))

	if rune(d.Content[index]) == rune(' ') {
		return Token{}, errors.New("No symbol at position")
	}

	start := uint(0)
	for i := index; i >= 0; i-- {
		r := rune(d.Content[i])
		//fmt.Printf("%c\n", r)
		if utils.IsSpaceOrNewline(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			start = i + 1
			break
		}
	}

	diff := index - start

	theRange := symbols.NewRange(
		position.Line, position.Character-diff,
		position.Line, position.Character,
	)
	return NewToken(d.Content[start:index], theRange), nil
}

func (d *Document) ParentSymbolInPosition(position symbols.Position) (Token, error) {
	if !d.HasPointInFrontSymbol(position) {
		return Token{}, errors.New("no previous '.' found")
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

const SymbolUntilSpace = 0     // Get symbol until previous space
const SymbolUntilSeparator = 1 // Get symbol until previous ./:

// Retrieves symbol
// Two modes: SymbolUntilSpace | SymbolUntilSeparator
func (d *Document) symbolInIndex(index int) (Token, error) {
	var start, end int
	var err error
	start, end, err = d.getSymbolRangeIndexesAtIndex(index)

	if err != nil {
		// Why is this logic here??
		// This causes problems, index+1 might be out of bounds!
		posRange := symbols.Range{
			Start: d.indexToPosition(index),
			End:   d.indexToPosition(index + 1),
		}
		return NewToken(d.Content[index:index+1], posRange), err
	}

	posRange := symbols.Range{
		Start: d.indexToPosition(start),
		End:   d.indexToPosition(end + 1),
	}
	return NewToken(d.Content[start:end+1], posRange), nil
}

func (d *Document) indexToPosition(index int) symbols.Position {
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

	return symbols.Position{
		Line:      uint(line),
		Character: uint(character),
	}
}

// Returns start and end index of symbol present in index.
// If no symbol is found in index, error will be returned
func (d *Document) getSymbolRangeIndexesAtIndex(index int) (int, int, error) {
	if !utils.IsAZ09_(rune(d.Content[index])) {
		return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(d.Content[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(d.Content) - 1
	for i := index; i < len(d.Content); i++ {
		r := rune(d.Content[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(d.Content) {
		return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(d.Content) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return symbolStart, symbolEnd, nil
}

// Returns start and end index of symbol present at index.
// It will search backwards until a space is found
func (d *Document) getFullSymbolRangeIndexesAtIndex(index int) (int, int, error) {
	if !utils.IsAZ09_(rune(d.Content[index])) {
		return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(d.Content[i])
		if r == rune(' ') {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(d.Content) - 1
	for i := index; i < len(d.Content); i++ {
		r := rune(d.Content[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(d.Content) {
		return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(d.Content) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return symbolStart, symbolEnd, nil
}
