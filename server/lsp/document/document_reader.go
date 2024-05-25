package document

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/lsp/document/sourcecode"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/utils"
)

// Retrieves symbol present in current cursor position
// Details: It will grab only the symbol until the next `.` or `:`
// If you want to grab full chain of symbols present un current cursor, use FullSymbolInPosition
func (d *Document) SymbolInPositionDeprecated(position symbols.Position) (sourcecode.Word, error) {
	index := position.IndexIn(d.SourceCode.Text)
	return d.symbolInIndexDeprecated(index)
}

// Retrieves text from previous space until cursor position.
func (d *Document) SymbolBeforeCursor(position symbols.Position) (sourcecode.Word, error) {
	index := uint(position.IndexIn(d.SourceCode.Text))

	currentChar := rune(d.SourceCode.Text[index])
	if currentChar == rune(' ') || currentChar == rune('.') {
		return sourcecode.Word{}, errors.New("No symbol at position")
	}

	start := uint(0)
	for i := index; i >= 0; i-- {
		r := rune(d.SourceCode.Text[i])
		//fmt.Printf("%c\n", r)
		if utils.IsSpaceOrNewline(r) || rune('.') == r {
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
	return sourcecode.NewWord(d.SourceCode.Text[start:index], theRange), nil
}

func (d *Document) ParentSymbolInPosition(position symbols.Position) (sourcecode.Word, error) {
	if !d.HasPointInFrontSymbol(position) {
		return sourcecode.Word{}, errors.New("no previous '.' found")
	}

	index := position.IndexIn(d.SourceCode.Text)
	start, _, errRange := d.getWordIndexLimits(index)
	if errRange != nil {
		return sourcecode.Word{}, errRange
	}

	index = start - 2
	foundPreviousChar := false
	for {
		if index == 0 {
			break
		}
		r := rune(d.SourceCode.Text[index])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			foundPreviousChar = true
			break
		}
		index -= 1
	}

	if foundPreviousChar {
		parentSymbol, errSymbol := d.symbolInIndexDeprecated(index)

		return parentSymbol, errSymbol
	}

	return sourcecode.Word{}, errors.New("No previous symbol found")
}

const SymbolUntilSpace = 0     // Get symbol until previous space
const SymbolUntilSeparator = 1 // Get symbol until previous ./:

// Retrieves symbol
func (d *Document) symbolInIndexDeprecated(index int) (sourcecode.Word, error) {
	var start, end int
	var err error
	start, end, err = d.getWordIndexLimits(index)

	if err != nil {
		// Why is this logic here??
		// This causes problems, index+1 might be out of bounds!
		posRange := symbols.Range{
			Start: d.indexToPosition(index),
			End:   d.indexToPosition(index + 1),
		}
		return sourcecode.NewWord(d.SourceCode.Text[index:index+1], posRange), err
	}

	posRange := symbols.Range{
		Start: d.indexToPosition(start),
		End:   d.indexToPosition(end + 1),
	}
	return sourcecode.NewWord(d.SourceCode.Text[start:end+1], posRange), nil
}

func (d *Document) indexToPosition(index int) symbols.Position {
	character := 0
	line := 0

	for i := 0; i < len(d.SourceCode.Text); {
		r, size := utf8.DecodeRuneInString(d.SourceCode.Text[i:])
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
func (d *Document) getWordIndexLimits(index int) (int, int, error) {
	if !utils.IsAZ09_(rune(d.SourceCode.Text[index])) {
		return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(d.SourceCode.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(d.SourceCode.Text) - 1
	for i := index; i < len(d.SourceCode.Text); i++ {
		r := rune(d.SourceCode.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(d.SourceCode.Text) {
		panic("start limit greater than content")
		//return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(d.SourceCode.Text) {
		panic("end limit greater than content")
		//return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		panic("start limit greater than end limit")
		//return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return symbolStart, symbolEnd, nil
}

// Returns start and end index of symbol present at index.
// It will search backwards until a space is found
func (d *Document) getFullSymbolRangeIndexesAtIndex(index int) (int, int, error) {
	if !utils.IsAZ09_(rune(d.SourceCode.Text[index])) {
		return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(d.SourceCode.Text[i])
		if r == rune(' ') {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(d.SourceCode.Text) - 1
	for i := index; i < len(d.SourceCode.Text); i++ {
		r := rune(d.SourceCode.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(d.SourceCode.Text) {
		return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(d.SourceCode.Text) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return symbolStart, symbolEnd, nil
}
