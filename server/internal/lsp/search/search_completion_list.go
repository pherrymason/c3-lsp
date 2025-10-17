package search

import (
	"cmp"
	"fmt"
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
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func isCompletingAModulePath(doc *document.Document, cursorPosition symbols.Position) (bool, string) {
	// Cursor is just right after last char, let's rewind one place
	position := cursorPosition
	if cursorPosition.Character > 0 {
		position = cursorPosition.RewindCharacter()
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
		position = cursorPosition.RewindCharacter()
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
	if docComment == nil || docComment.GetBody() == "" {
		return nil
	} else {
		return protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: docComment.GetBody(),
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

// Search for a type's methods.
func (s *Search) BuildMethodCompletions(
	state *l.ProjectState,
	parentTypeFQN string,
	filterMembers bool,
	symbolToSearch sourcecode.Word,
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

	replacementRange := protocol_utils.NewLSPRange(
		uint32(symbolToSearch.PrevAccessPath().TextRange().Start.Line),
		uint32(symbolToSearch.PrevAccessPath().TextRange().End.Character+1),
		uint32(symbolToSearch.PrevAccessPath().TextRange().Start.Line),
		uint32(symbolToSearch.PrevAccessPath().TextRange().End.Character+2),
	)
	methods = state.SearchByFQN(query)
	for _, idx := range methods {
		fn, success := idx.(*symbols.Function)
		if !success {
			s.logger.Warningf("unexpected: query returned non function symbol of type %T, skipping", idx)
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

	filterMembers := true /*
			symbolInPosition, error := doc.SymbolInPositionDeprecated(
				symbols.Position{
					Line:      uint(position.Line),
					Character: uint(position.Character - 1),
				})
				if error != nil {
			// Probably, theres no symbol at cursor!
			filterMembers = false
		}
	*/

	doc := state.GetDocument(ctx.DocURI)
	symbolInPosition := doc.SourceCode.SymbolInPosition(
		ctx.Position.RewindCharacter(),
		state.GetUnitModulesByDoc(doc.URI),
	)

	// Check if it might be a C3 language keyword
	keywordKind := protocol.CompletionItemKindKeyword
	for keyword := range c3.Keywords() {
		if strings.HasPrefix(keyword, symbolInPosition.Text()) {
			items = append(items, protocol.CompletionItem{
				Label: keyword,
				Kind:  &keywordKind,
			})
		}
	}

	if symbolInPosition.IsSeparator() {
		// Probably, theres no symbol at cursor!
		filterMembers = false
	}
	s.logger.Debug(fmt.Sprintf("building completion list: \"%s\"", symbolInPosition.Text())) //TODO warp %s en "

	// Check if module path is being written/exists
	isCompletingModulePath, possibleModulePath := isCompletingAModulePath(doc, ctx.Position)

	hasExplicitModulePath := option.None[symbols.ModulePath]()
	if isCompletingModulePath {
		hasExplicitModulePath = extractExplicitModulePath(possibleModulePath)
	}

	//isCompletingAChain, prevPosition := isCompletingAChain(doc, position)
	isCompletingAChain := symbolInPosition.HasAccessPath()

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

		searchParams := sp.BuildSearchBySymbolUnderCursor(
			doc,
			*state.GetUnitModulesByDoc(doc.URI),
			symbolInPosition.PrevAccessPath().TextRange().End.RewindCharacter(),
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

		if prevIndexableOption.IsNone() {
			return items
		}

		// Can only read methods if the current type being inspected wasn't the base type of a distinct,
		// or if it was, then we're currently inspecting an inline distinct INSTANCE and not the type itself, since
		// methods are scoped to their concrete type names.
		methodsReadable := fromDistinct == NotFromDistinct || (fromDistinct == InlineDistinct && !membersReadable)

		prevIndexable := prevIndexableOption.Get()
		// fmt.Print(prevIndexable.GetName())

		switch prevIndexable.(type) {

		case *symbols.Struct:
			strukt := prevIndexable.(*symbols.Struct)

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
				items = append(items, s.BuildMethodCompletions(state, strukt.GetFQN(), filterMembers, symbolInPosition)...)
			}

		case *symbols.Enumerator:
			enumerator := prevIndexable.(*symbols.Enumerator)

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
				items = append(items, s.BuildMethodCompletions(state, enumerator.GetEnumFQN(), filterMembers, symbolInPosition)...)
			}

		case *symbols.FaultConstant:
			constant := prevIndexable.(*symbols.FaultConstant)

			// Add parent fault's methods
			if methodsReadable && constant.GetModuleString() != "" && constant.GetFaultName() != "" {
				items = append(items, s.BuildMethodCompletions(state, constant.GetFaultFQN(), filterMembers, symbolInPosition)...)
			}

		case *symbols.Enum:
			enum := prevIndexable.(*symbols.Enum)

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
				items = append(items, s.BuildMethodCompletions(state, enum.GetFQN(), filterMembers, symbolInPosition)...)
			}

		case *symbols.Fault:
			fault := prevIndexable.(*symbols.Fault)

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
				items = append(items, s.BuildMethodCompletions(state, fault.GetFQN(), filterMembers, symbolInPosition)...)
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
			hasPrefix := strings.HasPrefix(storedIdentifier.GetName(), symbolInPosition.Text())
			if filterMembers && !hasPrefix {
				continue
			}

			if storedIdentifier.GetKind() == protocol.CompletionItemKindModule {
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
	}

	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})

	return items
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

		distinct, isDistinct := prevIndexable.(*symbols.Distinct)

		// If this distinct was the base type of a non-inline distinct, keep the
		// status of non-inline distinct, since we can no longer access methods
		if isDistinct && fromDistinct != NonInlineDistinct {
			// Complete distinct-exclusive methods, but only if this is the original
			// base type, an instance of it, or an instance of an inline distinct
			// pointing to it.
			if methodsReadable {
				items = append(items, s.BuildMethodCompletions(state, distinct.GetFQN(), filterMembers, symbolInPosition)...)
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
	switch prevIndexable.(type) {
	case *symbols.StructMember:
		var token sourcecode.Word
		structMember, _ := prevIndexable.(*symbols.StructMember)
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
