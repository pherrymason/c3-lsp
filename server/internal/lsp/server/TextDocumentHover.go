package server

import (
	"fmt"
	"strings"
	"unicode"

	ctx "github.com/pherrymason/c3-lsp/internal/lsp/context"
	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Support "Hover"
func (h *Server) TextDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	h.ensureDocumentIndexed(params.TextDocument.URI)

	cursorContext := ctx.BuildFromDocumentPosition(params.Position, params.TextDocument.URI, h.state)
	if cursorContext.IsLiteral {
		return nil, nil
	}

	pos := symbols.NewPositionFromLSPPosition(params.Position)
	docId := utils.NormalizePath(params.TextDocument.URI)
	foundSymbolOption := h.search.FindSymbolDeclarationInWorkspace(docId, pos, h.state)
	if foundSymbolOption.IsNone() {
		return nil, nil
	}

	foundSymbol := foundSymbolOption.Get()
	doc := h.state.GetDocument(docId)

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

	moduleGenericConstraints := ""
	if !isModule {
		if module := findModuleByName(h.state, foundSymbol.GetModuleString()); module != nil {
			constraints := symbols.ModuleGenericConstraintMarkdown(module)
			if constraints != "" {
				moduleGenericConstraints = "\n\n" + constraints
			}
		}
	}

	sizeInfo := ""
	if utils.IsFeatureEnabled("SIZE_ON_HOVER") {
		if hasSize(foundSymbol) {
			sizeInfo = "// size = " + calculateSize(foundSymbol) + ", align = " + calculateAlignment(foundSymbol) + "\n"
		}
	}

	hoverInfo := foundSymbol.GetHoverInfo()
	if doc != nil {
		if genericSuffix, ok := genericTypeSuffixAtPosition(doc.SourceCode.Text, pos); ok {
			hoverInfo = appendGenericSuffixToHoverInfo(foundSymbol, hoverInfo, genericSuffix)
		}
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind: protocol.MarkupKindMarkdown,
			Value: "```c3" + "\n" +
				sizeInfo +
				hoverInfo + "\n```" +
				extraLine +
				docComment +
				moduleGenericConstraints,
		},
	}

	return &hover, nil
}

func findModuleByName(state *project_state.ProjectState, moduleName string) *symbols.Module {
	if state == nil || moduleName == "" {
		return nil
	}

	for _, parsedModules := range state.GetAllUnitModules() {
		for _, module := range parsedModules.Modules() {
			if module.GetName() == moduleName {
				return module
			}
		}
	}

	return nil
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

func appendGenericSuffixToHoverInfo(foundSymbol symbols.Indexable, hoverInfo string, genericSuffix string) string {
	if genericSuffix == "" {
		return hoverInfo
	}

	switch foundSymbol.(type) {
	case *symbols.Struct, *symbols.Interface:
		if !strings.Contains(hoverInfo, genericSuffix) {
			return hoverInfo + genericSuffix
		}
	}

	return hoverInfo
}

func genericTypeSuffixAtPosition(source string, position symbols.Position) (string, bool) {
	index := position.IndexIn(source)
	if index < 0 || index >= len(source) {
		return "", false
	}

	if !isTypeIdentByte(source[index]) {
		if index > 0 && isTypeIdentByte(source[index-1]) {
			index--
		} else {
			return "", false
		}
	}

	end := index
	for end+1 < len(source) && isTypeIdentByte(source[end+1]) {
		end++
	}

	i := end + 1
	for i < len(source) && unicode.IsSpace(rune(source[i])) {
		i++
	}

	if i >= len(source) || source[i] != '{' {
		return "", false
	}

	start := i
	depth := 0
	for ; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : i+1], true
			}
		}
	}

	return "", false
}

func isTypeIdentByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
