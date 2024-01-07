package lsp

import (
	"errors"
	"unicode"
)

func boolPtr(v bool) *bool {
	b := v
	return &b
}

func wordInPosition(text string, position int) (string, error) {
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

	if wordStart > len(text) {
		return "", errors.New("wordStart out of bounds")
	} else if wordEnd > len(text) {
		return "", errors.New("wordEnd out of bounds")
	} else if wordStart > wordEnd {
		return "", errors.New("wordStart > wordEnd!")
	}

	word := text[wordStart : wordEnd+1]
	return word, nil
}
