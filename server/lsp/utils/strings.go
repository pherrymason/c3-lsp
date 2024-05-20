package utils

import "unicode"

func IsAZ09_(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

func IsNewLineSequence(r1, r2 rune) bool {
	return (r1 == '\r' && r2 == '\n') || (r1 == '\n')
}

func IsSpaceOrNewline(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r' || unicode.IsSpace(r)
}
