package language

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	"github.com/pherrymason/c3-lsp/lsp/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func buildSearchParamsForCompletionList(doc *document.Document, position indexables.Position) SearchParams {
	// There are two cases (TBC):
	// User writing with a new symbol: With a ¿space? in front
	// User writing a chained symbol and is expecting to autocomplete with member/methods children of previous token type.
	// So first detect which of these two cases are we in

	// Read backwards until first ¿space?

	searchParams := SearchParams{}

	return searchParams
}

// Checks if writing seems to be a chain of components (example: aStruct.aMember)
// If that's the case, it will return the position of the last character of previous token
func isCompletingAChain(doc *document.Document, cursorPosition indexables.Position) (bool, indexables.Position) {
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
	var previousPosition indexables.Position

	containsSeparator := strings.Contains(sentence, ".") || strings.Contains(sentence, ":")
	if containsSeparator {
		lastIndex := len(sentence) - 1
		if strings.Contains(sentence, ".") {
			lastIndex = strings.LastIndex(sentence, ".")
		} else if strings.Contains(sentence, "::") {
			lastIndex = strings.LastIndex(sentence, "::")
		}
		sub := len(sentence) - lastIndex
		previousPosition = indexables.NewPosition(
			position.Line,
			cursorPosition.Character-uint(sub)-1, // one extra -1 to stay just behind last character
		)
	}

	return containsSeparator, previousPosition
}

func (l *Language) BuildCompletionList(doc *document.Document, position indexables.Position) []protocol.CompletionItem {

	var items []protocol.CompletionItem
	filterMembers := true
	symbolInPosition, error := doc.SymbolInPosition(
		indexables.Position{
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
		searchParams, _ := NewSearchParamsFromPosition(doc, prevPosition)
		prevIndexable := l.findInParentSymbols(searchParams, FindDebugger{depth: 0})
		fmt.Print(prevIndexable.GetName())
		//filterMembers := len(symbolInPosition.Token) > 1

		switch prevIndexable.(type) {
		case indexables.Struct:
			strukt := prevIndexable.(indexables.Struct)

			for _, member := range strukt.GetMembers() {
				if !filterMembers || strings.HasPrefix(member.GetName(), symbolInPosition.Token) {
					items = append(items, protocol.CompletionItem{
						Label: member.GetName(),
						Kind:  &member.Kind,
					})
				}
			}
			// TODO get struct methods
		}
	} else {
		// Find symbols in document
		moduleSymbols := l.functionTreeByDocument[doc.URI]
		scopeSymbols := l.findAllScopeSymbols(&moduleSymbols, position)

		for _, storedIdentifier := range scopeSymbols {
			if !strings.HasPrefix(storedIdentifier.GetName(), symbolInPosition.Token) {
				continue
			}

			tempKind := storedIdentifier.GetKind()

			items = append(items, protocol.CompletionItem{
				Label: storedIdentifier.GetName(),
				Kind:  &tempKind,
			})
		}
	}

	return items
}
