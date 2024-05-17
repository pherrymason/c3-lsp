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

	// There are two cases (TBC):
	// User writing with a new symbol: With a ¿space? in front
	// User writing a chained symbol and is expecting to autocomplete with member/methods children of previous token type.
	// So first detect which of these two cases are we in
	isCompletingAChain, prevPosition := isCompletingAChain(doc, position)
	if isCompletingAChain {
		// Is writing a symbol child of a parent one.
		// We need to limit the search to subtypes of parent token
		// Let's find parent token
		searchParams := sp.BuildSearchBySymbolUnderCursor(
			doc,
			l.functionTreeByDocument[doc.URI],
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
		}
	} else {
		// Find symbols in document
		moduleSymbols := l.functionTreeByDocument[doc.URI]
		scopeSymbols := l.findAllScopeSymbols(&moduleSymbols, position)

		for _, storedIdentifier := range scopeSymbols {
			hasPrefix := strings.HasPrefix(storedIdentifier.GetName(), symbolInPosition.Token)
			if !filterMembers || (filterMembers && !hasPrefix) {
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
