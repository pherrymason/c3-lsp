package document

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
)

// Retrieves text from previous space until cursor position.
func (d *Document) SymbolBeforeCursor(position symbols.Position) (sourcecode.Word, error) {
	index := uint(position.IndexIn(d.SourceCode.Text))
	if len(d.SourceCode.Text) == 0 || index >= uint(len(d.SourceCode.Text)) {
		return sourcecode.Word{}, errors.New("no symbol at position")
	}

	currentChar := rune(d.SourceCode.Text[index])
	if currentChar == rune(' ') || currentChar == rune('.') {
		return sourcecode.Word{}, errors.New("no symbol at position")
	}

	start := 0
	for i := int(index); i >= 0; i-- {
		r := rune(d.SourceCode.Text[i])
		//fmt.Printf("%c\n", r)
		if utils.IsSpaceOrNewline(r) || rune('.') == r {
			// First invalid character found, that means previous iteration contained first character of symbol
			start = i + 1
			break
		}
	}

	diff := uint(int(index) - start)

	theRange := symbols.NewRange(
		position.Line, position.Character-diff,
		position.Line, position.Character,
	)
	return sourcecode.NewWord(d.SourceCode.Text[start:int(index)], theRange), nil
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
	for index != 0 {

		r := rune(d.SourceCode.Text[index])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			foundPreviousChar = true
			break
		}
		index -= 1
	}

	if foundPreviousChar {
		start, end, errSymbol := d.getWordIndexLimits(index)
		if errSymbol != nil {
			return sourcecode.Word{}, errSymbol
		}

		posRange := symbols.Range{
			Start: d.indexToPosition(start),
			End:   d.indexToPosition(end + 1),
		}
		parentSymbol := sourcecode.NewWord(d.SourceCode.Text[start:end+1], posRange)

		return parentSymbol, errSymbol
	}

	return sourcecode.Word{}, errors.New("no previous symbol found")
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
	if index < 0 || index >= len(d.SourceCode.Text) {
		return 0, 0, errors.New("no symbol at position")
	}
	for index > 0 && !utf8.RuneStart(d.SourceCode.Text[index]) {
		index--
	}

	r, size := utf8.DecodeRuneInString(d.SourceCode.Text[index:])
	if r == utf8.RuneError && size == 0 {
		return 0, 0, errors.New("no symbol at position")
	}

	if !utils.IsAZ09_(r) {
		return 0, 0, errors.New("no symbol at position")
	}

	symbolStart := index
	for current := index; current > 0; {
		prev := current - 1
		for prev > 0 && !utf8.RuneStart(d.SourceCode.Text[prev]) {
			prev--
		}

		pr, _ := utf8.DecodeRuneInString(d.SourceCode.Text[prev:])
		if !utils.IsAZ09_(pr) {
			break
		}

		symbolStart = prev
		current = prev
	}

	symbolEndExclusive := index + size
	for symbolEndExclusive < len(d.SourceCode.Text) {
		nextRune, nextSize := utf8.DecodeRuneInString(d.SourceCode.Text[symbolEndExclusive:])
		if !utils.IsAZ09_(nextRune) {
			break
		}
		symbolEndExclusive += nextSize
	}
	symbolEnd := symbolEndExclusive - 1

	if symbolStart < 0 || symbolStart >= len(d.SourceCode.Text) {
		return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd < 0 || symbolEnd >= len(d.SourceCode.Text) {
		return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		return 0, 0, errors.New("wordStart > wordEnd")
	}

	return symbolStart, symbolEnd, nil
}
