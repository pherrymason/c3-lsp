package lsp

import "C"
import (
	"errors"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"os"
)

type Identifier struct {
	name                string
	kind                protocol.CompletionItemKind
	declarationPosition protocol.Position
}

type Language struct {
	identifiers []Identifier
}

func (l *Language) RefreshDocumentIdentifiers(doc *document) {
	// Reparse document and find identifiers
	l.identifiers = FindIdentifiers(doc.Content, false)
}

func (l *Language) BuildCompletionList(text string, line protocol.UInteger, character protocol.UInteger) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, tag := range l.identifiers {
		items = append(items, protocol.CompletionItem{
			Label: tag.name,
			Kind:  &tag.kind,
		})
	}

	return items
}

func (l *Language) FindIdentifierDeclaration(identifier string) (Identifier, error) {
	for i := 0; i < len(l.identifiers); i++ {
		if l.identifiers[i].name == identifier {
			return l.identifiers[i], nil
		}
	}

	return Identifier{}, errors.New("no se encontró el string en el array")
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
