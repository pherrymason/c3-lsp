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

		prevIndexableOption := s.findParentType(searchParams, state, FindDebugger{depth: 0, enabled: true})
		if prevIndexableOption.IsNone() {
			return items
		}
		prevIndexable := prevIndexableOption.Get()
		//fmt.Print(prevIndexable.GetName())

		switch prevIndexable.(type) {

		case *symbols.Struct:
			strukt := prevIndexable.(*symbols.Struct)

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

			// Search in struct methods
			var methods []symbols.Indexable
			var query string
			if !filterMembers {
				query = strukt.GetFQN() + "."
			} else {
				query = strukt.GetFQN() + "." + symbolInPosition.Text() + "*"
			}

			replacementRange := protocol_utils.NewLSPRange(
				uint32(symbolInPosition.PrevAccessPath().TextRange().Start.Line),
				uint32(symbolInPosition.PrevAccessPath().TextRange().End.Character+1),
				uint32(symbolInPosition.PrevAccessPath().TextRange().Start.Line),
				uint32(symbolInPosition.PrevAccessPath().TextRange().End.Character+2),
			)
			methods = state.SearchByFQN(query)
			for _, idx := range methods {
				fn, _ := idx.(*symbols.Function)
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

		case *symbols.Enum:
			enum := prevIndexable.(*symbols.Enum)
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

		case *symbols.Fault:
			fault := prevIndexable.(*symbols.Fault)
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

func (s *Search) findParentType(searchParams sp.SearchParams, state *l.ProjectState, debugger FindDebugger) option.Option[symbols.Indexable] {
	prevIndexableResult := s.findInParentSymbols(searchParams, state, debugger)
	if prevIndexableResult.IsNone() {
		return prevIndexableResult.result
	}
	symbolsHierarchy := []symbols.Indexable{}

	prevIndexable := prevIndexableResult.Get()

	for {
		if !isInspectable(prevIndexable) {
			prevIndexable = s.resolve(prevIndexable, searchParams.DocId().Get(), searchParams.ModuleInCursor(), state, symbolsHierarchy, debugger)

			if prevIndexable == nil {
				return option.None[symbols.Indexable]()
			}
		} else {
			break
		}
	}

	switch prevIndexable.(type) {
	case *symbols.StructMember:
		var token sourcecode.Word
		structMember, _ := prevIndexable.(*symbols.StructMember)
		token = sourcecode.NewWord(structMember.GetType().GetName(), prevIndexable.GetIdRange())

		levelSearchParams := sp.NewSearchParamsBuilder().
			//WithSymbol(token.Text()).
			WithSymbolWord(
				sourcecode.NewWord(token.Text(), token.TextRange()),
			).
			WithDocId(prevIndexable.GetDocumentURI()).
			Build()

		prevIndexableResult = s.findClosestSymbolDeclaration(levelSearchParams, state, debugger.goIn())
	default:
		return option.Some(prevIndexable)
	}

	return prevIndexableResult.result
}
