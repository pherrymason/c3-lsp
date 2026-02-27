package symbols

import (
	"strings"
	"unicode"
)

func ModuleGenericConstraintLines(module *Module) []string {
	if module == nil || len(module.GenericParameters) == 0 {
		return nil
	}

	doc := module.GetDocComment()
	if doc == nil {
		return nil
	}

	lines := []string{}
	for _, contract := range doc.GetContracts() {
		if contract.GetName() != "@require" {
			continue
		}

		body := strings.TrimSpace(contract.GetBody())
		if body == "" {
			continue
		}

		if !containsAnyGenericIdentifier(body, module.GenericParameters) {
			continue
		}

		lines = append(lines, "**"+contract.Name+"** "+body)
	}

	return lines
}

func ModuleGenericConstraintMarkdown(module *Module) string {
	lines := ModuleGenericConstraintLines(module)
	if len(lines) == 0 {
		return ""
	}

	return "**Generic constraints:**\n\n" + strings.Join(lines, "\n")
}

func containsAnyGenericIdentifier(text string, genericParameters map[string]*GenericParameter) bool {
	for generic := range genericParameters {
		if containsIdentifier(text, generic) {
			return true
		}
	}

	return false
}

func containsIdentifier(text string, ident string) bool {
	if ident == "" {
		return false
	}

	for i := 0; i <= len(text)-len(ident); i++ {
		if text[i:i+len(ident)] != ident {
			continue
		}

		if i > 0 && isIdentifierRune(rune(text[i-1])) {
			continue
		}

		j := i + len(ident)
		if j < len(text) && isIdentifierRune(rune(text[j])) {
			continue
		}

		return true
	}

	return false
}

func isIdentifierRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
