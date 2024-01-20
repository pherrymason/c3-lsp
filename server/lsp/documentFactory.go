package lsp

import (
	"github.com/pherrymason/c3-lsp/lsp/document"
)

func NewDocumentFromString(docId string, documentContent string) document.Document {
	return document.NewDocument(docId, "", documentContent)
}
