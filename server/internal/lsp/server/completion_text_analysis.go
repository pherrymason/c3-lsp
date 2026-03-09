package server

import (
	"strings"
	"unicode"
)

func previousSignificantChar(text string, index int) rune {
	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		r := rune(text[i])
		if !unicode.IsSpace(r) {
			return r
		}
	}

	return 0
}

func nextSignificantChar(text string, index int) rune {
	if index < 0 {
		index = 0
	}

	for i := index; i < len(text); i++ {
		r := rune(text[i])
		if !unicode.IsSpace(r) {
			return r
		}
	}

	return 0
}

func nearestUnclosedDelimiter(text string, index int) (rune, int) {
	parenDepth := 0
	bracketDepth := 0

	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		r := rune(text[i])
		if parenDepth == 0 && bracketDepth == 0 && (r == ';' || r == '{' || r == '}') {
			break
		}

		switch r {
		case ')':
			parenDepth++
		case ']':
			bracketDepth++
		case '(':
			if parenDepth > 0 {
				parenDepth--
			} else {
				return '(', i
			}
		case '[':
			if bracketDepth > 0 {
				bracketDepth--
			} else {
				return '[', i
			}
		}
	}

	return 0, -1
}

func nearestUnclosedCurly(text string, index int) int {
	curlyDepth := 0

	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		r := rune(text[i])
		switch r {
		case '}':
			curlyDepth++
		case '{':
			if curlyDepth > 0 {
				curlyDepth--
			} else {
				return i
			}
		}
	}

	return -1
}

func previousNonSpaceIndex(text string, index int) int {
	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		if !unicode.IsSpace(rune(text[i])) {
			return i
		}
	}

	return -1
}

func isTypeArgumentContext(text string, index int) bool {
	openCurly := nearestUnclosedCurly(text, index)
	if openCurly < 0 {
		return false
	}

	prevIdx := previousNonSpaceIndex(text, openCurly)
	if prevIdx < 0 {
		return false
	}

	prev := rune(text[prevIdx])
	if !unicode.IsLetter(prev) && !unicode.IsDigit(prev) && prev != '_' && prev != ':' && prev != ']' && prev != '}' && prev != '*' {
		return false
	}

	keyword := previousWord(text, openCurly)
	return !isBlockHeaderKeyword(keyword)
}

func previousWord(text string, index int) string {
	if index > len(text) {
		index = len(text)
	}

	i := index - 1
	for i >= 0 && unicode.IsSpace(rune(text[i])) {
		i--
	}

	end := i + 1
	for i >= 0 {
		r := rune(text[i])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			i--
			continue
		}
		break
	}

	if end <= i+1 {
		return ""
	}

	return text[i+1 : end]
}

func isControlHeaderKeyword(keyword string) bool {
	switch keyword {
	case "do", "for", "foreach", "foreach_r", "while", "if", "switch", "catch":
		return true
	default:
		return false
	}
}

func isBlockHeaderKeyword(keyword string) bool {
	switch keyword {
	case "fn", "macro", "if", "else", "for", "foreach", "foreach_r", "while", "switch", "case", "catch", "do", "struct", "union", "enum", "interface", "faultdef":
		return true
	default:
		return false
	}
}

func isFunctionOrMacroSignatureContext(text string, openParenIndex int) bool {
	if openParenIndex < 0 || openParenIndex > len(text) {
		return false
	}

	lineStart := strings.LastIndex(text[:openParenIndex], "\n") + 1
	prefix := strings.TrimSpace(text[lineStart:openParenIndex])

	return strings.HasPrefix(prefix, "fn ") || strings.HasPrefix(prefix, "macro ")
}

func structCompletionContext(text string, symbolStartIndex int, cursorIndex int) structCompletionMode {
	if symbolStartIndex < 0 {
		symbolStartIndex = 0
	}

	if isTypeArgumentContext(text, symbolStartIndex) {
		return structCompletionNone
	}

	delimiter, delimiterIndex := nearestUnclosedDelimiter(text, cursorIndex)
	if delimiter == '(' {
		keyword := previousWord(text, delimiterIndex)
		if isFunctionOrMacroSignatureContext(text, delimiterIndex) {
			return structCompletionNone
		}

		if !isControlHeaderKeyword(keyword) {
			return structCompletionValue
		}
	}

	if delimiter == '[' {
		return structCompletionValue
	}

	prev := previousSignificantChar(text, symbolStartIndex)
	if prev == '=' || prev == '(' || prev == ',' || prev == '[' {
		return structCompletionValue
	}

	if prev != 0 {
		keyword := previousWord(text, symbolStartIndex)
		if keyword == "return" || keyword == "case" {
			return structCompletionValue
		}
	}

	if prev == 0 || prev == ';' || prev == '{' || prev == '}' {
		return structCompletionDeclaration
	}

	return structCompletionNone
}

func chooseTrailingToken(text string, cursorIndex int) string {
	next := nextSignificantChar(text, cursorIndex)
	if next == ';' || next == ',' || next == ')' || next == ']' || next == '}' {
		return ""
	}

	delimiter, delimiterIndex := nearestUnclosedDelimiter(text, cursorIndex)
	if delimiter == '(' {
		keyword := previousWord(text, delimiterIndex)
		if isControlHeaderKeyword(keyword) {
			return ""
		}

		return ","
	}

	if delimiter == '[' {
		return ","
	}

	return ";"
}
