package server

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/c3"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) hoverModuleDisplayName(symbol symbols.Indexable) string {
	moduleName := symbol.GetModuleString()
	docID := symbol.GetDocumentURI()
	if docID == "" {
		return moduleName
	}

	if moduleName != symbols.NormalizeModuleName(docID) {
		return moduleName
	}

	base := filepath.Base(docID)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	if base == "" {
		base = "anonymous"
	}

	return fmt.Sprintf("%s (anon#%s)", base, anonymousModuleDisambiguator(moduleName))
}

func anonymousModuleDisambiguator(moduleName string) string {
	if len(moduleName) >= 9 {
		sep := moduleName[len(moduleName)-9]
		suffix := moduleName[len(moduleName)-8:]
		if sep == '_' && isHexLower(suffix) {
			return suffix[:4]
		}
	}

	if len(moduleName) >= 4 {
		return moduleName[len(moduleName)-4:]
	}

	return moduleName
}

func (h *Server) syntheticIdentifierHover(docID string, pos symbols.Position) *protocol.Hover {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return nil
	}
	idx := pos.IndexIn(doc.SourceCode.Text)
	ident, ok := extractIdentifierTokenAt(doc.SourceCode.Text, idx)
	if !ok || ident == "" {
		return nil
	}
	if c3.IsLanguageKeyword(ident) {
		return nil
	}
	if r := rune(ident[0]); !unicode.IsUpper(r) {
		return nil
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "```c3\n" + ident + "\n```",
		},
	}

	return &hover
}

func (h *Server) syntheticCollectionLenHover(docID string, pos symbols.Position) *protocol.Hover {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return nil
	}

	unitModules := h.state.GetUnitModulesByDoc(doc.URI)
	if unitModules == nil {
		return nil
	}

	word := doc.SourceCode.SymbolInPosition(pos, unitModules)
	if word.Text() != "len" || !word.HasAccessPath() {
		return nil
	}

	receiver := word.PrevAccessPath()
	receiverDecl := h.search.FindSymbolDeclarationInWorkspace(docID, receiver.TextRange().Start, h.state)
	if receiverDecl.IsNone() {
		return nil
	}

	receiverType := inferTypeFromIndexable(receiverDecl.Get())
	if receiverType == nil || !receiverType.IsCollection() {
		return nil
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "```c3\nusz len\n```",
		},
	}

	return &hover
}

func (h *Server) hoverDocCommentForSymbol(symbol symbols.Indexable) *symbols.DocComment {
	if symbol == nil {
		return nil
	}

	if doc := symbol.GetDocComment(); doc != nil {
		return doc
	}

	faultConstant, ok := symbol.(*symbols.FaultConstant)
	if !ok || h.state == nil {
		return nil
	}

	module := findModuleByName(h.state, faultConstant.GetModuleString())
	if module != nil {
		if doc := faultDocCommentForConstantInModule(module, faultConstant); doc != nil {
			return doc
		}
	}

	var result *symbols.DocComment
	h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() != faultConstant.GetModuleString() {
			return false
		}
		if doc := faultDocCommentForConstantInModule(module, faultConstant); doc != nil {
			result = doc
			return true
		}
		return false
	})

	return result
}

func faultDocCommentForConstantInModule(module *symbols.Module, faultConstant *symbols.FaultConstant) *symbols.DocComment {
	if module == nil || faultConstant == nil {
		return nil
	}

	for _, fault := range module.FaultDefs {
		if fault == nil {
			continue
		}
		for _, constant := range fault.GetConstants() {
			if constant == nil {
				continue
			}
			if constant == faultConstant {
				return fault.GetDocComment()
			}
			if constant.GetName() == faultConstant.GetName() && constant.GetModuleString() == faultConstant.GetModuleString() {
				return fault.GetDocComment()
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

	_, isFault := symbol.(*symbols.FaultDef)
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

func sortedSetKeys(items map[string]struct{}) []string {
	values := make([]string, 0, len(items))
	for key := range items {
		values = append(values, key)
	}
	sort.Strings(values)
	return values
}

func isHexLower(s string) bool {
	if len(s) == 0 {
		return false
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}

	return true
}

func findStructInCandidates(snapshot *project_state.ProjectSnapshot, candidateModules map[string]struct{}, structTypeName string) *symbols.Struct {
	for moduleName := range candidateModules {
		for _, module := range snapshot.ModulesByName(moduleName) {
			if module == nil {
				continue
			}
			if strukt, ok := module.Structs[structTypeName]; ok {
				return strukt
			}
		}
	}

	return nil
}

func findStructByName(snapshot *project_state.ProjectSnapshot, structTypeName string) *symbols.Struct {
	if strings.Contains(structTypeName, "::") {
		modulePath := structTypeName[:strings.LastIndex(structTypeName, "::")]
		typeName := structTypeName[strings.LastIndex(structTypeName, "::")+2:]
		for _, module := range snapshot.ModulesByName(modulePath) {
			if module == nil {
				continue
			}
			if strukt, ok := module.Structs[typeName]; ok {
				return strukt
			}
		}
		return nil
	}

	var found *symbols.Struct
	snapshot.ForEachModule(func(module *symbols.Module) {
		if found != nil {
			return
		}
		if strukt, ok := module.Structs[structTypeName]; ok {
			found = strukt
		}
	})
	return found
}

func resolveCallbackSignature(snapshot *project_state.ProjectSnapshot, memberModule string, memberType string) string {
	if strings.HasPrefix(memberType, "fn ") {
		return memberType
	}

	if memberModule != "" {
		for _, module := range snapshot.ModulesByName(memberModule) {
			if module == nil {
				continue
			}
			if d, ok := module.Aliases[memberType]; ok {
				if d.ResolvesToType() {
					return d.ResolvedType().GetName()
				}
				return d.GetResolvesTo()
			}
		}
	}

	var result string
	snapshot.ForEachModule(func(module *symbols.Module) {
		if result != "" {
			return
		}
		if d, ok := module.Aliases[memberType]; ok {
			if d.ResolvesToType() {
				result = d.ResolvedType().GetName()
			} else {
				result = d.GetResolvesTo()
			}
		}
	})
	return result
}

func parseCallbackParamTypes(signature string) []string {
	start := strings.Index(signature, "(")
	end := strings.LastIndex(signature, ")")
	if start < 0 || end <= start {
		return nil
	}
	paramsText := strings.TrimSpace(signature[start+1 : end])
	if paramsText == "" {
		return []string{}
	}
	parts := strings.Split(paramsText, ",")
	types := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 1 {
			types = append(types, strings.TrimPrefix(fields[0], "&"))
			continue
		}
		typePart := strings.Join(fields[:len(fields)-1], " ")
		types = append(types, strings.TrimPrefix(typePart, "&"))
	}

	return types
}

func findModuleByName(state *project_state.ProjectState, moduleName string) *symbols.Module {
	if state == nil || moduleName == "" {
		return nil
	}

	var result *symbols.Module
	state.ForEachModuleUntil(func(module *symbols.Module) bool {
		if module.GetName() == moduleName {
			result = module
			return true
		}
		return false
	})
	return result
}

func constantValueFromDeclarationSnippet(declaration string) string {
	eqIndex := strings.Index(declaration, "=")
	if eqIndex < 0 {
		return ""
	}

	value := strings.TrimSpace(declaration[eqIndex+1:])
	value = strings.TrimSuffix(value, ";")
	return strings.TrimSpace(value)
}
