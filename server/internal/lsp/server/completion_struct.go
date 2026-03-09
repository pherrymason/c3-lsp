package server

import (
	"fmt"
	"strings"
	"unicode"

	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func extractStructFieldsFromData(data any) []string {
	if data == nil {
		return nil
	}

	dataMap, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	rawFields, ok := dataMap["structFields"]
	if !ok {
		return nil
	}

	fieldsAny, ok := rawFields.([]any)
	if ok {
		fields := make([]string, 0, len(fieldsAny))
		for _, field := range fieldsAny {
			name, ok := field.(string)
			if ok && name != "" {
				fields = append(fields, name)
			}
		}
		return fields
	}

	fieldsString, ok := rawFields.([]string)
	if ok {
		return fieldsString
	}

	return nil
}

func modulePathNameFromWord(word code.Word) string {
	if len(word.ResolvedModulePath()) > 0 {
		return strings.Join(word.ResolvedModulePath(), "::")
	}

	if !word.HasModulePath() {
		return ""
	}

	parts := make([]string, 0, len(word.ModulePath()))
	for _, token := range word.ModulePath() {
		parts = append(parts, token.Text())
	}

	return strings.Join(parts, "::")
}

func findStructFieldsInModule(state *l.ProjectState, moduleName string, label string) []string {
	if moduleName == "" {
		return nil
	}

	var result []string
	state.ForEachModule(func(module *symbols.Module) {
		if result != nil || module.GetName() != moduleName {
			return
		}
		for _, strukt := range module.Structs {
			if strukt.GetName() != label {
				continue
			}
			fields := make([]string, 0, len(strukt.GetMembers()))
			for _, member := range strukt.GetMembers() {
				fields = append(fields, member.GetName())
			}
			result = fields
			return
		}
	})
	return result
}

func structFieldsInScope(state *l.ProjectState, docURI protocol.DocumentUri, position symbols.Position, label string, explicitModuleName string) []string {
	unitModules := state.GetUnitModulesByDoc(docURI)
	if unitModules == nil {
		if explicitModuleName != "" {
			return findStructFieldsInModule(state, explicitModuleName, label)
		}

		return nil
	}

	currentModuleName := unitModules.FindContextModuleInCursorPosition(position)
	if currentModuleName == "" {
		if explicitModuleName != "" {
			return findStructFieldsInModule(state, explicitModuleName, label)
		}

		return nil
	}

	currentModule := unitModules.Get(currentModuleName)

	if explicitModuleName != "" {
		if fields := findStructFieldsInModule(state, explicitModuleName, label); len(fields) > 0 {
			return fields
		}

		for _, imported := range currentModule.Imports {
			if imported == explicitModuleName || strings.HasSuffix(imported, "::"+explicitModuleName) {
				if fields := findStructFieldsInModule(state, imported, label); len(fields) > 0 {
					return fields
				}
			}
		}
	}

	candidateModules := map[string]bool{currentModule.GetName(): true}
	for _, imported := range currentModule.Imports {
		candidateModules[imported] = true
	}

	var result []string
	state.ForEachModule(func(module *symbols.Module) {
		if result != nil || !candidateModules[module.GetName()] {
			return
		}
		for _, strukt := range module.Structs {
			if strukt.GetName() != label {
				continue
			}
			fields := make([]string, 0, len(strukt.GetMembers()))
			for _, member := range strukt.GetMembers() {
				fields = append(fields, member.GetName())
			}
			result = fields
			return
		}
	})

	return nil
}

func buildStructValueSnippet(fields []string, firstPlaceholder int) string {
	if len(fields) == 0 {
		return "{}"
	}

	if len(fields) == 1 {
		field := fields[0]
		return fmt.Sprintf("{ .%s = ${%d:%s} }", field, firstPlaceholder, escapeSnippetText(field))
	}

	lines := make([]string, 0, len(fields)+2)
	lines = append(lines, "{")
	for i, field := range fields {
		lines = append(lines, fmt.Sprintf("\t.%s = ${%d:%s},", field, firstPlaceholder+i, escapeSnippetText(field)))
	}
	lines = append(lines, "}")

	return strings.Join(lines, "\n")
}

func toLowerCamelName(typeName string) string {
	name := typeName
	if strings.Contains(name, "::") {
		parts := strings.Split(name, "::")
		name = parts[len(parts)-1]
	}

	runes := []rune(name)
	if len(runes) == 0 {
		return "value"
	}

	if len(runes) == 1 {
		return strings.ToLower(name)
	}

	if !unicode.IsUpper(runes[0]) {
		return name
	}

	upperRunEnd := 1
	for upperRunEnd < len(runes) && unicode.IsUpper(runes[upperRunEnd]) {
		upperRunEnd++
	}

	if upperRunEnd == 1 {
		runes[0] = unicode.ToLower(runes[0])
		return string(runes)
	}

	if upperRunEnd == len(runes) {
		return strings.ToLower(name)
	}

	prefixEnd := upperRunEnd - 1
	prefix := strings.ToLower(string(runes[:prefixEnd]))
	return prefix + string(runes[prefixEnd:])
}

func buildStructDeclarationSnippet(typeName string, fields []string) (string, bool) {
	if typeName == "" {
		return "", false
	}

	varName := toLowerCamelName(typeName)
	valueSnippet := buildStructValueSnippet(fields, 2)

	return fmt.Sprintf("%s ${1:%s} = %s;", typeName, escapeSnippetText(varName), valueSnippet), true
}

func completedStructTypeName(symbolInPositionText string, suggestionLabel string) string {
	if suggestionLabel == "" {
		return ""
	}

	if strings.Contains(symbolInPositionText, "::") {
		parts := strings.Split(symbolInPositionText, "::")
		if len(parts) > 1 {
			modulePath := strings.Join(parts[:len(parts)-1], "::")
			if modulePath != "" {
				return modulePath + "::" + suggestionLabel
			}
		}
	}

	return suggestionLabel
}

func buildStructSnippet(mode structCompletionMode, typeName string, fields []string) (string, bool) {
	switch mode {
	case structCompletionDeclaration:
		return buildStructDeclarationSnippet(typeName, fields)
	case structCompletionValue:
		return buildStructValueSnippet(fields, 1), true
	default:
		return "", false
	}
}
