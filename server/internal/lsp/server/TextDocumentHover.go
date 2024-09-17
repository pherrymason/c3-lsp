package server

import (
	"fmt"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (h *Server) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	pos := symbols.NewPositionFromLSPPosition(params.Position)
	docId := utils.NormalizePath(params.TextDocument.URI)
	foundSymbolOption := h.search.FindSymbolDeclarationInWorkspace(docId, pos, h.state)
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

	sizeInfo := ""
	if utils.IsFeatureEnabled("SIZE_ON_HOVER") {
		if hasSize(foundSymbol) {
			sizeInfo = "// size = " + calculateSize(foundSymbol) + ", align = " + calculateAlignment(foundSymbol) + "\n"
		}
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" +
				sizeInfo +
				foundSymbol.GetHoverInfo() + "\n```" +
				extraLine,
		},
	}

	return &hover, nil
}

func hasSize(symbol symbols.Indexable) bool {
	_, isVariable := symbol.(*symbols.Variable)
	if isVariable {
		return true
	}

	_, isMember := symbol.(*symbols.StructMember)
	if isMember {
		return true
	}

	_, isStruct := symbol.(*symbols.Struct)
	if isStruct {
		return true
	}

	_, isBitStruct := symbol.(*symbols.Bitstruct)
	if isBitStruct {
		return true
	}

	return false
}

func calculateSize(symbol symbols.Indexable) string {
	variable, isVariable := symbol.(*symbols.Variable)
	if isVariable {
		if variable.Type.IsPointer() {
			return fmt.Sprintf("%d", utils.PointerSize())
		}

		if variable.Type.IsBaseTypeLanguage() {
			return fmt.Sprintf("%d", getLanguageTypeSize(variable.Type.GetName()))
		}
	}

	member, isMember := symbol.(*symbols.StructMember)
	if isMember {
		if member.GetType().IsPointer() {
			return fmt.Sprintf("%d", utils.PointerSize())
		}

		if member.GetType().IsBaseTypeLanguage() {
			return fmt.Sprintf("%d", getLanguageTypeSize(member.GetType().GetName()))
		}
	}

	return "?"
}

func calculateAlignment(symbol symbols.Indexable) string {
	return ""
}

func getLanguageTypeSize(typeName string) uint {
	size := uint(0)
	switch typeName {
	case "bool":
		size = 1
	case "ichar", "char":
		size = 8
	case "short", "ushort":
		size = 16
	case "int", "uint":
		size = 32
	case "long", "ulong":
		size = 64
	case "int128", "uint128":
		size = 128
	case "iptr", "uptr", "isz", "usz":
		size = 0
	}

	return size
}
