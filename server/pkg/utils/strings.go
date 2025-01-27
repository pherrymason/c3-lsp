package utils

import (
	"strings"
	"unicode"

	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pkg/errors"
)

func IsAZ09_(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$'
}

func IsNewLineSequence(r1, r2 rune) bool {
	return (r1 == '\r' && r2 == '\n') || (r1 == '\n')
}

func IsSpaceOrNewline(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r' || unicode.IsSpace(r)
}

// Returns the { line, col } of the first occurrence of a substring inside a larger string.
// Assumes the substring has no newlines.
// Returns { 0, 0 } if the substring was not found.
//
// Note that the line number is one-indexed, so the first line is at line 1.
// However, columns are zero-indexed, that is, the first rune is at column 0.
func FindLineColOfSubstring(s string, substring string) (uint, uint) {
	var line uint = 0
	var col uint = 0

	for i, lineContents := range strings.Split(s, "\n") {
		offset := strings.Index(lineContents, substring)
		if offset >= 0 {
			line = uint(i) + 1
			col = uint(offset)
			break
		}
	}

	return line, col
}

func NormalizePath(pathOrUri string) string {
	path, err := fs.UriToPath(pathOrUri)
	if err != nil {
		panic(errors.Wrapf(err, "unable to parse URI: %s", pathOrUri))
	}
	return fs.GetCanonicalPath(path)
}
