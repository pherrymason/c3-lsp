package utils

import (
	"fmt"
	"strconv"
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

func NormalizePath(pathOrUri string) string {
	path, err := fs.UriToPath(pathOrUri)
	if err != nil {
		panic(errors.Wrapf(err, "unable to parse URI: %s", pathOrUri))
	}
	return fs.GetCanonicalPath(path)
}

func StringToUint(value string) uint {
	uint64Value, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		panic(fmt.Sprintf("Could not convert string to uint: %s", value))
	}

	// Convertir uint64 a uint (sólo si está dentro del rango de uint)
	return uint(uint64Value)
}
