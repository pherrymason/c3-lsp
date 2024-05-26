package sourcecode

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	"github.com/pherrymason/c3-lsp/option"
)

type symbolLimits struct {
	start int
	end   int
}

type SourceCode struct {
	Text string
}

func NewSourceCode(text string) SourceCode {
	return SourceCode{Text: text}
}

// Tries to find the symbol under cursor position
func (s SourceCode) SymbolInPosition(cursorPosition symbols.Position) Word {
	index := cursorPosition.IndexIn(s.Text)
	pattern := `^[a-zA-Z0-9_]+$`
	re, _ := regexp.Compile(pattern)

	baseFound := false
	gettingAccess := false
	gettingModule := false

	wb := NewWordBuilderE()
	var accessPath []Word
	var modulePath []Word

	for {
		if index < 0 {
			break
		}

		limitsOpt := s.getWordIndexLimits(index, true)
		if limitsOpt.IsNone() {
			panic("error")
		}

		limits := limitsOpt.Get()
		symbol := s.Text[limits.start : limits.end+1]
		posRange := symbols.Range{
			Start: s.indexToPosition(limits.start),
			End:   s.indexToPosition(limits.end + 1),
		}

		if re.MatchString(symbol) {
			if gettingAccess {
				accessPath = append([]Word{{
					text:      symbol,
					textRange: posRange,
				}}, accessPath...)
			} else if gettingModule {
				modulePath = append([]Word{{
					text:      symbol,
					textRange: posRange,
				}}, modulePath...)
			} else {
				wb.WithText(symbol, posRange)
				baseFound = true
			}
			index = limits.start - 1
		} else {
			if !baseFound {
				wb.WithText(symbol, posRange)
				baseFound = true
			}

			if symbol == "." {
				gettingAccess = true
			} else if symbol == ":" {
				gettingAccess = false
				gettingModule = true
			} else if symbol == "(" || symbol == ")" {

			} else {
				// End
				break
			}
			index--
		}
	}
	wb.WithAccessPath(accessPath).WithModule(modulePath)

	return wb.Build()
}

// Tries to find the symbol under cursor position
func (s SourceCode) SymbolInPosition2(cursorPosition symbols.Position) Word {
	index := cursorPosition.IndexIn(s.Text)

	limitsOpt := s.getWordIndexLimits(index, false)

	var start, end int
	isPreviousCharacterDot := false
	isPreviousCharacterModuleSplit := false
	if limitsOpt.IsNone() {
		if s.Text[index] == '.' {
			start = index
			end = index + 1
			isPreviousCharacterDot = true
		} else if s.Text[index] == ':' {
			start = index
			end = index + 1
			isPreviousCharacterModuleSplit = true
		} else {
			panic("No word found in cursor position")
		}
	} else {
		start = limitsOpt.Get().start
		end = limitsOpt.Get().end + 1
		isPreviousCharacterDot = s.Text[start-1] == '.'
		isPreviousCharacterModuleSplit = s.Text[start-1] == ':'
	}

	posRange := symbols.Range{
		Start: s.indexToPosition(start),
		End:   s.indexToPosition(end),
	}

	wb := NewWordBuilder(s.Text[start:end], posRange)

	// Try to get accessPath if exists
	if start > 0 && isPreviousCharacterDot {
		//accessPath := s.extractAccessPath(posRange.Start)
		var accessPath []Word
		startIndex := start
		for i := start; i >= 0; i-- {
			r := rune(s.Text[i])
			//fmt.Printf("%c\n", r)
			if utils.IsAZ09_(r) || r == '.' /*|| r == ':'*/ {
				startIndex = i
			} else {
				break
			}
		}

		sentence := s.Text[startIndex:start]
		tokens := strings.Split(sentence, ".")
		for _, token := range tokens {
			if len(token) == 0 {
				continue
			}

			endIndex := startIndex + len(token)
			accessPath = append(
				accessPath,
				Word{
					text: token,
					textRange: symbols.Range{
						Start: s.indexToPosition(startIndex),
						End:   s.indexToPosition(endIndex),
					},
				},
			)
			startIndex = endIndex + 1 // 1 is to take '.' into account
		}

		wb.WithAccessPath(accessPath)
	}

	// Try to get modulePath if exists
	if start > 0 && isPreviousCharacterModuleSplit {
		n := start
		modulePath := []Word{}
		consume := "colon"
		colonCount := 0
		lastColon := start
		exit := false
		for {
			//fmt.Printf("%c\n", s.Text[n])
			switch consume {
			case "colon":
				if s.Text[n] == ':' {
					colonCount++
				}

				if colonCount == 2 {
					consume = "path"
					colonCount = 0
					lastColon = n
				}
			case "path":
				r := rune(s.Text[n])
				if !utils.IsAZ09_(r) {
					// First invalid character found, that means previous iteration contained first character of symbol
					modulePath = append([]Word{{
						text: s.Text[n+1 : lastColon],
						textRange: symbols.Range{
							Start: s.indexToPosition(n + 1),
							End:   s.indexToPosition(lastColon),
						},
					}}, modulePath...)

					if r == ':' {
						consume = "colon"
						colonCount = 1
					} else if r != ':' {
						exit = true
					}
				}
			}

			if exit {
				break
			}
			n--
		}
		wb.WithModule(modulePath)

		/*rewindedStart := start
		skipOnNonModSplit := false
		colonCount := 0
		modulePath := []Word{}
		firstColonConsumed := false
		for {
			if s.Text[rewindedStart] == ':' {
				skipOnNonModSplit = true
				colonCount++
			}

			if colonCount == 2 {
				fmt.Println("New path")

			}

			if s.Text[rewindedStart] != ':' && skipOnNonModSplit {
				rewindedStart++
				break
			}
			rewindedStart--
		}

		modulePath := s.modulePathAtIndex(rewindedStart)
		if modulePath.IsSome() {
			wb.WithModule(modulePath.Get())
		}*/
	}

	return wb.Build()
}

// Returns start and end index of symbol present in index.
// If no symbol is found in index, error will be returned
func (s SourceCode) getWordIndexLimits(index int, returnAnyway bool) option.Option[symbolLimits] {
	if index >= len(s.Text) {
		return option.None[symbolLimits]()
	}

	if !utils.IsAZ09_(rune(s.Text[index])) {
		if returnAnyway {
			return option.Some(symbolLimits{index, index})
		} else {
			return option.None[symbolLimits]()
		}
		//return 0, 0, errors.New("No symbol at position")
	}

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(s.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i + 1
			break
		}
	}

	symbolEnd := len(s.Text) - 1
	for i := index; i < len(s.Text); i++ {
		r := rune(s.Text[i])
		if !utils.IsAZ09_(r) {
			// First invalid character found, that means previous iteration contained last character of symbol
			symbolEnd = i - 1
			break
		}
	}

	if symbolStart > len(s.Text) {
		panic("start limit greater than content")
		//return 0, 0, errors.New("wordStart out of bounds")
	} else if symbolEnd > len(s.Text) {
		panic("end limit greater than content")
		//return 0, 0, errors.New("wordEnd out of bounds")
	} else if symbolStart > symbolEnd {
		panic("start limit greater than end limit")
		//return 0, 0, errors.New("wordStart > wordEnd!")
	}

	return option.Some(symbolLimits{symbolStart, symbolEnd})
}

func (d SourceCode) indexToPosition(index int) symbols.Position {
	character := 0
	line := 0

	for i := 0; i < len(d.Text); {
		r, size := utf8.DecodeRuneInString(d.Text[i:])
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

func (s SourceCode) modulePathAtIndex(index int) option.Option[string] {
	/*if !utils.IsAZ09_(rune(s.Text[index])) {
		return option.None[string]()
	}*/

	symbolStart := 0
	for i := index; i >= 0; i-- {
		r := rune(s.Text[i])
		if utils.IsAZ09_(r) || r == ':' {
			// First invalid character found, that means previous iteration contained first character of symbol
			symbolStart = i
		} else {
			break
		}
	}

	return option.Some(s.Text[symbolStart:index])
}

/*
func (s *SourceCode) symbolAtIndex(index int) (Word, error) {
	var start, end int
	var err error
	//start, end, err = s.getWordIndexLimits(index)
	limitsOpt := s.getWordIndexLimits(index, true)

	if err != nil {
		// Why is this logic here??
		// This causes problems, index+1 might be out of bounds!
		posRange := symbols.Range{
			Start: s.indexToPosition(index),
			End:   s.indexToPosition(index + 1),
		}
		return NewWord(s.Text[index:index+1], posRange), err
	}

	posRange := symbols.Range{
		Start: s.indexToPosition(start),
		End:   s.indexToPosition(end + 1),
	}
	return NewWord(s.Text[start:end+1], posRange), nil
}
*/
