package lsp

import "C"
import (
	"github.com/pherrymason/c3-lsp/c3"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"os"
)

type Language struct {
	identifiers []string
}

func (l *Language) RefreshDocumentIdentifiers(doc *document) {
	// Reparse document and find identifiers
	l.identifiers = c3.FindIdentifiers(doc.Content)
}

func (l *Language) BuildCompletionList(text string, line protocol.UInteger, character protocol.UInteger) []protocol.CompletionItem {
	/*parser := sitter.NewParser()
	parser.SetLanguage(c3.GetLanguage())

	sourceCode := []byte(text)
	tree := parser.Parse(nil, sourceCode)
	n := tree.RootNode()

	debugParser(fmt.Sprint(n))*/
	var items []protocol.CompletionItem
	for _, tag := range l.identifiers {
		items = append(items, protocol.CompletionItem{
			Label: tag,
			//InsertText: s.buildInsertForTag(tag.Name, prefix, notebook.Config),
			//Detail:     stringPtr(fmt.Sprintf("%d %s", tag.NoteCount, strutil.Pluralize("note", tag.NoteCount))),
		})
	}

	items = append(items, protocol.CompletionItem{
		Label: "talcual jejejeje",
	})

	return items
}

func debugParser(n string) {
	f, _ := os.Create("/Volumes/Development/raul/c3/go-lsp/parsing.txt")
	defer f.Close()

	d2 := []byte(n)
	f.Write(d2)
}
