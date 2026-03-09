package server

import (
	stdctx "context"
	"regexp"
	"strings"
	"unicode"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/internal/lsp/search_params"
	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) hoverReceiverTypeHint(requestCtx stdctx.Context, docID string, searchParams search_params.SearchParams) string {
	fullPath := searchParams.GetFullAccessPath()
	if len(fullPath) < 2 {
		return ""
	}

	receiverPos := fullPath[0].TextRange().Start
	receiver := h.findSymbolDeclarationWithContext(requestCtx, docID, receiverPos)
	if receiver.IsNone() {
		return ""
	}

	switch typed := receiver.Get().(type) {
	case *symbols.Variable:
		return typed.GetType().GetName()
	case *symbols.StructMember:
		return typed.GetType().GetName()
	case *symbols.Function:
		if ret := typed.GetReturnType(); ret != nil {
			return ret.GetName()
		}
	}

	return ""
}

func (h *Server) inferLambdaReceiverTypeFromContext(doc *document.Document, pos symbols.Position, searchParams search_params.SearchParams, snapshot *project_state.ProjectSnapshot, unit symbols_table.UnitModules) string {
	if doc == nil || snapshot == nil {
		return ""
	}
	accessPath := searchParams.GetFullAccessPath()
	if len(accessPath) == 0 {
		return ""
	}
	receiverName := accessPath[0].Text()
	if receiverName == "" {
		return ""
	}

	types := h.inferLambdaParamTypesFromContext(doc, pos, snapshot, unit, searchParams.ModulePathInCursor().GetName())
	if inferred, ok := types[receiverName]; ok {
		return inferred
	}

	return ""
}

func (h *Server) inferLambdaParamTypesFromContext(doc *document.Document, pos symbols.Position, snapshot *project_state.ProjectSnapshot, unit symbols_table.UnitModules, contextModule string) map[string]string {
	lbCtx, ok := h.resolveLambdaCallbackContext(doc, pos, snapshot, unit, contextModule)
	if !ok {
		return nil
	}

	paramTypes := parseCallbackParamTypes(lbCtx.callbackSignature)
	if len(paramTypes) == 0 {
		return nil
	}

	resolved := map[string]string{}
	for i, name := range lbCtx.lambdaParams {
		if i >= len(paramTypes) {
			break
		}
		if name == "" || paramTypes[i] == "" {
			continue
		}
		resolved[name] = paramTypes[i]
	}

	return resolved
}

type lambdaCallbackContext struct {
	memberName        string
	lambdaParams      []string
	structTypeName    string
	memberType        string
	memberModule      string
	callbackSignature string
}

func (h *Server) resolveLambdaCallbackContext(doc *document.Document, pos symbols.Position, snapshot *project_state.ProjectSnapshot, unit symbols_table.UnitModules, contextModule string) (lambdaCallbackContext, bool) {
	if doc == nil || snapshot == nil {
		return lambdaCallbackContext{}, false
	}
	source := doc.SourceCode.Text
	cursorIndex := pos.IndexIn(source)
	if cursorIndex < 0 || cursorIndex > len(source) {
		return lambdaCallbackContext{}, false
	}
	prefixEnd := cursorIndex + 1
	if prefixEnd > len(source) {
		prefixEnd = len(source)
	}
	prefix := source[:prefixEnd]

	lambdaAssignRe := regexp.MustCompile(`\.([A-Za-z_][A-Za-z0-9_]*)\s*=\s*fn`)
	matches := lambdaAssignRe.FindAllStringSubmatchIndex(prefix, -1)
	if len(matches) == 0 {
		return lambdaCallbackContext{}, false
	}
	last := matches[len(matches)-1]
	memberName := prefix[last[2]:last[3]]
	fnTokenEnd := last[1]

	openParenRel := strings.Index(source[fnTokenEnd:], "(")
	if openParenRel < 0 {
		return lambdaCallbackContext{}, false
	}
	openParen := fnTokenEnd + openParenRel
	closeParen, ok := findMatchingParen(source, openParen)
	if !ok {
		return lambdaCallbackContext{}, false
	}
	paramsText := source[openParen+1 : closeParen]
	lambdaParams := parseLambdaParamNames(paramsText)
	if len(lambdaParams) == 0 {
		return lambdaCallbackContext{}, false
	}

	structTypeRe := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*)\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*\{`)
	structMatches := structTypeRe.FindAllStringSubmatchIndex(prefix[:last[0]], -1)
	if len(structMatches) == 0 {
		return lambdaCallbackContext{}, false
	}
	structTypeName := strings.TrimSpace(prefix[structMatches[len(structMatches)-1][2]:structMatches[len(structMatches)-1][3]])
	if structTypeName == "" {
		return lambdaCallbackContext{}, false
	}

	candidateModules := h.collectImportedCandidateModules(snapshot, unit, contextModule)
	strukt := findStructInCandidates(snapshot, candidateModules, structTypeName)
	if strukt == nil {
		strukt = findStructByName(snapshot, structTypeName)
	}
	if strukt == nil {
		return lambdaCallbackContext{}, false
	}

	memberType := ""
	memberModule := ""
	for _, member := range strukt.GetMembers() {
		if member == nil || member.GetName() != memberName {
			continue
		}
		memberType = member.GetType().GetName()
		memberModule = member.GetType().GetModule()
		break
	}
	if memberType == "" {
		return lambdaCallbackContext{}, false
	}

	callbackSignature := resolveCallbackSignature(snapshot, memberModule, memberType)
	if callbackSignature == "" {
		return lambdaCallbackContext{}, false
	}

	return lambdaCallbackContext{
		memberName:        memberName,
		lambdaParams:      lambdaParams,
		structTypeName:    structTypeName,
		memberType:        memberType,
		memberModule:      memberModule,
		callbackSignature: callbackSignature,
	}, true
}

func findMatchingParen(source string, openIdx int) (int, bool) {
	if openIdx < 0 || openIdx >= len(source) || source[openIdx] != '(' {
		return 0, false
	}
	depth := 0
	for i := openIdx; i < len(source); i++ {
		switch source[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}

	return 0, false
}

func (h *Server) enrichInferredLambdaParamTypeFromContext(docID string, pos symbols.Position, foundSymbol symbols.Indexable) {
	variable, ok := foundSymbol.(*symbols.Variable)
	if !ok {
		return
	}
	if variable.GetType().GetName() != "" {
		return
	}

	doc := h.state.GetDocument(docID)
	if doc == nil {
		return
	}
	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	types := h.inferLambdaParamTypesFromContext(doc, pos, snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	inferred, ok := types[variable.GetName()]
	if !ok || inferred == "" {
		return
	}

	variable.Type = symbols.NewTypeFromString(inferred, "")
}

func (h *Server) resolveDesignatedStructMemberHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}
	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)

	source := doc.SourceCode.Text
	cursorIndex := pos.IndexIn(source)
	if cursorIndex <= 0 || cursorIndex > len(source) {
		return option.None[symbols.Indexable]()
	}
	prefix := source[:cursorIndex]

	memberName, designatorStart, ok := extractDesignatorMemberAt(source, cursorIndex)
	if !ok {
		return option.None[symbols.Indexable]()
	}

	structTypeRe := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*)\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*\{`)
	structTypeName := inferStructTypeNameFromInitializerPrefix(prefix[:designatorStart], structTypeRe)
	if structTypeName == "" {
		return option.None[symbols.Indexable]()
	}

	candidateModules := h.collectImportedCandidateModules(snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	strukt := findStructInCandidates(snapshot, candidateModules, structTypeName)
	if strukt == nil {
		strukt = findStructByName(snapshot, structTypeName)
	}
	if strukt == nil {
		return option.None[symbols.Indexable]()
	}

	for _, member := range strukt.GetMembers() {
		if member != nil && member.GetName() == memberName {
			return option.Some[symbols.Indexable](member)
		}
	}

	return option.None[symbols.Indexable]()
}

func inferStructTypeNameFromInitializerPrefix(prefix string, typedInitRe *regexp.Regexp) string {
	if typedInitRe == nil {
		typedInitRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*)\s+[A-Za-z_][A-Za-z0-9_]*\s*=\s*\{`)
	}

	structMatches := typedInitRe.FindAllStringSubmatchIndex(prefix, -1)
	if len(structMatches) > 0 {
		return strings.TrimSpace(prefix[structMatches[len(structMatches)-1][2]:structMatches[len(structMatches)-1][3]])
	}

	assignmentRe := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*=\s*\{`)
	assignmentMatches := assignmentRe.FindAllStringSubmatchIndex(prefix, -1)
	if len(assignmentMatches) == 0 {
		return ""
	}
	lastAssign := assignmentMatches[len(assignmentMatches)-1]
	varName := strings.TrimSpace(prefix[lastAssign[2]:lastAssign[3]])
	if varName == "" {
		return ""
	}

	declRe := regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:]*)\s+` + regexp.QuoteMeta(varName) + `\s*(?:;|=)`)
	declMatches := declRe.FindAllStringSubmatchIndex(prefix[:lastAssign[0]], -1)
	if len(declMatches) == 0 {
		return ""
	}
	lastDecl := declMatches[len(declMatches)-1]
	return strings.TrimSpace(prefix[lastDecl[2]:lastDecl[3]])
}

func (h *Server) resolveLambdaFnSymbolHoverFallback(docID string, pos symbols.Position) option.Option[symbols.Indexable] {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return option.None[symbols.Indexable]()
	}
	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return option.None[symbols.Indexable]()
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return option.None[symbols.Indexable]()
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	if searchParams.Symbol() != "fn" {
		return option.None[symbols.Indexable]()
	}

	lbCtx, ok := h.resolveLambdaCallbackContext(doc, pos, snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	if !ok {
		return option.None[symbols.Indexable]()
	}

	if def := resolveDefByName(snapshot, lbCtx.memberModule, lbCtx.memberType); def != nil {
		return option.Some[symbols.Indexable](def)
	}

	return option.None[symbols.Indexable]()
}

func (h *Server) syntheticLambdaFnHover(docID string, pos symbols.Position) *protocol.Hover {
	doc := h.state.GetDocument(docID)
	if doc == nil {
		return nil
	}
	unit := h.state.GetUnitModulesByDoc(docID)
	if unit == nil {
		return nil
	}
	snapshot := h.state.Snapshot()
	if snapshot == nil {
		return nil
	}

	searchParams := search_params.BuildSearchBySymbolUnderCursor(doc, *unit, pos)
	if searchParams.Symbol() != "fn" {
		return nil
	}

	lbCtx, ok := h.resolveLambdaCallbackContext(doc, pos, snapshot, *unit, searchParams.ModulePathInCursor().GetName())
	if !ok {
		return nil
	}
	if def := resolveDefByName(snapshot, lbCtx.memberModule, lbCtx.memberType); def != nil {
		return nil
	}

	hover := protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "```c3\n" + lbCtx.callbackSignature + "\n```",
		},
	}

	return &hover
}

func resolveDefByName(snapshot *project_state.ProjectSnapshot, preferredModule string, defName string) *symbols.Alias {
	if snapshot == nil || defName == "" {
		return nil
	}
	if preferredModule != "" {
		for _, module := range snapshot.ModulesByName(preferredModule) {
			if module == nil {
				continue
			}
			if d, ok := module.Aliases[defName]; ok {
				return d
			}
		}
	}

	var found *symbols.Alias
	snapshot.ForEachModule(func(module *symbols.Module) {
		if found != nil {
			return
		}
		if d, ok := module.Aliases[defName]; ok {
			found = d
		}
	})
	return found
}

func (h *Server) enrichConstantSymbolFromSource(symbol symbols.Indexable) {
	variable, ok := symbol.(*symbols.Variable)
	if !ok || !variable.IsConstant() || variable.GetConstantValue() != "" {
		return
	}

	doc := h.state.GetDocument(variable.GetDocumentURI())
	if doc == nil {
		return
	}

	rng := variable.GetDocumentRange()
	start := rng.Start.IndexIn(doc.SourceCode.Text)
	end := rng.End.IndexIn(doc.SourceCode.Text)
	if start < 0 || end <= start || end > len(doc.SourceCode.Text) {
		return
	}

	declaration := doc.SourceCode.Text[start:end]
	value := constantValueFromDeclarationSnippet(declaration)
	if value == "" {
		return
	}

	variable.SetConstantValue(value)
}

func (h *Server) enrichUnwrapBindingVariableTypeFromSource(requestCtx stdctx.Context, docID string, symbol symbols.Indexable) {
	variable, ok := symbol.(*symbols.Variable)
	if !ok {
		return
	}

	if strings.TrimSpace(variable.GetType().String()) != "" {
		return
	}

	doc := h.state.GetDocument(docID)
	if doc == nil {
		return
	}

	declRange := variable.GetDocumentRange()
	declStart := declRange.Start.IndexIn(doc.SourceCode.Text)
	declEnd := declRange.End.IndexIn(doc.SourceCode.Text)
	if declStart < 0 || declEnd <= declStart || declEnd > len(doc.SourceCode.Text) {
		return
	}

	declaration := doc.SourceCode.Text[declStart:declEnd]
	if !strings.Contains(declaration, "try "+variable.GetName()+" =") && !strings.Contains(declaration, "catch "+variable.GetName()+" =") {
		return
	}

	rhsEq := strings.Index(declaration, "=")
	if rhsEq < 0 {
		return
	}

	rhs := strings.TrimSpace(declaration[rhsEq+1:])
	lookupExpr := strings.TrimSpace(rhs)
	if openParen := strings.Index(rhs, "("); openParen >= 0 {
		lookupExpr = strings.TrimSpace(rhs[:openParen])
	}
	if lookupExpr == "" {
		return
	}

	lookupInDecl := strings.Index(declaration, lookupExpr)
	if lookupInDecl < 0 {
		return
	}
	lookupAbs := declStart + lookupInDecl + len(lookupExpr) - 1
	if lookupAbs < 0 || lookupAbs >= len(doc.SourceCode.Text) {
		return
	}
	lookupPos := symbols.NewPositionFromLSPPosition(byteIndexToLSPPosition(doc.SourceCode.Text, lookupAbs))
	calleeDecl := h.findSymbolDeclarationWithContext(requestCtx, docID, lookupPos)
	if calleeDecl.IsNone() {
		return
	}

	inferredType := inferTypeFromIndexable(calleeDecl.Get())
	if inferredType == nil {
		return
	}

	inferredTypeText := strings.TrimSpace(inferredType.String())
	if inferredTypeText == "" {
		return
	}
	if strings.HasSuffix(inferredTypeText, "?") {
		inferredTypeText = strings.TrimSpace(strings.TrimSuffix(inferredTypeText, "?"))
	}
	inferredTypeText = qualifyInferredTypeText(inferredType, inferredTypeText)
	if inferredTypeText == "" {
		return
	}

	variable.Type = symbols.NewTypeFromString(inferredTypeText, inferredType.GetModule())
}

func inferTypeFromIndexable(indexable symbols.Indexable) *symbols.Type {
	if typed, ok := indexable.(symbols.Typeable); ok {
		return typed.GetType()
	}

	if function, ok := indexable.(*symbols.Function); ok {
		return function.GetReturnType()
	}

	return nil
}

var returnContractFaultRegex = regexp.MustCompile(`^\?\s*([A-Za-z_][A-Za-z0-9_:.]*)`)
var returnFaultRegex = regexp.MustCompile(`\breturn\s+([A-Za-z_][A-Za-z0-9_:.]*)\s*~`)
var unwrapCallRegex = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_:.]*)\s*\([^\n{};]*\)!`)

func (h *Server) buildFunctionFaultsHoverSection(requestCtx stdctx.Context, docID string, symbol symbols.Indexable) string {
	fn, ok := symbol.(*symbols.Function)
	if !ok {
		return ""
	}

	ret := fn.GetReturnType()
	if ret == nil || !ret.IsOptional() {
		return ""
	}

	docFaults := map[string]struct{}{}
	inferredFaults := map[string]struct{}{}

	if docComment := symbol.GetDocComment(); docComment != nil {
		for _, contract := range docComment.GetContracts() {
			if contract == nil {
				continue
			}
			for _, faultToken := range extractFaultTokensFromReturnContract(contract) {
				docFaults[faultToken] = struct{}{}
			}
		}
	}

	if len(docFaults) == 0 {
		h.collectInferredFunctionFaults(requestCtx, fn, inferredFaults, true)
	}

	if len(docFaults) == 0 && len(inferredFaults) == 0 {
		return ""
	}

	qualifiedDocFaults := h.qualifyFaultSet(fn.GetModuleString(), docFaults)
	qualifiedInferredFaults := h.qualifyFaultSet(fn.GetModuleString(), inferredFaults)

	section := ""
	if len(docFaults) > 0 {
		section += "\n\n@faults: " + strings.Join(sortedSetKeys(qualifiedDocFaults), ", ")
	}
	if len(inferredFaults) > 0 {
		section += "\n\n@faults (inferred): " + strings.Join(sortedSetKeys(qualifiedInferredFaults), ", ")
	}

	return section
}

func extractFaultTokensFromReturnContract(contract *symbols.DocCommentContract) []string {
	if contract == nil {
		return nil
	}

	name := strings.TrimSpace(contract.GetName())
	body := strings.TrimSpace(contract.GetBody())

	if name != "@return" && name != "@return?" {
		return nil
	}

	if strings.HasPrefix(body, "?") {
		body = strings.TrimSpace(strings.TrimPrefix(body, "?"))
	}
	if body == "" {
		return nil
	}

	match := returnContractFaultRegex.FindStringSubmatch("? " + body)
	if len(match) < 2 {
		return nil
	}

	// Only read until optional contract explanation separator.
	if colon := strings.Index(body, ":"); colon >= 0 {
		body = body[:colon]
	}

	parts := strings.Split(body, ",")
	tokens := []string{}
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens
}

func (h *Server) qualifyFaultSet(moduleName string, items map[string]struct{}) map[string]struct{} {
	qualified := map[string]struct{}{}
	for token := range items {
		qualified[h.qualifyFaultToken(moduleName, token)] = struct{}{}
	}
	return qualified
}

func (h *Server) qualifyFaultToken(moduleName string, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return token
	}
	if strings.HasPrefix(token, "@builtin::") {
		return token
	}
	if strings.Contains(token, "::") {
		if strings.HasPrefix(token, "std::core::builtin::") {
			return "@builtin::" + strings.TrimPrefix(token, "std::core::builtin::")
		}
		return token
	}

	if h.state != nil {
		builtinFQN := "std::core::builtin::" + token
		for _, symbol := range h.state.SearchByFQN(builtinFQN) {
			if isFaultLikeSymbol(symbol) {
				return "@builtin::" + token
			}
		}

		if moduleName != "" {
			moduleFQN := moduleName + "::" + token
			for _, symbol := range h.state.SearchByFQN(moduleFQN) {
				if isFaultLikeSymbol(symbol) {
					return moduleFQN
				}
			}
		}

		modules := map[string]struct{}{}
		h.state.ForEachModule(func(module *symbols.Module) {
			for _, fault := range module.FaultDefs {
				if fault == nil {
					continue
				}
				for _, constant := range fault.GetConstants() {
					if constant != nil && constant.GetName() == token {
						modules[constant.GetModuleString()] = struct{}{}
					}
				}
			}
		})

		if len(modules) == 1 {
			for module := range modules {
				if module == "std::core::builtin" {
					return "@builtin::" + token
				}
				return module + "::" + token
			}
		}
	}

	return token
}

func isFaultLikeSymbol(symbol symbols.Indexable) bool {
	if symbol == nil {
		return false
	}
	if _, ok := symbol.(*symbols.FaultConstant); ok {
		return true
	}
	if _, ok := symbol.(*symbols.FaultDef); ok {
		return true
	}
	return false
}

func (h *Server) collectInferredFunctionFaults(requestCtx stdctx.Context, fn *symbols.Function, target map[string]struct{}, includeUnwrapCalls bool) {
	if fn == nil || target == nil {
		return
	}

	functionDoc := h.state.GetDocument(fn.GetDocumentURI())
	if functionDoc == nil {
		return
	}

	rng := fn.GetDocumentRange()
	start := rng.Start.IndexIn(functionDoc.SourceCode.Text)
	end := rng.End.IndexIn(functionDoc.SourceCode.Text)
	if start < 0 || end <= start || end > len(functionDoc.SourceCode.Text) {
		return
	}

	body := functionDoc.SourceCode.Text[start:end]
	functionDocID := utils.NormalizePath(functionDoc.URI)
	for _, match := range returnFaultRegex.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		target[match[1]] = struct{}{}
	}

	if !includeUnwrapCalls {
		return
	}

	for _, match := range unwrapCallRegex.FindAllStringSubmatchIndex(body, -1) {
		if len(match) < 4 {
			continue
		}

		select {
		case <-requestCtx.Done():
			return
		default:
		}

		calleeExpr := body[match[2]:match[3]]
		calleeName := calleeExpr
		if sep := strings.LastIndex(calleeName, "::"); sep >= 0 {
			calleeName = calleeName[sep+2:]
		}
		if sep := strings.LastIndex(calleeName, "."); sep >= 0 {
			calleeName = calleeName[sep+1:]
		}
		calleeName = strings.TrimSpace(calleeName)
		if calleeName == "" {
			continue
		}

		offsetWithinExpr := strings.LastIndex(calleeExpr, calleeName)
		if offsetWithinExpr < 0 {
			offsetWithinExpr = 0
		}
		calleeAbs := start + match[2] + offsetWithinExpr
		calleePos := symbols.NewPositionFromLSPPosition(byteIndexToLSPPosition(functionDoc.SourceCode.Text, calleeAbs+1))
		calleeDecl := h.findSymbolDeclarationWithContext(requestCtx, functionDocID, calleePos)
		if calleeDecl.IsNone() {
			continue
		}

		calleeFn, ok := calleeDecl.Get().(*symbols.Function)
		if !ok {
			continue
		}

		if docComment := calleeFn.GetDocComment(); docComment != nil {
			for _, contract := range docComment.GetContracts() {
				if contract == nil || contract.GetName() != "@return" {
					continue
				}
				contractMatch := returnContractFaultRegex.FindStringSubmatch(strings.TrimSpace(contract.GetBody()))
				if len(contractMatch) >= 2 {
					target[contractMatch[1]] = struct{}{}
				}
			}
		}

		h.collectInferredFunctionFaults(requestCtx, calleeFn, target, false)
	}
}

func qualifyInferredTypeText(inferredType *symbols.Type, rendered string) string {
	if inferredType == nil || rendered == "" || inferredType.IsBaseTypeLanguage() {
		return rendered
	}

	module := strings.TrimSpace(inferredType.GetModule())
	if module == "" || strings.Contains(rendered, "::") {
		return rendered
	}

	start := -1
	end := -1
	for i, r := range rendered {
		if start < 0 {
			if unicode.IsLetter(r) || r == '_' {
				start = i
			}
			continue
		}

		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			end = i
			break
		}
	}

	if start < 0 {
		return rendered
	}
	if end < 0 {
		end = len(rendered)
	}

	return rendered[:start] + module + "::" + rendered[start:end] + rendered[end:]
}
