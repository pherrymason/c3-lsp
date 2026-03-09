package search

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/context"
	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	protocol_utils "github.com/pherrymason/c3-lsp/internal/lsp/protocol"
	sp "github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/c3"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

type completionScopeContext struct {
	currentModule       *symbols.Module
	currentFunction     *symbols.Function
	importedModuleNames map[string]struct{}
}

func buildCompletionScopeContext(state *l.ProjectState, doc *document.Document, cursorPosition symbols.Position) completionScopeContext {
	ctx := completionScopeContext{importedModuleNames: map[string]struct{}{}}
	if doc == nil {
		return ctx
	}

	unitModules := state.GetUnitModulesByDoc(doc.URI)
	for _, module := range unitModules.Modules() {
		if !module.GetDocumentRange().HasPosition(cursorPosition) {
			continue
		}

		ctx.currentModule = module
		for _, imported := range module.Imports {
			ctx.importedModuleNames[imported] = struct{}{}
		}

		for _, function := range module.ChildrenFunctions {
			if function.GetDocumentRange().HasPosition(cursorPosition) {
				ctx.currentFunction = function
				break
			}
		}
		break
	}

	return ctx
}

func importedModuleItems(state *l.ProjectState, scopeCtx completionScopeContext, prefix string) []protocol.CompletionItem {
	if scopeCtx.currentModule == nil || len(scopeCtx.importedModuleNames) == 0 {
		return []protocol.CompletionItem{}
	}

	snapshot := state.Snapshot()

	items := []protocol.CompletionItem{}
	for imported := range scopeCtx.importedModuleNames {
		if prefix != "" && !strings.HasPrefix(imported, prefix) {
			continue
		}

		moduleItem := protocol.CompletionItem{
			Label:  imported,
			Kind:   cast.ToPtr(protocol.CompletionItemKindModule),
			Detail: cast.ToPtr("Module"),
		}

		// Use O(1) index lookup instead of O(D×M) scan.
		if snapshot != nil {
			if mods := snapshot.ModulesByName(imported); len(mods) > 0 {
				moduleItem.Documentation = GetCompletableDocComment(mods[0])
				moduleItem.Detail = GetCompletionDetail(mods[0])
			}
		}

		items = append(items, moduleItem)
	}

	return items
}

func completionScopeRank(item protocol.CompletionItem, scopeCtx completionScopeContext, hasExplicitModulePath bool) int {
	if item.Kind != nil && *item.Kind == protocol.CompletionItemKindKeyword {
		if strings.HasPrefix(item.Label, "$") {
			return 99
		}
		return 90
	}

	if item.Kind != nil && *item.Kind == protocol.CompletionItemKindModule {
		if hasExplicitModulePath {
			return 5
		}
		if _, ok := scopeCtx.importedModuleNames[item.Label]; ok {
			return 15
		}
		return 40
	}

	if hasExplicitModulePath {
		return 25
	}

	if scopeCtx.currentModule == nil {
		return 45
	}

	if scopeCtx.currentFunction != nil && item.Kind != nil && *item.Kind == protocol.CompletionItemKindVariable {
		for _, variable := range scopeCtx.currentFunction.Variables {
			if variable.GetName() != item.Label {
				continue
			}
			if detail := GetCompletionDetail(variable); detail != nil && item.Detail != nil && *detail == *item.Detail {
				return 0
			}
		}
	}

	return 20
}

func completionSortText(item protocol.CompletionItem, scopeCtx completionScopeContext, hasExplicitModulePath bool) string {
	rank := completionScopeRank(item, scopeCtx, hasExplicitModulePath)
	return fmt.Sprintf("%02d_%s", rank, strings.ToLower(item.Label))
}

func lineHasOnlyWhitespaceBeforeCursor(text string, cursorPosition symbols.Position) bool {
	cursorIndex := cursorPosition.IndexIn(text)
	if cursorIndex < 0 || cursorIndex > len(text) {
		return false
	}

	lineStart := cursorIndex
	for lineStart > 0 && text[lineStart-1] != '\n' {
		lineStart--
	}

	for i := lineStart; i < cursorIndex; i++ {
		if text[i] != ' ' && text[i] != '\t' && text[i] != '\r' {
			return false
		}
	}

	return true
}

func hasCompletionContextAtCursor(symbolInPosition sourcecode.Word, cursorPosition symbols.Position, text string) bool {
	if symbolInPosition.Text() == "" {
		return false
	}

	rangeAtCursor := symbolInPosition.FullTextRange()

	if symbolInPosition.IsSeparator() &&
		rangeAtCursor.End.Line+1 == cursorPosition.Line &&
		lineHasOnlyWhitespaceBeforeCursor(text, cursorPosition) {
		return true
	}

	if rangeAtCursor.Start.Line != cursorPosition.Line || rangeAtCursor.End.Line != cursorPosition.Line {
		return false
	}

	if cursorPosition.Character+1 < rangeAtCursor.Start.Character {
		return false
	}

	if cursorPosition.Character > rangeAtCursor.End.Character+2 {
		return false
	}

	if symbolInPosition.IsSeparator() || symbolInPosition.HasAccessPath() || symbolInPosition.HasModulePath() {
		return true
	}

	return true
}

func completionPrefix(symbolInPosition sourcecode.Word, hasContextAtCursor bool) string {
	if !hasContextAtCursor || symbolInPosition.IsSeparator() {
		return ""
	}

	return symbolInPosition.Text()
}

func inferVariableTypeNameFromText(doc *document.Document, cursorPosition symbols.Position, variableName string) option.Option[string] {
	if variableName == "" {
		return option.None[string]()
	}

	lines := strings.Split(doc.SourceCode.Text, "\n")
	if len(lines) == 0 {
		return option.None[string]()
	}

	lineLimit := int(cursorPosition.Line)
	if lineLimit >= len(lines) {
		lineLimit = len(lines) - 1
	}

	namePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(variableName) + `\b`)
	baseTypePattern := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)?)\s*(?:\{|\[|\*|$)`)

	for line := lineLimit; line >= 0; line-- {
		content := lines[line]
		matches := namePattern.FindAllStringIndex(content, -1)
		for _, match := range matches {
			next := match[1]
			for next < len(content) && (content[next] == ' ' || content[next] == '\t') {
				next++
			}

			if next >= len(content) || (content[next] != ';' && content[next] != '=') {
				continue
			}

			prefix := strings.TrimSpace(content[:match[0]])
			if prefix == "" {
				continue
			}

			baseMatches := baseTypePattern.FindAllStringSubmatch(prefix, -1)
			if len(baseMatches) == 0 {
				continue
			}

			base := baseMatches[len(baseMatches)-1][1]
			if base != "" {
				return option.Some(base)
			}
		}
	}

	return option.None[string]()
}

func completionSymbolAtCursor(doc *document.Document, state *l.ProjectState, cursorPosition symbols.Position) sourcecode.Word {
	unitModules := state.GetUnitModulesByDoc(doc.URI)
	atCursor, atCursorOk := symbolInPositionSafe(doc, unitModules, cursorPosition)
	rewoundPos := doc.RewindPosition(cursorPosition)
	rewound, rewoundOk := symbolInPositionSafe(doc, unitModules, rewoundPos)

	if !atCursorOk {
		return rewound
	}

	if !rewoundOk {
		return atCursor
	}

	isChainLike := func(w sourcecode.Word) bool {
		return w.IsSeparator() || w.HasAccessPath() || w.HasModulePath()
	}

	isIdentifierLike := func(w sourcecode.Word) bool {
		if w.Text() == "" {
			return false
		}

		return utils.IsAZ09_(rune(w.Text()[0]))
	}

	if rewound.IsSeparator() &&
		rewound.HasAccessPath() &&
		atCursor.HasAccessPath() &&
		cursorPosition.Line == atCursor.TextRange().Start.Line &&
		cursorPosition.Character == atCursor.TextRange().Start.Character {
		return rewound
	}

	if isChainLike(atCursor) {
		return atCursor
	}

	if isChainLike(rewound) {
		return rewound
	}

	if isIdentifierLike(atCursor) {
		return atCursor
	}

	if isIdentifierLike(rewound) {
		return rewound
	}

	return rewound
}

func symbolInPositionSafe(doc *document.Document, unitModules *symbols_table.UnitModules, cursorPosition symbols.Position) (word sourcecode.Word, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	word = doc.SourceCode.SymbolInPosition(cursorPosition, unitModules)
	return word, true
}

func isCompletingAModulePath(doc *document.Document, cursorPosition symbols.Position) (bool, string) {
	// Cursor is just right after last char, let's rewind one place
	position := cursorPosition
	if cursorPosition.Character > 0 {
		position = doc.RewindPosition(cursorPosition)
	}

	index := position.IndexIn(doc.SourceCode.Text)

	// Read backwards until a separator character is found.
	startIndex := index
	for i := index; i >= 0; i-- {
		r := rune(doc.SourceCode.Text[i])
		//fmt.Printf("%c\n", r)
		if utils.IsAZ09_(r) || r == '.' || r == ':' {
			startIndex = i
		} else {
			break
		}
	}
	sentence := doc.SourceCode.Text[startIndex : index+1]
	// fmt.Println("sentence: ", sentence)

	containsModulePathSeparator := strings.Contains(sentence, ":")
	containsChainSeparator := strings.Contains(sentence, ".")

	return (!containsModulePathSeparator && !containsChainSeparator) || (containsModulePathSeparator && !containsChainSeparator), sentence
}

// Checks if writing seems to be a chain of components (example: aStruct.aMember)
// If that's the case, it will return the position of the last character of previous token
func isCompletingAChain(doc *document.Document, cursorPosition symbols.Position) (bool, symbols.Position) {
	// Cursor is just right after last char, let's rewind one place
	position := cursorPosition
	if cursorPosition.Character > 0 {
		position = doc.RewindPosition(cursorPosition)
	}

	index := position.IndexIn(doc.SourceCode.Text)

	// Read backwards until a separator character is found.
	startIndex := index
	for i := index; i >= 0; i-- {
		r := rune(doc.SourceCode.Text[i])
		//fmt.Printf("%c\n", r)
		if utils.IsAZ09_(r) || r == '.' || r == ':' {
			startIndex = i
		} else {
			break
		}
	}
	sentence := doc.SourceCode.Text[startIndex : index+1]
	//fmt.Println("sentence: ", sentence)
	var previousPosition symbols.Position

	containsSeparator := strings.Contains(sentence, ".") || strings.Contains(sentence, ":")
	if containsSeparator {
		lastIndex := len(sentence) - 1
		if strings.Contains(sentence, ".") {
			lastIndex = strings.LastIndex(sentence, ".")
		} else if strings.Contains(sentence, "::") {
			lastIndex = strings.LastIndex(sentence, "::")
		}
		sub := len(sentence) - lastIndex
		previousPosition = symbols.NewPosition(
			position.Line,
			cursorPosition.Character-uint(sub)-1, // one extra -1 to stay just behind last character
		)
	}

	return containsSeparator, previousPosition
}

func extractExplicitModulePath(possibleModulePath string) option.Option[symbols.ModulePath] {
	// Read backwards until a separator character is found.
	lastCharIndex := len(possibleModulePath) - 1
	firstDoubleColonFound := -1
	separatorsInARow := 0

	for i := lastCharIndex; i >= 0; i-- {
		r := rune(possibleModulePath[i])
		//fmt.Printf("%c\n", r)
		if firstDoubleColonFound == -1 {
			if r == ':' {
				separatorsInARow++
			}

			if separatorsInARow == 2 {
				firstDoubleColonFound = i
			}
		}

		if r != ':' {
			separatorsInARow = 0
		}

		if r == '.' {
			break
		}
	}

	if firstDoubleColonFound != -1 {
		return option.Some(symbols.NewModulePathFromString(possibleModulePath[0:firstDoubleColonFound]))
	}
	return option.None[symbols.ModulePath]()
}

// Obtains a doc comment's representation as markup, or nil.
// Only the body is included (not contracts) for brevity.
// Returns: nil | MarkupContent
func GetCompletableDocComment(s symbols.Indexable) any {
	docComment := s.GetDocComment()
	content := ""
	if docComment != nil {
		content = docComment.GetBody()
	}

	if module, ok := s.(*symbols.Module); ok {
		constraints := symbols.ModuleGenericConstraintMarkdown(module)
		if constraints != "" {
			if content != "" {
				content += "\n\n"
			}
			content += constraints
		}
	}

	if content == "" {
		return nil
	} else {
		return protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: content,
		}
	}
}

func GetCompletionDetail(s symbols.Indexable) *string {
	detail := s.GetCompletionDetail()
	if detail == "" {
		return nil
	} else {
		return &detail
	}
}

func completionItemKey(item protocol.CompletionItem) string {
	kind := ""
	if item.Kind != nil {
		kind = fmt.Sprintf("%d", *item.Kind)
	}

	detail := ""
	if item.Detail != nil {
		detail = *item.Detail
	}

	return item.Label + "|" + kind + "|" + detail
}

// Search for a type's methods.
func (s *Search) BuildMethodCompletions(
	state *l.ProjectState,
	parentTypeFQN string,
	filterMembers bool,
	symbolToSearch sourcecode.Word,
	docText string,
) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Search in enum methods
	var methods []symbols.Indexable
	var query string
	if !filterMembers {
		query = parentTypeFQN + "."
	} else {
		query = parentTypeFQN + "." + symbolToSearch.Text() + "*"
	}

	replaceStart := symbolToSearch.PrevAccessPath().TextRange().End
	replaceStart.Character += 1
	replaceEnd := replaceStart
	replaceEnd.Character += 1
	if symbolToSearch.IsSeparator() {
		replaceStart = symbolToSearch.TextRange().End
		replaceEnd = symbolToSearch.TextRange().End

		nextIndex := replaceStart.IndexIn(docText)
		if nextIndex >= 0 && nextIndex < len(docText) && utils.IsAZ09_(rune(docText[nextIndex])) {
			replaceEnd.Character += 1
		}
	}

	replacementRange := protocol_utils.NewLSPRange(
		uint32(replaceStart.Line),
		uint32(replaceStart.Character),
		uint32(replaceEnd.Line),
		uint32(replaceEnd.Character),
	)
	methods = state.SearchByFQN(query)
	for _, idx := range methods {
		fn, success := idx.(*symbols.Function)
		if !success {
			s.logger.Warning("unexpected: query returned non function symbol", "type", fmt.Sprintf("%T", idx))
			continue
		}
		kind := idx.GetKind()
		items = append(items, protocol.CompletionItem{
			Label: fn.GetName(),
			Kind:  &kind,
			TextEdit: protocol.TextEdit{
				NewText: fn.GetMethodName(),
				Range:   replacementRange,
			},
			Documentation: GetCompletableDocComment(fn),
			Detail:        GetCompletionDetail(fn),
		})
	}

	return items
}

// Returns: []CompletionItem | CompletionList | nil
func (s *Search) BuildCompletionList(
	ctx context.CursorContext,
	state *l.ProjectState,
) []protocol.CompletionItem {
	if ctx.IsLiteral {
		return []protocol.CompletionItem{}
	}

	var items []protocol.CompletionItem

	filterMembers := true

	doc := state.GetDocument(ctx.DocURI)
	unlockDocument := state.LockDocument(doc.URI)
	defer unlockDocument()

	symbolInPosition := completionSymbolAtCursor(doc, state, ctx.Position)
	scopeCtx := buildCompletionScopeContext(state, doc, ctx.Position)
	hasContextAtCursor := hasCompletionContextAtCursor(symbolInPosition, ctx.Position, doc.SourceCode.Text)
	prefix := completionPrefix(symbolInPosition, hasContextAtCursor)

	if prefix == "" {
		filterMembers = false
	}

	// Check if it might be a C3 language keyword
	keywordKind := protocol.CompletionItemKindKeyword
	for keyword := range c3.Keywords() {
		shouldIncludeKeyword := false
		if prefix != "" {
			shouldIncludeKeyword = strings.HasPrefix(keyword, prefix)
		} else if !hasContextAtCursor {
			// Empty/blank invocation context (for example Ctrl+Space in an empty line)
			// should still offer keyword suggestions.
			shouldIncludeKeyword = true
		}

		if shouldIncludeKeyword {
			items = append(items, protocol.CompletionItem{
				Label: keyword,
				Kind:  &keywordKind,
			})
		}
	}

	if symbolInPosition.IsSeparator() || !hasContextAtCursor {
		// Probably, theres no symbol at cursor!
		filterMembers = false
	}
	s.logger.Debug("building completion list", "symbol", symbolInPosition.Text())

	// Check if module path is being written/exists
	isCompletingModulePath, possibleModulePath := isCompletingAModulePath(doc, ctx.Position)
	if !hasContextAtCursor {
		isCompletingModulePath = false
	} else if symbolInPosition.IsSeparator() && symbolInPosition.HasAccessPath() {
		isCompletingModulePath = false
	}

	hasExplicitModulePath := option.None[symbols.ModulePath]()
	if isCompletingModulePath {
		hasExplicitModulePath = extractExplicitModulePath(possibleModulePath)
	}

	//isCompletingAChain, prevPosition := isCompletingAChain(doc, position)
	isCompletingAChain := symbolInPosition.HasAccessPath() && hasContextAtCursor

	// There are two cases (TBC):
	// User writing a symbol:
	//		user expects either
	//		- autocomplete to suggest loadable symbol names. Including module names!
	//		- help him with autocompleting a module path they are currently writing
	// User completing a chain of calls:
	//		user expects to autocomplete with member/methods of previous children.

	if !isCompletingModulePath && isCompletingAChain {
		// Is writing a symbol child of a parent one.
		// We need to limit the search to subtypes of parent token
		// Let's find parent token
		prevEndPos := symbolInPosition.PrevAccessPath().TextRange().End
		rewoundPos := doc.RewindPosition(prevEndPos)

		searchParams := sp.BuildSearchBySymbolUnderCursor(
			doc,
			*state.GetUnitModulesByDoc(doc.URI),
			rewoundPos,
		)

		//	searchParams.scopeMode = AnyPosition

		membersReadable, fromDistinct, initialItems, prevIndexableOption := s.findParentTypeWithCompletions(
			filterMembers,
			symbolInPosition,
			searchParams,
			state,
			FindDebugger{depth: 0, enabled: true},
		)

		items = append(items, initialItems...)

		if prevIndexableOption.IsNone() && symbolInPosition.IsSeparator() {
			// The search found nothing and we're on a bare dot (e.g. "foo." with no
			// character after the dot). This typically happens because tree-sitter
			// cannot parse "foo.;" as a valid expression and produces an ERROR node,
			// which causes local variable declarations to be lost from the function body.
			//
			// Work around this by inserting a placeholder identifier at the cursor
			// position, re-parsing the document temporarily, and retrying the search.
			placeholderDoc, placeholderSymbol, cleanup := s.retryWithPlaceholder(doc, ctx.Position, state)
			if placeholderDoc != nil {
				defer cleanup()

				prevEndPos := placeholderSymbol.PrevAccessPath().TextRange().End
				rewoundPos := placeholderDoc.RewindPosition(prevEndPos)
				searchParams = sp.BuildSearchBySymbolUnderCursor(
					placeholderDoc,
					*state.GetUnitModulesByDoc(doc.URI),
					rewoundPos,
				)
				membersReadable, fromDistinct, initialItems, prevIndexableOption = s.findParentTypeWithCompletions(
					false,
					placeholderSymbol,
					searchParams,
					state,
					FindDebugger{depth: 0, enabled: true},
				)
				items = append(items, initialItems...)
			}
		}

		if prevIndexableOption.IsNone() {
			inferredType := inferVariableTypeNameFromText(doc, ctx.Position, symbolInPosition.PrevAccessPath().Text())
			if inferredType.IsSome() {
				fallbackParams := sp.NewSearchParamsBuilder().
					WithText(inferredType.Get(), symbolInPosition.PrevAccessPath().TextRange()).
					WithDocId(doc.URI).
					WithContextModuleName(searchParams.ModuleInCursor()).
					WithScopeMode(sp.InModuleRoot).
					Build()

				fallbackParent := s.findClosestSymbolDeclaration(fallbackParams, state, FindDebugger{depth: 0, enabled: true})
				if fallbackParent.IsSome() {
					prevIndexableOption = option.Some(fallbackParent.Get())
					membersReadable = false
					fromDistinct = NotFromDistinct
				}
			}
		}

		if prevIndexableOption.IsNone() {
			return items
		}

		// Can only read methods if the current type being inspected wasn't the base type of a distinct,
		// or if it was, then we're currently inspecting an inline distinct INSTANCE and not the type itself, since
		// methods are scoped to their concrete type names.
		methodsReadable := fromDistinct == NotFromDistinct || (fromDistinct == InlineDistinct && !membersReadable)

		prevIndexable := prevIndexableOption.Get()
		// fmt.Print(prevIndexable.GetName())

		switch prevIndexable := prevIndexable.(type) {

		case *symbols.Struct:
			strukt := prevIndexable

			// We don't check for 'membersReadable' here since even variables of structs
			// can access its members. In addition, distincts of structs can always
			// access struct members regardless of being inline, so we don't need to
			// check for distinct procedence here either.
			// TODO: Actually, maybe we should check for NOT membersReadable if it is
			// impossible to access Struct.member as a type.
			for _, member := range strukt.GetMembers() {
				if !filterMembers || strings.HasPrefix(member.GetName(), symbolInPosition.Text()) {
					items = append(items, protocol.CompletionItem{
						Label: member.GetName(),
						Kind:  &member.Kind,

						// At this moment, struct members cannot receive documentation
						Documentation: nil,

						Detail: GetCompletionDetail(member),
					})
				}
			}

			// If this struct was the base type of a non-inline distinct variable,
			// do not suggest its methods, as they cannot be accessed
			if methodsReadable {
				items = append(items, s.BuildMethodCompletions(state, strukt.GetFQN(), filterMembers, symbolInPosition, doc.SourceCode.Text)...)
			}

		case *symbols.Enumerator:
			enumerator := prevIndexable

			// Associated values are always available regardless of distinct status.
			for _, assoc := range enumerator.AssociatedValues {
				if !filterMembers || strings.HasPrefix(assoc.GetName(), symbolInPosition.Text()) {
					items = append(items, protocol.CompletionItem{
						Label: assoc.GetName(),
						Kind:  &assoc.Kind,

						// No documentation for associated values at this time
						Documentation: nil,

						Detail: GetCompletionDetail(&assoc),
					})
				}
			}

			// Add parent enum's methods, but only if this doesn't come from a non-inline distinct.
			if methodsReadable && enumerator.GetModuleString() != "" && enumerator.GetEnumName() != "" {
				items = append(items, s.BuildMethodCompletions(state, enumerator.GetEnumFQN(), filterMembers, symbolInPosition, doc.SourceCode.Text)...)
			}

		case *symbols.FaultConstant:
			constant := prevIndexable

			// Add parent fault's methods
			if methodsReadable && constant.GetModuleString() != "" && constant.GetFaultName() != "" {
				items = append(items, s.BuildMethodCompletions(state, constant.GetFaultFQN(), filterMembers, symbolInPosition, doc.SourceCode.Text)...)
			}

		case *symbols.Enum:
			enum := prevIndexable

			// Accessing MyEnum.VALUE is ok, but not MyEnum.VALUE.VALUE,
			// so don't search for enumerators within enumerators
			// (membersReadable = false).
			// However, 'DistinctEnum.VALUE' is always invalid.
			if membersReadable && fromDistinct == NotFromDistinct {
				for _, enumerator := range enum.GetEnumerators() {
					if !filterMembers || strings.HasPrefix(enumerator.GetName(), symbolInPosition.Text()) {
						items = append(items, protocol.CompletionItem{
							Label: enumerator.GetName(),
							Kind:  &enumerator.Kind,

							// No documentation for enumerators at this time
							Documentation: nil,

							Detail: GetCompletionDetail(enumerator),
						})
					}
				}
			} else if !membersReadable {
				// This is an enum instance, so we can access associated values.
				// Always valid for distincts, so we don't check this here.
				for _, assoc := range enum.GetAssociatedValues() {
					if !filterMembers || strings.HasPrefix(assoc.GetName(), symbolInPosition.Text()) {
						items = append(items, protocol.CompletionItem{
							Label: assoc.GetName(),
							Kind:  &assoc.Kind,

							// No documentation for associated values at this time
							Documentation: nil,

							Detail: GetCompletionDetail(&assoc),
						})
					}
				}
			}

			if methodsReadable {
				items = append(items, s.BuildMethodCompletions(state, enum.GetFQN(), filterMembers, symbolInPosition, doc.SourceCode.Text)...)
			}

		case *symbols.FaultDef:
			fault := prevIndexable

			// Accessing MyFault.VALUE is ok, but not MyFault.VALUE.VALUE,
			// so don't search for constants within constants
			// (membersReadable = false).
			if membersReadable && fromDistinct == NotFromDistinct {
				for _, constant := range fault.GetConstants() {
					if !filterMembers || strings.HasPrefix(constant.GetName(), symbolInPosition.Text()) {
						items = append(items, protocol.CompletionItem{
							Label: constant.GetName(),
							Kind:  &constant.Kind,

							// No documentation for fault constants at this time
							Documentation: nil,

							Detail: GetCompletionDetail(constant),
						})
					}
				}
			}

			if methodsReadable {
				items = append(items, s.BuildMethodCompletions(state, fault.GetFQN(), filterMembers, symbolInPosition, doc.SourceCode.Text)...)
			}
		}
	} else {
		// Find all symbols in module
		params := FindSymbolsParams{
			docId:              doc.URI,
			scopedToModulePath: hasExplicitModulePath,
			position:           option.Some(ctx.Position),
		}
		// Search symbols loadable in module located in position
		scopeSymbols := s.findSymbolsInScope(params, state)

		for _, storedIdentifier := range scopeSymbols {
			if storedIdentifier.GetKind() == protocol.CompletionItemKindMethod {
				continue
			}

			hasPrefix := strings.HasPrefix(storedIdentifier.GetName(), prefix)
			if filterMembers && !hasPrefix {
				continue
			}

			if storedIdentifier.GetKind() == protocol.CompletionItemKindModule && hasContextAtCursor {
				/*fullSymbolAtCursor, _ := doc.SymbolBeforeCursor(
					symbols.Position{
						Line:      uint(position.Line),
						Character: uint(position.Character) - 1,
					})
				fullSymbolAtCursor.AdvanceEndCharacter()*/
				editRange := symbolInPosition.FullTextRange().ToLSP()

				items = append(items, protocol.CompletionItem{
					Label: storedIdentifier.GetName(),
					Kind:  cast.ToPtr(storedIdentifier.GetKind()),
					TextEdit: protocol.TextEdit{
						NewText: storedIdentifier.GetName(),
						Range:   editRange,
					},
					Documentation: GetCompletableDocComment(storedIdentifier),
					Detail:        GetCompletionDetail(storedIdentifier),
				})
			} else {
				items = append(items, protocol.CompletionItem{
					Label:         storedIdentifier.GetName(),
					Kind:          cast.ToPtr(storedIdentifier.GetKind()),
					Documentation: GetCompletableDocComment(storedIdentifier),
					Detail:        GetCompletionDetail(storedIdentifier),
				})
			}
		}

		if !hasContextAtCursor && hasExplicitModulePath.IsNone() {
			items = append(items, importedModuleItems(state, scopeCtx, prefix)...)
		}
	}

	uniqueItems := make([]protocol.CompletionItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := completionItemKey(item)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		uniqueItems = append(uniqueItems, item)
	}

	if !isCompletingAChain && !hasContextAtCursor {
		hasExplicitModulePathBool := hasExplicitModulePath.IsSome()
		for i := range uniqueItems {
			sortText := completionSortText(uniqueItems[i], scopeCtx, hasExplicitModulePathBool)
			uniqueItems[i].SortText = &sortText
		}

		slices.SortFunc(uniqueItems, func(a, b protocol.CompletionItem) int {
			arank := completionScopeRank(a, scopeCtx, hasExplicitModulePathBool)
			brank := completionScopeRank(b, scopeCtx, hasExplicitModulePathBool)
			if arank != brank {
				return cmp.Compare(arank, brank)
			}

			return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
		})
	} else {
		slices.SortFunc(uniqueItems, func(a, b protocol.CompletionItem) int {
			return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
		})
	}

	return uniqueItems
}

// retryWithPlaceholder works around a tree-sitter parsing limitation where
// a bare dot expression like "c.;" produces an ERROR node, causing local
// variable declarations in the same function body to be lost.
//
// It patches ALL bare dot expressions in the source (not just the one at the
// cursor) because a single broken expression elsewhere in the file can cause
// tree-sitter's error recovery to swallow the entire function body into an
// ERROR node, hiding variable declarations needed for completion.
//
// A "bare dot" is a '.' followed by a non-identifier character such as
// whitespace, ';', newline, ')', '}', or ']'. For each bare dot it inserts
// a placeholder identifier ("_z") and, when there is no trailing semicolon
// before a newline, also appends one so tree-sitter can parse the statement.
//
// Returns the placeholder document, the symbol-in-position from the modified
// source, and a cleanup function that MUST be called (typically via defer) to
// restore the original document in the project state.
//
// If the placeholder insertion fails for any reason, returns
// (nil, Word{}, func() {}).
func (s *Search) retryWithPlaceholder(
	doc *document.Document,
	cursorPos symbols.Position,
	state *l.ProjectState,
) (*document.Document, sourcecode.Word, func()) {
	cleanup := func() {}

	source := doc.SourceCode.Text
	cursorOffset := cursorPos.IndexIn(source)
	if cursorOffset < 0 || cursorOffset > len(source) {
		return nil, sourcecode.Word{}, cleanup
	}

	modified, newCursorOffset := patchBrokenSeparators(source, cursorOffset)

	placeholderDoc := document.NewDocument(doc.URI, modified)
	parser := p.NewParser(s.logger)
	state.RefreshDocumentIdentifiers(&placeholderDoc, &parser)

	// Compute the new cursor position from the adjusted offset.
	newCursorPos := placeholderDoc.SourceCode.OffsetToPosition(newCursorOffset)

	placeholderSymbol := placeholderDoc.SourceCode.SymbolInPosition(
		newCursorPos,
		state.GetUnitModulesByDoc(doc.URI),
	)

	cleanup = func() {
		// Restore the original document's symbols in the project state.
		parser := p.NewParser(s.logger)
		state.RefreshDocumentIdentifiers(doc, &parser)
	}

	return &placeholderDoc, placeholderSymbol, cleanup
}

// patchBrokenSeparators finds every bare "." and bare "::" in source and
// inserts a placeholder identifier ("_z") so tree-sitter can parse them as
// valid field accesses / module paths. If a bare separator appears at the end
// of a statement line (no semicolon before newline and not inside parentheses),
// a semicolon is also appended.
//
// cursorOffset is the byte offset of the cursor in the original source
// (typically the character right after the trigger separator).
//
// Returns the patched source and the adjusted cursor offset in the new source.
func patchBrokenSeparators(source string, cursorOffset int) (string, int) {
	var b strings.Builder
	b.Grow(len(source) + 64)

	newCursorOffset := cursorOffset
	parenDepth := 0
	i := 0

	for i < len(source) {
		ch := source[i]

		// Track parenthesis depth so we don't insert semicolons inside calls.
		if ch == '(' {
			parenDepth++
		} else if ch == ')' && parenDepth > 0 {
			parenDepth--
		}

		// --- Bare "::" ---
		if ch == ':' && i+1 < len(source) && source[i+1] == ':' && i > 0 {
			prevCh := source[i-1]
			if isIdentChar(prevCh) {
				afterColons := byte(0)
				if i+2 < len(source) {
					afterColons = source[i+2]
				}
				if !isIdentChar(afterColons) {
					b.WriteString("::")
					posAfter := i + 2
					i = posAfter

					b.WriteString("_z")
					if posAfter < cursorOffset {
						newCursorOffset += 2
					}

					newCursorOffset += writeStatementEnding(&b, source, i, parenDepth, posAfter, cursorOffset)
					parenDepth = 0 // closing parens resets depth
					continue
				}
			}
		}

		// --- Bare "." ---
		if ch == '.' && i > 0 {
			prevCh := source[i-1]
			if isIdentChar(prevCh) || prevCh == ')' {
				nextCh := byte(0)
				if i+1 < len(source) {
					nextCh = source[i+1]
				}
				if !isIdentChar(nextCh) {
					b.WriteByte('.')
					posAfter := i + 1
					i = posAfter

					b.WriteString("_z")
					if posAfter < cursorOffset {
						newCursorOffset += 2
					}

					newCursorOffset += writeStatementEnding(&b, source, i, parenDepth, posAfter, cursorOffset)
					parenDepth = 0 // closing parens resets depth
					continue
				}
			}
		}

		b.WriteByte(ch)
		i++
	}

	return b.String(), newCursorOffset
}

// writeStatementEnding appends closing parentheses (if parenDepth > 0) and a
// semicolon (if the rest of the line has none) after a patched separator.
// Returns the number of bytes by which the cursor offset should be increased.
func writeStatementEnding(b *strings.Builder, source string, i int, parenDepth int, posAfter int, cursorOffset int) int {
	shift := 0
	beforeCursor := posAfter < cursorOffset

	if needsSemicolon(source, i) {
		// Close any unclosed parentheses.
		for p := 0; p < parenDepth; p++ {
			b.WriteByte(')')
			if beforeCursor {
				shift++
			}
		}
		b.WriteByte(';')
		if beforeCursor {
			shift++
		}
	}
	// If parenDepth > 0 but there's already a semicolon on this line, we don't
	// close parens because the user may have the closing paren elsewhere.

	return shift
}

// isIdentChar returns true if c is a valid C3 identifier character [a-zA-Z0-9_$].
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

// needsSemicolon checks whether position i in source is followed (ignoring
// spaces/tabs) by a newline or EOF, meaning the statement has no semicolon.
func needsSemicolon(source string, i int) bool {
	for j := i; j < len(source); j++ {
		switch source[j] {
		case ' ', '\t':
			continue
		case '\n', '\r':
			return true
		case ';':
			return false
		default:
			return false
		}
	}
	// EOF without semicolon
	return true
}

// Returns whether members can be read from the found symbol, the 'fromDistinct' status, the list of
// completions found while resolving distincts in a distinct chain (if any), as well as the final symbol
// found for further completions.
func (s *Search) findParentTypeWithCompletions(
	filterMembers bool,
	symbolInPosition sourcecode.Word,
	searchParams sp.SearchParams,
	state *l.ProjectState,
	debugger FindDebugger,
) (bool, int, []protocol.CompletionItem, option.Option[symbols.Indexable]) {
	prevIndexableResult := s.findInParentSymbols(searchParams, state, debugger)
	membersReadable := prevIndexableResult.membersReadable
	fromDistinct := prevIndexableResult.fromDistinct
	items := []protocol.CompletionItem{}
	if prevIndexableResult.IsNone() {
		return membersReadable, fromDistinct, items, prevIndexableResult.result
	}
	symbolsHierarchy := []symbols.Indexable{}
	prevIndexable := prevIndexableResult.Get()

	// Can only read methods if the current type being inspected wasn't the base type of a distinct,
	// or if it was, then we're currently inspecting an inline distinct INSTANCE and not the type itself, since
	// methods are scoped to their concrete type names.
	methodsReadable := fromDistinct == NotFromDistinct || (fromDistinct == InlineDistinct && !membersReadable)

	// Use a loop to iteratively resolve and add completions of distincts in a distinct
	// chain, that is, a distinct of distinct of ... of (base type).
	// Completions all the way down are valid.
	// This same loop will resolve any distincts pointing to def aliases and such through `s.resolve`.
	// Then, the indexable is converted into its base type, which needs no further resolution.
	protect := 0
	for {
		if protect > 1000 {
			return true, NotFromDistinct, items, option.None[symbols.Indexable]()
		}
		protect++

		distinct, isDistinct := prevIndexable.(*symbols.TypeDef)

		// If this distinct was the base type of a non-inline distinct, keep the
		// status of non-inline distinct, since we can no longer access methods
		if isDistinct && fromDistinct != NonInlineDistinct {
			docText := ""
			if doc := state.GetDocument(searchParams.DocId().Get()); doc.URI != "" {
				docText = doc.SourceCode.Text
			}
			// Complete distinct-exclusive methods, but only if this is the original
			// base type, an instance of it, or an instance of an inline distinct
			// pointing to it.
			if methodsReadable {
				items = append(items, s.BuildMethodCompletions(state, distinct.GetFQN(), filterMembers, symbolInPosition, docText)...)
			}

			if distinct.IsInline() {
				fromDistinct = InlineDistinct

				// Can only read methods on INSTANCES of inline distincts.
				methodsReadable = methodsReadable && !membersReadable
			} else {
				fromDistinct = NonInlineDistinct
				methodsReadable = false
			}
		}

		if isDistinct || !isInspectable(prevIndexable) {
			prevIndexable = s.resolve(prevIndexable, searchParams.DocId().Get(), searchParams.ModuleInCursor(), state, symbolsHierarchy, debugger)
			if prevIndexable == nil {
				// No point in trying to complete methods / members when the resolved type is not
				// inspectable and doesn't resolve to anything that is inspectable
				return true, NotFromDistinct, items, option.None[symbols.Indexable]()
			}

			// Important for generic type resolution above
			symbolsHierarchy = append(symbolsHierarchy, prevIndexable)
		} else {
			// Hit a concrete, inspectable type to analyze, let's proceed.
			break
		}
	}

	var resolvedIndexable option.Option[symbols.Indexable]

	// Might need to do an additional resolution step even if it's inspectable
	switch prevIndexable := prevIndexable.(type) {
	case *symbols.StructMember:
		var token sourcecode.Word
		structMember := prevIndexable
		token = sourcecode.NewWord(structMember.GetType().GetName(), prevIndexable.GetIdRange())

		// Resolve a struct member into its field type for completion
		levelSearchParams := sp.NewSearchParamsBuilder().
			//WithSymbol(token.Text()).
			WithSymbolWord(
				sourcecode.NewWord(token.Text(), token.TextRange()),
			).
			WithDocId(prevIndexable.GetDocumentURI()).
			Build()

		resolvedIndexable = s.findClosestSymbolDeclaration(levelSearchParams, state, debugger.goIn()).result
	default:
		resolvedIndexable = option.Some(prevIndexable)
	}

	return membersReadable, fromDistinct, items, resolvedIndexable
}
