package language

import (
	"fmt"
	"strings"

	"github.com/pherrymason/c3-lsp/lsp/document"
	"github.com/pherrymason/c3-lsp/lsp/indexables"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (l *Language) BuildCompletionList(doc *document.Document, position indexables.Position) []protocol.CompletionItem {
	// 2 - TODO if previous character is '.', find previous symbol and if a struct, complete only with struct methods
	// 3 - TODO if writing function call arguments, complete with argument names. ¿Feasible?

	symbolInPosition, _ := doc.SymbolInPosition(
		indexables.Position{
			Line:      uint(position.Line),
			Character: uint(position.Character - 1),
		})
	l.logger.Debug(fmt.Sprintf("building completion list: %s", symbolInPosition.Token))

	var parentSymbol indexables.Indexable
	if symbolInPosition.Token == "." {
		prevPosition := indexables.Position{
			Line:      uint(position.Line),
			Character: uint(position.Character - 2),
		}
		searchParams, _ := NewSearchParamsFromPosition(doc, prevPosition)
		parentSymbol = l.findInParentSymbols(searchParams, NewFindDebugger(false))
	}

	var items []protocol.CompletionItem
	if parentSymbol != nil {
		filterMembers := len(symbolInPosition.Token) > 1
		switch parentSymbol.(type) {
		case indexables.Struct:
			strukt := parentSymbol.(indexables.Struct)

			for _, member := range strukt.GetMembers() {
				if !filterMembers || strings.HasPrefix(member.GetName(), symbolInPosition.Token) {
					items = append(items, protocol.CompletionItem{
						Label: member.GetName(),
						Kind:  &member.Kind,
					})
				}
			}

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
