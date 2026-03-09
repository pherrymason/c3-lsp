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

var symbolPattern = regexp.MustCompile(`^[\$#@a-zA-Z0-9_]+$`)

func isSymbolRune(r rune) bool {
	return utils.IsAZ09_(r) || r == '@' || r == '$' || r == '#'
}

// Tries to find the symbol under cursor position
func (s SourceCode) SymbolInPosition(cursorPosition symbols.Position, docModules *symbols_table.UnitModules) Word {
	index := cursorPosition.IndexIn(s.Text)

	baseFound := false
	gettingAccess := false
	gettingModule := false
	ignoreSymbol := false
	insideAccessNesting := 0

	wb := NewWordBuilderE()
	var accessPath []Word
	var modulePath []Word

	for index >= 0 {

		limitsOpt := s.getWordIndexLimits(index, true)
		if limitsOpt.IsNone() {
			break
		}

		limits := limitsOpt.Get()
		symbol := s.Text[limits.start : limits.end+1]
		posRange := symbols.Range{
			Start: s.indexToPosition(limits.start),
			End:   s.indexToPosition(limits.end + 1),
		}

		// Just ignore content inside parenthesis
		if symbolPattern.MatchString(symbol) {
			if gettingAccess && insideAccessNesting == 0 /*!ignoreSymbol*/ {
				accessPath = append([]Word{{
					text:      symbol,
					textRange: posRange,
				}}, accessPath...)
			} else if gettingModule && insideAccessNesting == 0 /*!ignoreSymbol*/ {
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
				if symbol == "." || symbol == ":" || symbol == "(" || symbol == ")" {
					// Access operators and parentheses: set as base symbol
					// (needed for autocompletion when cursor is on '.' or ':')
					wb.WithText(symbol, posRange)
					baseFound = true
				} else {
					// Non-symbol, non-operator characters (like ';', ' ', '='):
					// skip backwards to find the actual symbol
					index--
					continue
				}
			}

			if symbol == "." {
				gettingAccess = true
			} else if symbol == ":" {
				gettingAccess = false
				gettingModule = true
			} else if gettingAccess && (symbol == "(" || symbol == "[") {
				insideAccessNesting--
				if insideAccessNesting < 0 {
					break
				}
				if insideAccessNesting == 0 {
					ignoreSymbol = false
				}
			} else if gettingAccess && (symbol == ")" || symbol == "]") {
				ignoreSymbol = true
				insideAccessNesting++
			} else if insideAccessNesting > 0 {

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
	for cursorPosition.Character != 0 {

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
	if index < 0 {
		return option.None[symbolLimits]()
	}

	if index >= len(s.Text) {
		return option.None[symbolLimits]()
	}

	for index > 0 && !utf8.RuneStart(s.Text[index]) {
		index--
	}

	r, size := utf8.DecodeRuneInString(s.Text[index:])
	if r == utf8.RuneError && size == 0 {
		return option.None[symbolLimits]()
	}

	if !isSymbolRune(r) {
		if returnAnyway {
			return option.Some(symbolLimits{index, index})
		} else {
			return option.None[symbolLimits]()
		}
	}

	symbolStart := index
	for current := index; current > 0; {
		prev := current - 1
		for prev > 0 && !utf8.RuneStart(s.Text[prev]) {
			prev--
		}

		pr, _ := utf8.DecodeRuneInString(s.Text[prev:])
		if !isSymbolRune(pr) {
			break
		}

		symbolStart = prev
		current = prev
	}

	symbolEndExclusive := index + size
	for symbolEndExclusive < len(s.Text) {
		nextRune, nextSize := utf8.DecodeRuneInString(s.Text[symbolEndExclusive:])
		if !isSymbolRune(nextRune) {
			break
		}
		symbolEndExclusive += nextSize
	}
	symbolEnd := symbolEndExclusive - 1

	if symbolStart < 0 || symbolStart >= len(s.Text) {
		return option.None[symbolLimits]()
	} else if symbolEnd < 0 || symbolEnd >= len(s.Text) {
		return option.None[symbolLimits]()
	} else if symbolStart > symbolEnd {
		return option.None[symbolLimits]()
	}

	return option.Some(symbolLimits{symbolStart, symbolEnd})
}

// OffsetToPosition converts a byte offset into a line/character Position.
func (d SourceCode) OffsetToPosition(index int) symbols.Position {
	return d.indexToPosition(index)
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
