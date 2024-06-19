package utils

import (
	"unicode"

	"github.com/pherrymason/c3-lsp/fs"
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

func NormalizePath(pathOrUri string) (string, error) {
	path, err := fs.UriToPath(pathOrUri)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse URI: %s", pathOrUri)
	}
	return fs.GetCanonicalPath(path), nil
}
