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
	var items []protocol.CompletionItem
	for _, tag := range l.identifiers {
		items = append(items, protocol.CompletionItem{
			Label: tag,
		})
	}

	return items
}

func debugParser(n string) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Crear el archivo en el directorio de trabajo actual
	filePath := workingDir + "/parsing.txt"
	f, _ := os.Create(filePath)
	defer f.Close()

	d2 := []byte(n)
	f.Write(d2)
}
