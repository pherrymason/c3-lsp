package sourcecode

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
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
func (s SourceCode) SymbolInPosition(cursorPosition symbols.Position, docModules *symbols_table.UnitModules) Word {
	index := cursorPosition.IndexIn(s.Text)
	pattern := `^[a-zA-Z0-9_]+$`
	re, _ := regexp.Compile(pattern)

	baseFound := false
	gettingAccess := false
	gettingModule := false
	ignoreSymbol := false

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

		// Just ignore content inside parenthesis
		if re.MatchString(symbol) {
			if gettingAccess && !ignoreSymbol {
				accessPath = append([]Word{{
					text:      symbol,
					textRange: posRange,
				}}, accessPath...)
			} else if gettingModule && !ignoreSymbol {
				modulePath = append([]Word{{
					text:      symbol,
					textRange: posRange,
				}}, modulePath...)
			} else if !ignoreSymbol {
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
			} else if gettingAccess && symbol == "(" {
				ignoreSymbol = false
			} else if gettingAccess && symbol == ")" {
				ignoreSymbol = true
			} else {
				// End
				break
			}
			index--
		}
	}
	wb.WithAccessPath(accessPath).WithModule(modulePath)

	wb = tryToResolveFullModulePaths(wb, docModules, cursorPosition)

	return wb.Build()
}

func (s SourceCode) RewindBeforePreviousParenthesis(cursorPosition symbols.Position) option.Option[symbols.Position] {

	parentFound := false
	for {
		if cursorPosition.Character == 0 {
			break
		}

		cursorPosition.Character -= 1
		index := cursorPosition.IndexIn(s.Text)

		if parentFound {
			return option.Some(cursorPosition)
		}

		if rune(s.Text[index]) == '(' {
			//fmt.Println("Found at ", cursorPosition.Character)
			parentFound = true
		}
	}

	return option.None[symbols.Position]()
}

func tryToResolveFullModulePaths(wb *WordBuilder, unitModules *symbols_table.UnitModules, cursorPosition symbols.Position) *WordBuilder {
	if len(wb.word.modulePath) == 0 {
		return wb
	}

	paths := []string{}
	for _, m := range wb.word.modulePath {
		paths = append(paths, m.text)
	}
	moduleName := strings.Join(paths, "::")

	// Search if any of the imported modules matches this possible partial module path
	moduleInPosition := unitModules.FindContextModuleInCursorPosition(cursorPosition)
	if moduleInPosition != "" {
		module := unitModules.Get(moduleInPosition)
		for _, importedModule := range module.Imports {
			if strings.HasSuffix(importedModule, "::"+moduleName) {
				wb.WithResolvedModulePath(importedModule)
			}
		}
	}

	return wb
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
