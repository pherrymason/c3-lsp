package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
)

func GetDefinitionLocation(document *document.Document, pos lsp.Position) option.Option[Location] {

	return option.None[Location]()
}
