package server

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func clientSupportsCompletionSnippets(capabilities protocol.ClientCapabilities) bool {
	if capabilities.TextDocument == nil || capabilities.TextDocument.Completion == nil {
		return false
	}

	completionItem := capabilities.TextDocument.Completion.CompletionItem
	if completionItem == nil || completionItem.SnippetSupport == nil {
		return false
	}

	return *completionItem.SnippetSupport
}

func buildCallableSnippet(label string, detail string) (string, bool) {
	if dot := strings.LastIndex(label, "."); dot >= 0 && dot+1 < len(label) {
		label = label[dot+1:]
	}

	end := strings.LastIndex(detail, ")")
	if end == -1 {
		return "", false
	}

	start := strings.LastIndex(detail[:end], "(")
	if start == -1 || start > end {
		return "", false
	}

	args := detail[start+1 : end]
	if i := strings.Index(args, ";"); i >= 0 {
		args = args[:i]
	}

	parts := splitArgs(args)
	required := []string{}
	for _, part := range parts {
		arg := strings.TrimSpace(part)
		if arg == "" {
			continue
		}

		if strings.Contains(arg, "=") {
			continue
		}

		if strings.Contains(arg, "...") {
			continue
		}

		argName := extractArgName(arg, len(required)+1)
		if argName == "self" {
			continue
		}

		required = append(required, argName)
	}

	if len(required) == 0 {
		return label + "()", true
	}

	placeholders := make([]string, 0, len(required))
	for i, arg := range required {
		placeholders = append(placeholders, fmt.Sprintf("${%d:%s}", i+1, escapeSnippetText(arg)))
	}

	return label + "(" + strings.Join(placeholders, ", ") + ")", true
}

func splitArgs(args string) []string {
	parts := []string{}
	current := strings.Builder{}
	parenDepth := 0
	angleDepth := 0
	bracketDepth := 0

	for _, r := range args {
		switch r {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '<':
			angleDepth++
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		}

		if r == ',' && parenDepth == 0 && angleDepth == 0 && bracketDepth == 0 {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}

		current.WriteRune(r)
	}

	parts = append(parts, current.String())
	return parts
}

func extractArgName(arg string, fallback int) string {
	fields := strings.Fields(arg)
	if len(fields) <= 1 {
		return fmt.Sprintf("arg%d", fallback)
	}

	name := strings.TrimPrefix(fields[len(fields)-1], "&")
	name = strings.TrimPrefix(name, "...")

	if name == "" {
		return fmt.Sprintf("arg%d", fallback)
	}

	return name
}

func escapeSnippetText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "$", "\\$")
	value = strings.ReplaceAll(value, "}", "\\}")
	return value
}

func snippetToPlainInsertText(snippet string) string {
	result := strings.Builder{}

	for i := 0; i < len(snippet); i++ {
		if i+2 < len(snippet) && snippet[i] == '$' && snippet[i+1] == '{' {
			j := i + 2
			for j < len(snippet) && snippet[j] >= '0' && snippet[j] <= '9' {
				j++
			}

			if j < len(snippet) && snippet[j] == ':' {
				j++
				start := j
				for j < len(snippet) && snippet[j] != '}' {
					j++
				}

				if j < len(snippet) && snippet[j] == '}' {
					result.WriteString(snippet[start:j])
					i = j
					continue
				}
			}
		}

		result.WriteByte(snippet[i])
	}

	return result.String()
}
