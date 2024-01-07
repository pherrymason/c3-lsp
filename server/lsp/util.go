package lsp

import "unicode"

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func wordInPosition(text string, position int) string {
	wordStart := 0
	for i := position; i >= 0; i-- {
		if !(unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_') {
			wordStart = i + 1
			break
		}
	}

	wordEnd := len(text) - 1
	for i := position; i < len(text); i++ {
		if !(unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_') {
			wordEnd = i - 1
			break
		}
	}

	word := text[wordStart : wordEnd+1]
	return word
}
