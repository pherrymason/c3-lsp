package handlers

import (
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (h *Handlers) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	pos := symbols.NewPositionFromLSPPosition(params.Position)
	foundSymbolOption := h.search.FindSymbolDeclarationInWorkspace(params.TextDocument.URI, pos, h.state)
	if foundSymbolOption.IsNone() {
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature
	// hovering on members: same as variable

	extraLine := ""

	_, isModule := foundSymbol.(*symbols.Module)
	if !isModule {
		extraLine += "\n\nIn module **[" + foundSymbol.GetModuleString() + "]**"
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" + foundSymbol.GetHoverInfo() + "\n```" +
				extraLine,
		},
	}

	return &hover, nil
}
