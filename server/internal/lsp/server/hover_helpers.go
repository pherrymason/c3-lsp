package server

import (
	"reflect"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func extractDesignatorMemberAt(source string, cursorIndex int) (string, int, bool) {
	if cursorIndex < 0 || cursorIndex > len(source) {
		return "", 0, false
	}
	if len(source) == 0 {
		return "", 0, false
	}

	start := cursorIndex
	if start == len(source) {
		start--
	}
	if start < len(source) && !isIdentByte(source[start]) {
		if source[start] == '.' && start+1 < len(source) && isIdentByte(source[start+1]) {
			start++
		} else if start > 0 && isIdentByte(source[start-1]) {
			start--
		} else {
			for start < len(source) && !isIdentByte(source[start]) {
				if source[start] == '\n' || source[start] == ';' {
					return "", 0, false
				}
				start++
			}
			if start >= len(source) || !isIdentByte(source[start]) {
				return "", 0, false
			}
		}
	}
	for start > 0 && isIdentByte(source[start-1]) {
		start--
	}
	end := start
	for end < len(source) && isIdentByte(source[end]) {
		end++
	}
	if start >= end {
		return "", 0, false
	}

	member := source[start:end]
	if member == "" {
		return "", 0, false
	}

	idx := start - 1
	for idx >= 0 && (source[idx] == ' ' || source[idx] == '\t') {
		idx--
	}
	if idx < 0 || source[idx] != '.' {
		return "", 0, false
	}

	return member, idx, true
}

func extractQualifiedSymbolAt(source string, cursorIndex int) (string, string, bool) {
	if len(source) == 0 || cursorIndex < 0 || cursorIndex > len(source) {
		return "", "", false
	}

	idx := cursorIndex
	if idx == len(source) {
		idx--
	}
	if idx < 0 {
		return "", "", false
	}

	if !isIdentByte(source[idx]) {
		if idx > 0 && isIdentByte(source[idx-1]) {
			idx--
		} else {
			return "", "", false
		}
	}

	start := idx
	for start > 0 && isIdentByte(source[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(source) && isIdentByte(source[end]) {
		end++
	}

	left := start
	for left > 0 {
		if source[left-1] != ':' {
			break
		}
		if left-2 < 0 || source[left-2] != ':' {
			break
		}
		left -= 2
		for left > 0 && isIdentByte(source[left-1]) {
			left--
		}
	}

	chain := source[left:end]
	parts := strings.Split(chain, "::")
	if len(parts) < 2 {
		return "", "", false
	}
	for _, p := range parts {
		if p == "" {
			return "", "", false
		}
	}

	modulePath := strings.Join(parts[:len(parts)-1], "::")
	symbolName := parts[len(parts)-1]
	return modulePath, symbolName, true
}

func extractModuleTokenAt(source string, cursorIndex int) (string, bool) {
	if len(source) == 0 || cursorIndex < 0 || cursorIndex > len(source) {
		return "", false
	}
	if cursorIndex < len(source) && !isIdentByte(source[cursorIndex]) {
		return "", false
	}

	idx := cursorIndex
	if idx == len(source) {
		idx--
	}
	if idx < 0 {
		return "", false
	}

	if !isIdentByte(source[idx]) {
		return "", false
	}

	start := idx
	for start > 0 && isIdentByte(source[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(source) && isIdentByte(source[end]) {
		end++
	}
	if start >= end {
		return "", false
	}

	hasRightSeparator := end+1 < len(source) && source[end] == ':' && source[end+1] == ':'
	if !hasRightSeparator {
		return "", false
	}

	return source[start:end], true
}

func extractIdentifierTokenAt(source string, cursorIndex int) (string, bool) {
	if len(source) == 0 || cursorIndex < 0 || cursorIndex > len(source) {
		return "", false
	}

	idx := cursorIndex
	if idx == len(source) {
		idx--
	}
	if idx < 0 {
		return "", false
	}

	if !isIdentByte(source[idx]) {
		if idx > 0 && isIdentByte(source[idx-1]) {
			idx--
		} else {
			return "", false
		}
	}

	start := idx
	for start > 0 && isIdentByte(source[start-1]) {
		start--
	}
	end := idx + 1
	for end < len(source) && isIdentByte(source[end]) {
		end++
	}

	if start >= end {
		return "", false
	}

	return source[start:end], true
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func parseLambdaParamNames(paramsText string) []string {
	parts := strings.Split(paramsText, ",")
	params := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		name := fields[len(fields)-1]
		name = strings.TrimPrefix(name, "&")
		name = strings.TrimSuffix(name, ",")
		params = append(params, name)
	}

	return params
}

func isTypeIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func isNilIndexable(symbol symbols.Indexable) bool {
	if symbol == nil {
		return true
	}

	v := reflect.ValueOf(symbol)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
