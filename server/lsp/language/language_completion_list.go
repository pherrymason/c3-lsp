package language

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	sp "github.com/pherrymason/c3-lsp/lsp/search_params"
	"github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	"github.com/pherrymason/c3-lsp/option"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func isCompletingAModulePath(doc *document.Document, cursorPosition symbols.Position) (bool, string) {
	// Cursor is just right after last char, let's rewind one place
	position := cursorPosition
	if cursorPosition.Character > 0 {
		position = cursorPosition.RewindCharacter()
	}

	index := position.IndexIn(doc.Content)

	// Read backwards until a separator character is found.
	startIndex := index
	for i := index; i >= 0; i-- {
		r := rune(doc.Content[i])
		//fmt.Printf("%c\n", r)
		if utils.IsAZ09_(r) || r == '.' || r == ':' {
			startIndex = i
		} else {
			break
		}
	}
	sentence := doc.Content[startIndex : index+1]
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

	index := position.IndexIn(doc.Content)

	// Read backwards until a separator character is found.
	startIndex := index
	for i := index; i >= 0; i-- {
		r := rune(doc.Content[i])
		//fmt.Printf("%c\n", r)
		if utils.IsAZ09_(r) || r == '.' || r == ':' {
			startIndex = i
		} else {
			break
		}
	}
	sentence := doc.Content[startIndex : index+1]
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

func (l *Language) BuildCompletionList(doc *document.Document, position symbols.Position) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	filterMembers := true
	symbolInPosition, error := doc.SymbolInPosition(
		symbols.Position{
			Line:      uint(position.Line),
			Character: uint(position.Character - 1),
		})
	if error != nil {
		// Probably, theres no symbol at cursor!
		filterMembers = false
	}
	l.logger.Debug(fmt.Sprintf("building completion list: %s", symbolInPosition.Token))

	// Check if module path is being written/exists
	isCompletingModulePath, possibleModulePath := isCompletingAModulePath(doc, position)

	hasImplicitModulePath := option.None[symbols.ModulePath]()
	if isCompletingModulePath {
		hasImplicitModulePath = extractExplicitModulePath(possibleModulePath)
	}
	isCompletingAChain, prevPosition := isCompletingAChain(doc, position)

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
			l.parsedModulesByDocument[doc.URI],
			prevPosition,
		)

		//	searchParams.scopeMode = AnyPosition

		prevIndexableOption := l.findParentType(searchParams, FindDebugger{depth: 0, enabled: true})
		if prevIndexableOption.IsNone() {
			return items
		}
		prevIndexable := prevIndexableOption.Get()
		//fmt.Print(prevIndexable.GetName())

		switch prevIndexable.(type) {
		case *symbols.Struct:
			strukt := prevIndexable.(*symbols.Struct)

			for _, member := range strukt.GetMembers() {
				if !filterMembers || strings.HasPrefix(member.GetName(), symbolInPosition.Token) {
					items = append(items, protocol.CompletionItem{
						Label: member.GetName(),
						Kind:  &member.Kind,
					})
				}
			}
			// TODO get struct methods
			// Current way of storing struct methods makes this kind of difficult.

		case *symbols.Enum:
			enum := prevIndexable.(*symbols.Enum)
			for _, enumerator := range enum.GetEnumerators() {
				if !filterMembers || strings.HasPrefix(enumerator.GetName(), symbolInPosition.Token) {
					items = append(items, protocol.CompletionItem{
						Label: enumerator.GetName(),
						Kind:  &enumerator.Kind,
					})
				}
			}

		case *symbols.Fault:
			fault := prevIndexable.(*symbols.Fault)
			for _, constant := range fault.GetConstants() {
				if !filterMembers || strings.HasPrefix(constant.GetName(), symbolInPosition.Token) {
					items = append(items, protocol.CompletionItem{
						Label: constant.GetName(),
						Kind:  &constant.Kind,
					})
				}
			}
		}
	} else {
		// Find symbols in document
		params := FindSymbolsParams{
			docId:              doc.URI,
			scopedToModulePath: hasImplicitModulePath,
			position:           option.Some(position),
		}
		// Search symbols loadable in module located in position
		scopeSymbols := l.findSymbolsInScope(params)

		for _, storedIdentifier := range scopeSymbols {
			hasPrefix := strings.HasPrefix(storedIdentifier.GetName(), symbolInPosition.Token)
			if filterMembers && !hasPrefix {
				continue
			}

			tempKind := storedIdentifier.GetKind()

			items = append(items, protocol.CompletionItem{
				Label: storedIdentifier.GetName(),
				Kind:  &tempKind,
			})
		}
	}

	slices.SortFunc(items, func(a, b protocol.CompletionItem) int {
		return cmp.Compare(a.Label, b.Label)
	})

	return items
}

func (l *Language) findParentType(searchParams sp.SearchParams, debugger FindDebugger) option.Option[symbols.Indexable] {
	prevIndexableOption := l.findInParentSymbols(searchParams, debugger)
	if prevIndexableOption.IsNone() {
		return prevIndexableOption
	}

	prevIndexable := prevIndexableOption.Get()

	_, isStructMember := prevIndexable.(*symbols.StructMember)
	if isStructMember {
		var token document.Token
		switch prevIndexable.(type) {
		case *symbols.StructMember:
			structMember, _ := prevIndexable.(*symbols.StructMember)
			token = document.NewToken(structMember.GetType().GetName(), prevIndexable.GetIdRange())
		}
		levelSearchParams := sp.NewSearchParamsBuilder().
			WithSymbol(token.Token).
			WithDocId(prevIndexable.GetDocumentURI()).
			Build()

		prevIndexableOption = l.findClosestSymbolDeclaration(levelSearchParams, debugger.goIn())
	}

	return prevIndexableOption
}
