package lsp

import (
	"github.com/pherrymason/c3-lsp/pkg/document"
)

func NewDocumentFromString(docId string, documentContent string) document.Document {
	return document.NewDocument(docId, documentContent)
}
