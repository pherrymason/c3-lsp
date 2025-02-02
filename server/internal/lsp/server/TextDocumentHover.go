package server

import (
	"fmt"
	"github.com/pherrymason/c3-lsp/internal/lsp"
	"github.com/pherrymason/c3-lsp/internal/lsp/analysis"
	"github.com/pherrymason/c3-lsp/pkg/featureflags"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (srv *Server) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	if featureflags.IsActive(featureflags.UseGeneratedAST) {
		doc, _ := srv.documents.GetDocument(params.TextDocument.URI)
		hoverInfo := analysis.GetHoverInfo(
			doc,
			lsp.NewPositionFromProtocol(params.Position),
			srv.documents,
			srv.symbolTable,
		)

		if hoverInfo != nil {
			return hoverInfo, nil
		}

		return nil, nil
	}

	// -----------------------
	// Old implementation
	// -----------------------

	pos := symbols.NewPositionFromLSPPosition(params.Position)
	docId := utils.NormalizePath(params.TextDocument.URI)
	foundSymbolOption := srv.search.FindSymbolDeclarationInWorkspace(docId, pos, srv.state)
	if foundSymbolOption.IsNone() {
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()

	// expected behaviour:
	// hovering on variables: display variable type + any description
	// hovering on functions: display function signature + docs
	// hovering on members: same as variable

	docCommentData := foundSymbol.GetDocComment()
	docComment := ""
	if docCommentData != nil {
		docComment = "\n\n" + docCommentData.DisplayBodyWithContracts()
	}

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
				extraLine +
				docComment,
		},
	}

	return &hover, nil
}

const (
	UNKNOWN = iota
	VAR
	STRUCT
	STRUCT_MEMBER
	BITSTRUCT
	FAULT
	ENUM
)

func typeOfSymbol(symbol symbols.Indexable) uint {
	_, isVariable := symbol.(*symbols.Variable)
	if isVariable {
		return VAR
	}

	_, isMember := symbol.(*symbols.StructMember)
	if isMember {
		return STRUCT_MEMBER
	}

	_, isStruct := symbol.(*symbols.Struct)
	if isStruct {
		return STRUCT
	}

	_, isBitStruct := symbol.(*symbols.Bitstruct)
	if isBitStruct {
		return BITSTRUCT
	}

	_, isFault := symbol.(*symbols.Fault)
	if isFault {
		return FAULT
	}
	_, isEnum := symbol.(*symbols.Enum)
	if isEnum {
		return ENUM
	}

	return UNKNOWN
}

func hasSize(symbol symbols.Indexable) bool {
	kind := typeOfSymbol(symbol)

	sizeableKinds := []uint{VAR, STRUCT, STRUCT_MEMBER, BITSTRUCT, FAULT, ENUM}
	for _, v := range sizeableKinds {
		if v == kind {
			return true
		}
	}

	return false
}

func calculateSize(symbol symbols.Indexable) string {

	switch typeOfSymbol(symbol) {
	case VAR:
		variable := symbol.(*symbols.Variable)
		if variable.Type.IsPointer() {
			return fmt.Sprintf("%d", utils.PointerSize())
		}

		if variable.Type.IsBaseTypeLanguage() {
			return fmt.Sprintf("%d", getLanguageTypeSize(variable.Type.GetName()))
		}

	case STRUCT_MEMBER:
		member := symbol.(*symbols.StructMember)
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
		size = 1
	case "short", "ushort":
		size = 16 / 8
	case "int", "uint":
		size = 32 / 8
	case "long", "ulong":
		size = 64 / 8
	case "int128", "uint128":
		size = 128 / 8
	case "iptr", "uptr":
		size = utils.PointerSize()
	case "isz", "usz":
		size = 0
	}

	return size
}
