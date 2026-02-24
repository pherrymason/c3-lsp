package server

import (
	"fmt"
	"strings"
	"unicode"

	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type completionItemLabelDetails struct {
	Detail      *string `json:"detail,omitempty"`
	Description *string `json:"description,omitempty"`
}

type completionItemWithLabelDetails struct {
	protocol.CompletionItem
	LabelDetails *completionItemLabelDetails `json:"labelDetails,omitempty"`
}

func completionKindDescription(kind *protocol.CompletionItemKind) *string {
	if kind == nil {
		return nil
	}

	switch *kind {
	case protocol.CompletionItemKindFunction:
		return cast.ToPtr("function")
	case protocol.CompletionItemKindMethod:
		return cast.ToPtr("method")
	case protocol.CompletionItemKindStruct:
		return cast.ToPtr("struct")
	case protocol.CompletionItemKindModule:
		return cast.ToPtr("module")
	case protocol.CompletionItemKindField:
		return cast.ToPtr("field")
	case protocol.CompletionItemKindVariable:
		return cast.ToPtr("variable")
	case protocol.CompletionItemKindKeyword:
		return cast.ToPtr("keyword")
	default:
		return nil
	}
}

func signatureDocumentation(detail string) protocol.MarkupContent {
	return protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: "```c3\n" + detail + "\n```",
	}
}

type structCompletionMode int

const (
	structCompletionNone structCompletionMode = iota
	structCompletionDeclaration
	structCompletionValue
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
	if strings.HasPrefix(name, "...") {
		name = strings.TrimPrefix(name, "...")
	}

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

	for _, parsedModulesByDoc := range state.GetAllUnitModules() {
		for _, module := range parsedModulesByDoc.Modules() {
			if module.GetName() != moduleName {
				continue
			}

			for _, strukt := range module.Structs {
				if strukt.GetName() != label {
					continue
				}

				fields := make([]string, 0, len(strukt.GetMembers()))
				for _, member := range strukt.GetMembers() {
					fields = append(fields, member.GetName())
				}

				return fields
			}
		}
	}

	return nil
}

func structFieldsInScope(state *l.ProjectState, docURI protocol.DocumentUri, position symbols.Position, label string, explicitModuleName string) []string {
	unitModules := state.GetUnitModulesByDoc(docURI)
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

	for _, parsedModulesByDoc := range state.GetAllUnitModules() {
		for _, module := range parsedModulesByDoc.Modules() {
			if !candidateModules[module.GetName()] {
				continue
			}

			for _, strukt := range module.Structs {
				if strukt.GetName() != label {
					continue
				}

				fields := make([]string, 0, len(strukt.GetMembers()))
				for _, member := range strukt.GetMembers() {
					fields = append(fields, member.GetName())
				}

				return fields
			}
		}
	}

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

func previousSignificantChar(text string, index int) rune {
	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		r := rune(text[i])
		if !unicode.IsSpace(r) {
			return r
		}
	}

	return 0
}

func nextSignificantChar(text string, index int) rune {
	if index < 0 {
		index = 0
	}

	for i := index; i < len(text); i++ {
		r := rune(text[i])
		if !unicode.IsSpace(r) {
			return r
		}
	}

	return 0
}

func nearestUnclosedDelimiter(text string, index int) (rune, int) {
	parenDepth := 0
	bracketDepth := 0

	if index > len(text) {
		index = len(text)
	}

	for i := index - 1; i >= 0; i-- {
		r := rune(text[i])
		if parenDepth == 0 && bracketDepth == 0 && (r == ';' || r == '{' || r == '}') {
			break
		}

		switch r {
		case ')':
			parenDepth++
		case ']':
			bracketDepth++
		case '(':
			if parenDepth > 0 {
				parenDepth--
			} else {
				return '(', i
			}
		case '[':
			if bracketDepth > 0 {
				bracketDepth--
			} else {
				return '[', i
			}
		}
	}

	return 0, -1
}

func previousWord(text string, index int) string {
	if index > len(text) {
		index = len(text)
	}

	i := index - 1
	for i >= 0 && unicode.IsSpace(rune(text[i])) {
		i--
	}

	end := i + 1
	for i >= 0 {
		r := rune(text[i])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			i--
			continue
		}
		break
	}

	if end <= i+1 {
		return ""
	}

	return text[i+1 : end]
}

func isControlHeaderKeyword(keyword string) bool {
	switch keyword {
	case "do", "for", "foreach", "foreach_r", "while", "if", "switch", "catch":
		return true
	default:
		return false
	}
}

func isFunctionOrMacroSignatureContext(text string, openParenIndex int) bool {
	if openParenIndex < 0 || openParenIndex > len(text) {
		return false
	}

	lineStart := strings.LastIndex(text[:openParenIndex], "\n") + 1
	prefix := strings.TrimSpace(text[lineStart:openParenIndex])

	return strings.HasPrefix(prefix, "fn ") || strings.HasPrefix(prefix, "macro ")
}

func structCompletionContext(text string, symbolStartIndex int, cursorIndex int) structCompletionMode {
	if symbolStartIndex < 0 {
		symbolStartIndex = 0
	}

	delimiter, delimiterIndex := nearestUnclosedDelimiter(text, cursorIndex)
	if delimiter == '(' {
		keyword := previousWord(text, delimiterIndex)
		if isFunctionOrMacroSignatureContext(text, delimiterIndex) {
			return structCompletionNone
		}

		if !isControlHeaderKeyword(keyword) {
			return structCompletionValue
		}
	}

	if delimiter == '[' {
		return structCompletionValue
	}

	prev := previousSignificantChar(text, symbolStartIndex)
	if prev == '=' || prev == '(' || prev == ',' || prev == '[' {
		return structCompletionValue
	}

	if prev != 0 {
		keyword := previousWord(text, symbolStartIndex)
		if keyword == "return" || keyword == "case" {
			return structCompletionValue
		}
	}

	if prev == 0 || prev == ';' || prev == '{' || prev == '}' {
		return structCompletionDeclaration
	}

	return structCompletionNone
}

func chooseTrailingToken(text string, cursorIndex int) string {
	next := nextSignificantChar(text, cursorIndex)
	if next == ';' || next == ',' || next == ')' || next == ']' || next == '}' {
		return ""
	}

	delimiter, delimiterIndex := nearestUnclosedDelimiter(text, cursorIndex)
	if delimiter == '(' {
		keyword := previousWord(text, delimiterIndex)
		if isControlHeaderKeyword(keyword) {
			return ""
		}

		return ","
	}

	if delimiter == '[' {
		return ","
	}

	return ";"
}

// Support "Completion"
// Returns: []CompletionItem | CompletionList | nil
func (h *Server) TextDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {

	cursorContext := ctx.BuildFromDocumentPosition(
		params.Position,
		utils.NormalizePath(params.TextDocument.URI),
		h.state,
	)

	suggestions := h.search.BuildCompletionList(
		cursorContext,
		h.state,
	)
	snippetSupport := clientSupportsCompletionSnippets(h.clientCapabilities)

	doc := h.state.GetDocument(cursorContext.DocURI)
	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	symbolInPosition := doc.SourceCode.SymbolInPosition(
		cursorContext.Position.RewindCharacter(),
		unitModules,
	)
	symbolStartIndex := symbolInPosition.FullTextRange().Start.IndexIn(doc.SourceCode.Text)
	replaceRange := symbolInPosition.FullTextRange().ToLSP()
	cursorIndex := cursorContext.Position.IndexIn(doc.SourceCode.Text)
	explicitModuleName := modulePathNameFromWord(symbolInPosition)

	// Send labelDetails as an extension field so clients that support richer
	// completion rendering (such as Zed) can show the signature beside symbols.
	items := make([]completionItemWithLabelDetails, 0, len(suggestions))
	for _, suggestion := range suggestions {
		item := completionItemWithLabelDetails{CompletionItem: suggestion}
		symbolInPositionText := symbolInPosition.GetFullQualifiedName()

		if suggestion.Detail != nil {
			item.LabelDetails = &completionItemLabelDetails{
				Description: cast.ToPtr(" " + *suggestion.Detail),
				Detail:      completionKindDescription(suggestion.Kind),
			}

			// Keep classic detail for clients that don't render labelDetails.
		}

		if suggestion.Detail != nil && item.Documentation == nil {
			if suggestion.Kind != nil && (*suggestion.Kind == protocol.CompletionItemKindFunction || *suggestion.Kind == protocol.CompletionItemKindMethod || *suggestion.Kind == protocol.CompletionItemKindStruct) {
				doc := signatureDocumentation(*suggestion.Detail)
				item.Documentation = doc
			}
		}

		isCallable := false
		isStructSnippet := false
		structMode := structCompletionNone

		if suggestion.Kind != nil && suggestion.Detail != nil {
			if *suggestion.Kind == protocol.CompletionItemKindFunction || *suggestion.Kind == protocol.CompletionItemKindMethod {
				if snippet, ok := buildCallableSnippet(suggestion.Label, *suggestion.Detail); ok {
					isCallable = true
					insertText := snippet
					if !snippetSupport {
						insertText = snippetToPlainInsertText(snippet)
					}

					if textEdit, ok := item.TextEdit.(protocol.TextEdit); ok {
						textEdit.NewText = insertText
						item.TextEdit = textEdit
					} else {
						item.InsertText = cast.ToPtr(insertText)
					}

					if snippetSupport {
						snippetFormat := protocol.InsertTextFormatSnippet
						item.InsertTextFormat = &snippetFormat
					}
				}
			}
		}

		if suggestion.Kind != nil && *suggestion.Kind == protocol.CompletionItemKindStruct {
			structMode = structCompletionContext(doc.SourceCode.Text, symbolStartIndex, cursorIndex)
			if structMode != structCompletionNone {
				fields := extractStructFieldsFromData(suggestion.Data)
				if len(fields) == 0 {
					fields = structFieldsInScope(h.state, doc.URI, cursorContext.Position, suggestion.Label, explicitModuleName)
				}

				structTypeName := completedStructTypeName(symbolInPositionText, suggestion.Label)

				if snippet, ok := buildStructSnippet(structMode, structTypeName, fields); ok {
					isStructSnippet = true
					insertText := snippet
					if !snippetSupport {
						insertText = snippetToPlainInsertText(snippet)
					}

					item.TextEdit = protocol.TextEdit{
						NewText: insertText,
						Range:   replaceRange,
					}
					item.InsertText = nil

					if snippetSupport {
						snippetFormat := protocol.InsertTextFormatSnippet
						item.InsertTextFormat = &snippetFormat
					}
				}
			}
		}

		shouldAddTrailing := isCallable || (isStructSnippet && structMode == structCompletionValue)
		if shouldAddTrailing {
			trailing := chooseTrailingToken(doc.SourceCode.Text, cursorIndex)
			if trailing != "" {
				if textEdit, ok := item.TextEdit.(protocol.TextEdit); ok {
					if !strings.HasSuffix(textEdit.NewText, trailing) {
						textEdit.NewText += trailing
						item.TextEdit = textEdit
					}
				} else if item.InsertText != nil {
					if !strings.HasSuffix(*item.InsertText, trailing) {
						item.InsertText = cast.ToPtr(*item.InsertText + trailing)
					}
				} else {
					item.InsertText = cast.ToPtr(item.Label + trailing)
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}
