package symbols

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type Enumerator struct {
	value            string
	AssociatedValues []Variable
	EnumName         string
	BaseIndexable
}

func NewEnumerator(name string, value string, associatedValues []Variable, enumName string, module string, idRange Range, docId string) *Enumerator {
	enumerator := &Enumerator{
		value:            value,
		AssociatedValues: associatedValues,
		EnumName:         enumName,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			idRange,
			NewRange(0, 0, 0, 0),
			protocol.CompletionItemKindEnumMember,
		),
	}

	for _, av := range associatedValues {
		enumerator.InsertNestedScope(&av)
	}

	return enumerator
}

func (e *Enumerator) GetEnumName() string {
	return e.EnumName
}

func (e *Enumerator) GetEnumFQN() string {
	return fmt.Sprintf("%s::%s", e.GetModule().GetName(), e.GetEnumName())
}

func (e Enumerator) GetHoverInfo() string {
	value := strings.TrimSpace(e.value)
	members := renderAssociatedMembers(e.AssociatedValues, value)
	prefix := e.Name
	if strings.TrimSpace(e.EnumName) != "" {
		prefix = fmt.Sprintf("enum %s.%s", e.EnumName, e.Name)
	}

	if members != "" {
		if value != "" && !looksLikeAssociatedValuesLiteral(value) {
			return fmt.Sprintf("%s = %s %s", prefix, value, members)
		}
		return fmt.Sprintf("%s %s", prefix, members)
	}

	if value != "" {
		return fmt.Sprintf("%s = %s", prefix, value)
	}

	return prefix
}

func (e Enumerator) GetCompletionDetail() string {
	return "Enum Value"
}

func renderAssociatedMembers(associated []Variable, value string) string {
	if len(associated) == 0 {
		return ""
	}

	values := parseAssociatedValues(value)
	parts := make([]string, 0, len(associated))
	for i, member := range associated {
		memberType := strings.TrimSpace(member.GetType().String())
		memberName := strings.TrimSpace(member.GetName())
		memberValue := "value"
		if i < len(values) && strings.TrimSpace(values[i]) != "" {
			memberValue = strings.TrimSpace(values[i])
		}

		prefix := memberName
		if memberType != "" {
			prefix = memberType + " " + memberName
		}
		parts = append(parts, fmt.Sprintf("%s: %s", prefix, memberValue))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

func looksLikeAssociatedValuesLiteral(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")
}

func parseAssociatedValues(value string) []string {
	trimmed := strings.TrimSpace(value)
	if !looksLikeAssociatedValuesLiteral(trimmed) {
		return nil
	}

	inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if inner == "" {
		return nil
	}

	parts := []string{}
	start := 0
	depthParen := 0
	depthBrace := 0
	inString := false
	escaped := false

	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case ',':
			if depthParen == 0 && depthBrace == 0 {
				parts = append(parts, strings.TrimSpace(inner[start:i]))
				start = i + 1
			}
		}
	}

	parts = append(parts, strings.TrimSpace(inner[start:]))
	return parts
}
