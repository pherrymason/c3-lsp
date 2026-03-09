package symbols

import (
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"
)

// NormalizeModuleName converts an arbitrary file/module name into a valid C3
// module identifier (lowercase alphanumeric + underscores, max 31 chars).
// When the input contains path separators a short FNV hash is appended to
// preserve uniqueness across different directories.
func NormalizeModuleName(input string) string {
	const maxLength = 31
	var modifiedName []rune
	original := input

	input = strings.TrimSuffix(input, ".c3")

	for _, char := range input {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			if unicode.IsLower(char) {
				modifiedName = append(modifiedName, char)
			} else {
				modifiedName = append(modifiedName, unicode.ToLower(char))
			}
		} else {
			modifiedName = append(modifiedName, '_')
		}

		if len(modifiedName) >= maxLength {
			break
		}
	}

	base := string(modifiedName)
	if base == "" {
		base = "anonymous"
	}
	if !strings.ContainsAny(original, `/\\:`) {
		return base
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(original))
	suffix := fmt.Sprintf("_%08x", hasher.Sum32())

	baseRunes := []rune(base)
	suffixRunes := []rune(suffix)
	if len(baseRunes)+len(suffixRunes) > maxLength {
		baseRunes = baseRunes[:maxLength-len(suffixRunes)]
	}

	return string(baseRunes) + suffix
}
