package server

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func canUseReferencesBackedRename(decl symbols.Indexable) bool {
	if indexableIsNil(decl) {
		return false
	}

	_, isFunction := decl.(*symbols.Function)
	if isFunction {
		return true
	}

	variable, isVariable := decl.(*symbols.Variable)
	if isVariable {
		if variable == nil {
			return false
		}
		return !variable.IsConstant()
	}

	_, isEnumerator := decl.(*symbols.Enumerator)
	if isEnumerator {
		return true
	}

	_, isFaultConstant := decl.(*symbols.FaultConstant)
	if isFaultConstant {
		return true
	}

	_, isStructMember := decl.(*symbols.StructMember)
	return isStructMember
}

func validateSymbolRenameNewName(target symbols.Indexable, newName string) (string, error) {
	if normalizedName, ok := normalizeSigilIdentifierRenameInput(target, newName); ok {
		return normalizedName, nil
	}

	normalizedName, ok := normalizeIdentifierRenameInput(newName)
	if ok {
		return normalizedName, nil
	}

	return "", fmt.Errorf("invalid %s name: %s", renameSymbolKindLabel(target), newName)
}

func renameSymbolKindLabel(target symbols.Indexable) string {
	if indexableIsNil(target) {
		return "identifier"
	}

	switch target.(type) {
	case *symbols.Function:
		return "function"
	case *symbols.Variable:
		return "variable"
	case *symbols.StructMember:
		return "struct member"
	case *symbols.Enumerator:
		return "enum member"
	case *symbols.FaultConstant:
		return "fault constant"
	default:
		return "identifier"
	}
}

func normalizeIdentifierRenameInput(newName string) (string, bool) {
	if identifierNamePattern.MatchString(newName) {
		return newName, true
	}

	if !strings.Contains(newName, "::") {
		return "", false
	}

	parts := strings.Split(newName, "::")
	if len(parts) == 0 {
		return "", false
	}

	candidate := parts[len(parts)-1]
	if !identifierNamePattern.MatchString(candidate) {
		return "", false
	}

	return candidate, true
}

func normalizeSigilIdentifierRenameInput(target symbols.Indexable, newName string) (string, bool) {
	variable, ok := target.(*symbols.Variable)
	if !ok || variable == nil {
		return "", false
	}

	oldName := variable.GetName()
	if len(oldName) < 2 {
		return "", false
	}

	prefix := oldName[0]
	if prefix != '$' && prefix != '#' {
		return "", false
	}

	if len(newName) >= 2 && newName[0] == prefix && identifierNamePattern.MatchString(newName[1:]) {
		return newName, true
	}

	if identifierNamePattern.MatchString(newName) {
		return string(prefix) + newName, true
	}

	return "", false
}

func isRenameCandidateName(name string) bool {
	if identifierNamePattern.MatchString(name) {
		return true
	}

	return isSigilIdentifier(name)
}

func isSigilIdentifier(name string) bool {
	if len(name) < 2 {
		return false
	}
	if name[0] != '$' && name[0] != '#' {
		return false
	}

	return identifierNamePattern.MatchString(name[1:])
}

func (h *Server) symbolRenameTargetWithTimeout(docURI string, source string, position protocol.Position, unitModules *symbols_table.UnitModules) (renameTarget, bool) {
	timeout := h.options.Runtime.PrepareRenameTimeout
	if timeout <= 0 {
		return h.symbolRenameTarget(docURI, source, position, unitModules)
	}

	type symbolTargetResult struct {
		target renameTarget
		ok     bool
	}

	resultCh := make(chan symbolTargetResult, 1)
	go func() {
		target, ok := h.symbolRenameTarget(docURI, source, position, unitModules)
		resultCh <- symbolTargetResult{target: target, ok: ok}
	}()

	select {
	case result := <-resultCh:
		return result.target, result.ok
	case <-time.After(timeout):
		if h.server != nil && h.server.Log != nil {
			h.server.Log.Warning("prepareRename timed out resolving symbol target", "timeout", timeout.String(), "uri", docURI, "line", position.Line, "char", position.Character)
		}
		return renameTarget{}, false
	}
}

func (h *Server) symbolRenameTarget(docURI string, source string, position protocol.Position, unitModules *symbols_table.UnitModules) (renameTarget, bool) {
	if unitModules == nil {
		return renameTarget{}, false
	}

	cursorIndex := symbols.NewPositionFromLSPPosition(position).IndexIn(source)
	if tokenInCommentOrString(source, cursorIndex) {
		return renameTarget{}, false
	}

	sourceCode := code.NewSourceCode(source)
	cursorPosition := symbols.NewPositionFromLSPPosition(position)
	cursorCandidates := []symbols.Position{cursorPosition}
	if cursorPosition.Line > 0 {
		cursorCandidates = append(cursorCandidates, symbols.NewPosition(cursorPosition.Line-1, cursorPosition.Character))
	}
	cursorCandidates = append(cursorCandidates, symbols.NewPosition(cursorPosition.Line+1, cursorPosition.Character))

	for _, candidate := range cursorCandidates {
		for _, probe := range renameProbePositions(source, candidate) {
			word := sourceCode.SymbolInPosition(probe, unitModules)
			if word.Text() == "" || word.IsSeparator() {
				continue
			}
			if !isRenameCandidateName(word.Text()) {
				continue
			}

			if memberTarget, found := structMemberRenameTargetAtPosition(unitModules, probe); found {
				return memberTarget, true
			}

			if paramTarget, found := parameterRenameTargetAtPosition(unitModules, probe, word); found {
				return paramTarget, true
			}

			decl := h.search.FindSymbolDeclarationInWorkspace(docURI, probe, h.state)
			if decl.IsNone() {
				memberDecl, found := h.memberDeclarationFromAccessWord(docURI, word, probe, unitModules)
				if found {
					return renameTarget{
						name:         word.Text(),
						renameRange:  word.TextRange().ToLSP(),
						declaration:  memberDecl,
						sourceDocURI: docURI,
					}, true
				}

				fallbackDecl, found := declarationAtPositionByName(unitModules, probe, word.Text())
				if !found {
					continue
				}
				return renameTarget{
					name:         word.Text(),
					renameRange:  word.TextRange().ToLSP(),
					declaration:  fallbackDecl,
					sourceDocURI: docURI,
				}, true
			}

			resolvedDecl := decl.Get()
			if indexableIsNil(resolvedDecl) {
				continue
			}

			if !isRenameNameMatch(resolvedDecl, word.Text()) {
				if word.HasAccessPath() {
					memberDecl, found := h.memberDeclarationFromAccessWord(docURI, word, probe, unitModules)
					if found {
						return renameTarget{
							name:         word.Text(),
							renameRange:  word.TextRange().ToLSP(),
							declaration:  memberDecl,
							sourceDocURI: docURI,
						}, true
					}
				}

				contextDecl, found := declarationByNameInContext(unitModules, word.Text(), probe)
				if found {
					return renameTarget{
						name:         word.Text(),
						renameRange:  word.TextRange().ToLSP(),
						declaration:  contextDecl,
						sourceDocURI: docURI,
					}, true
				}
			}

			if !word.HasAccessPath() && !resolvedDecl.GetIdRange().HasPosition(probe) {
				fallbackDecl, found := declarationAtPositionByName(unitModules, probe, word.Text())
				if found {
					return renameTarget{
						name:         word.Text(),
						renameRange:  word.TextRange().ToLSP(),
						declaration:  fallbackDecl,
						sourceDocURI: docURI,
					}, true
				}
			}

			return renameTarget{
				name:         word.Text(),
				renameRange:  word.TextRange().ToLSP(),
				declaration:  resolvedDecl,
				sourceDocURI: docURI,
			}, true
		}
	}

	return renameTarget{}, false
}

func renameProbePositions(source string, cursor symbols.Position) []symbols.Position {
	probes := []symbols.Position{cursor}
	index := cursor.IndexIn(source)
	if index < 0 {
		return probes
	}

	seen := map[int]struct{}{index: {}}
	for _, preferred := range preferredIdentifierProbeIndices(source, index) {
		if _, ok := seen[preferred]; ok {
			continue
		}

		p := byteIndexToLSPPosition(source, preferred)
		if uint(p.Line) != cursor.Line {
			continue
		}

		seen[preferred] = struct{}{}
		probes = append(probes, symbols.NewPositionFromLSPPosition(p))
	}

	for delta := 1; delta <= 64; delta++ {
		left := index - delta
		if left >= 0 && isIdentifierByte(source[left]) {
			if _, ok := seen[left]; !ok {
				p := byteIndexToLSPPosition(source, left)
				if uint(p.Line) == cursor.Line {
					seen[left] = struct{}{}
					probes = append(probes, symbols.NewPositionFromLSPPosition(p))
				}
			}
		}

		right := index + delta
		if right < len(source) && isIdentifierByte(source[right]) {
			if _, ok := seen[right]; !ok {
				p := byteIndexToLSPPosition(source, right)
				if uint(p.Line) == cursor.Line {
					seen[right] = struct{}{}
					probes = append(probes, symbols.NewPositionFromLSPPosition(p))
				}
			}
		}
	}

	return probes
}

func preferredIdentifierProbeIndices(source string, cursorIndex int) []int {
	if cursorIndex < 0 || cursorIndex >= len(source) {
		return nil
	}

	if isIdentifierByte(source[cursorIndex]) {
		return nil
	}

	indices := make([]int, 0, 2)

	// For delimiters such as comma in argument lists, prefer left identifier first.
	if left := nearestIdentifierIndex(source, cursorIndex, -1); left >= 0 {
		indices = append(indices, left)
	}
	if right := nearestIdentifierIndex(source, cursorIndex, 1); right >= 0 {
		indices = append(indices, right)
	}

	return indices
}

func nearestIdentifierIndex(source string, cursorIndex int, step int) int {
	if step != -1 && step != 1 {
		return -1
	}

	for i := cursorIndex + step; i >= 0 && i < len(source); i += step {
		if source[i] == '\n' {
			return -1
		}

		if isIdentifierByte(source[i]) {
			return i
		}
	}

	return -1
}

func structMemberRenameTargetAtPosition(unitModules *symbols_table.UnitModules, cursor symbols.Position) (renameTarget, bool) {
	if unitModules == nil {
		return renameTarget{}, false
	}

	for _, module := range unitModules.Modules() {
		for _, strukt := range module.Structs {
			if strukt == nil {
				continue
			}
			for _, member := range strukt.GetMembers() {
				if member == nil {
					continue
				}
				if !member.GetDocumentRange().HasPosition(cursor) {
					continue
				}

				return renameTarget{
					name:        member.GetName(),
					renameRange: member.GetIdRange().ToLSP(),
					declaration: member,
				}, true
			}
		}
	}

	return renameTarget{}, false
}

func (h *Server) semanticRenameChanges(target renameTarget, newName string, cache *renameExecutionCache) map[protocol.DocumentUri][]protocol.TextEdit {
	if indexableIsNil(target.declaration) {
		return map[protocol.DocumentUri][]protocol.TextEdit{}
	}

	identity := symbolIdentityFrom(target.declaration)
	changes := map[protocol.DocumentUri][]protocol.TextEdit{}
	paramScope := parameterRenameScope(h.state, target.declaration)
	ownerDocOnly := ""
	if paramScope.IsSome() {
		ownerDocOnly = paramScope.Get().ownerDocURI
	}

	targetStructMember, targetIsStructMember := target.declaration.(*symbols.StructMember)
	if targetIsStructMember && targetStructMember == nil {
		targetIsStructMember = false
	}

	for docID := range h.state.GetAllUnitModules() {
		docURI := string(docID)
		if ownerDocOnly != "" && docURI != ownerDocOnly {
			continue
		}
		doc := h.state.GetDocument(docURI)
		if doc == nil {
			continue
		}
		if !strings.Contains(doc.SourceCode.Text, target.name) {
			continue
		}
		if targetIsStructMember && docURI != target.sourceDocURI && !strings.Contains(doc.SourceCode.Text, "."+target.name) {
			continue
		}

		edits := h.semanticRenameEditsInDocument(docURI, doc.SourceCode.Text, target.name, newName, identity, target.declaration, target.sourceDocURI, target.renameRange, paramScope, cache)
		if len(edits) == 0 {
			continue
		}

		changes[toWorkspaceEditURI(doc.URI, h.options.C3.StdlibPath)] = edits
	}

	return changes
}

func (h *Server) semanticRenameEditsInDocument(docURI string, source string, oldName string, newName string, identity symbolIdentity, targetDecl symbols.Indexable, targetDocURI string, targetRange protocol.Range, paramScope option.Option[parameterScope], cache *renameExecutionCache) []protocol.TextEdit {
	matches := findRenameTokenMatches(source, oldName)
	edits := make([]protocol.TextEdit, 0, len(matches))
	spans := buildCommentStringSpans(source)
	posCursor := newLSPPositionCursor(source)

	for _, match := range matches {
		start := match[0]
		end := match[1]
		if end <= start {
			continue
		}

		if byteIndexInSpans(spans, start) {
			continue
		}

		if tokenInModulePathContext(source, start, end) {
			continue
		}

		startPos := posCursor.PositionAt(start)
		startCursor := symbols.NewPositionFromLSPPosition(startPos)
		endPos := posCursor.PositionAt(end)
		if docURI == targetDocURI && rangeEquals(protocol.Range{Start: startPos, End: endPos}, targetRange) {
			edits = append(edits, protocol.TextEdit{
				Range:   protocol.Range{Start: startPos, End: endPos},
				NewText: newName,
			})
			continue
		}

		if targetMember, ok := targetDecl.(*symbols.StructMember); ok && targetMember != nil {
			isDeclSite := h.structMemberDeclarationMatch(docURI, source, start, targetDecl, cache)
			if !isDeclSite {
				if _, _, hasParentAccess := tokenParentIdentifierBounds(source, start); !hasParentAccess {
					continue
				}

				if h.qualifiedStructMemberAccessMatch(docURI, source, start, targetDecl, cache) {
					edits = append(edits, protocol.TextEdit{
						Range:   protocol.Range{Start: startPos, End: endPos},
						NewText: newName,
					})
					continue
				}
			}
		}

		if !indexableIsNil(targetDecl) && string(targetDecl.GetDocumentURI()) == docURI && targetDecl.GetIdRange().HasPosition(startCursor) {
			edits = append(edits, protocol.TextEdit{
				Range:   protocol.Range{Start: startPos, End: endPos},
				NewText: newName,
			})
			continue
		}

		if paramScope.IsSome() {
			scope := paramScope.Get()
			if scope.ownerDocURI != docURI || !scope.fnRange.HasPosition(startCursor) {
				continue
			}
		}

		decl, found := h.resolveDeclarationAtCandidate(docURI, source, oldName, startPos, cache)
		if !found {
			endProbePos := posCursor.PositionAt(end - 1)
			decl, found = h.resolveDeclarationAtCandidate(docURI, source, oldName, endProbePos, cache)
			if !found {
				if paramScope.IsSome() {
					scope := paramScope.Get()
					if scope.ownerDocURI == docURI && scope.fnRange.HasPosition(startCursor) && scope.name == oldName {
						edits = append(edits, protocol.TextEdit{
							Range:   protocol.Range{Start: startPos, End: endPos},
							NewText: newName,
						})
						continue
					}
				}

				if h.structMemberDeclarationMatch(docURI, source, start, targetDecl, cache) {
					edits = append(edits, protocol.TextEdit{
						Range:   protocol.Range{Start: startPos, End: endPos},
						NewText: newName,
					})
					continue
				}

				if structMemberAccessMatch := h.qualifiedStructMemberAccessMatch(docURI, source, start, targetDecl, cache); structMemberAccessMatch {
					edits = append(edits, protocol.TextEdit{
						Range:   protocol.Range{Start: startPos, End: endPos},
						NewText: newName,
					})
					continue
				}

				if qualifiedModulePathAccessMatch(source, start, targetDecl) {
					edits = append(edits, protocol.TextEdit{
						Range:   protocol.Range{Start: startPos, End: endPos},
						NewText: newName,
					})
					continue
				}

				if !qualifiedOwnerAccessMatch(source, start, targetDecl) {
					continue
				}
				edits = append(edits, protocol.TextEdit{
					Range:   protocol.Range{Start: startPos, End: endPos},
					NewText: newName,
				})
				continue
			}
		}

		if !symbolIdentityMatches(symbolIdentityFrom(decl), identity) {
			if h.structMemberDeclarationMatch(docURI, source, start, targetDecl, cache) {
				edits = append(edits, protocol.TextEdit{
					Range:   protocol.Range{Start: startPos, End: endPos},
					NewText: newName,
				})
				continue
			}

			if !allowParameterScopedRenameFallback(h.state, docURI, startPos, oldName, targetDecl, decl, paramScope) {
				continue
			}
		}

		edits = append(edits, protocol.TextEdit{
			Range:   protocol.Range{Start: startPos, End: endPos},
			NewText: newName,
		})
	}

	return dedupeTextEdits(edits)
}

func qualifiedOwnerAccessMatch(source string, tokenStart int, targetDecl symbols.Indexable) bool {
	if indexableIsNil(targetDecl) {
		return false
	}

	owner := ""
	switch typed := targetDecl.(type) {
	case *symbols.Enumerator:
		if typed == nil {
			return false
		}
		owner = typed.GetEnumName()
	case *symbols.FaultConstant:
		if typed == nil {
			return false
		}
		owner = typed.GetFaultName()
	default:
		return false
	}

	i := tokenStart - 1
	for i >= 0 && (source[i] == ' ' || source[i] == '\t' || source[i] == '\n' || source[i] == '\r') {
		i--
	}
	if i < 0 || source[i] != '.' {
		return false
	}

	i--
	for i >= 0 && (source[i] == ' ' || source[i] == '\t') {
		i--
	}
	if i < 0 {
		return false
	}

	end := i + 1
	for i >= 0 && ((source[i] >= 'a' && source[i] <= 'z') || (source[i] >= 'A' && source[i] <= 'Z') || (source[i] >= '0' && source[i] <= '9') || source[i] == '_') {
		i--
	}
	start := i + 1
	if start >= end {
		return false
	}

	return source[start:end] == owner
}

func qualifiedModulePathAccessMatch(source string, tokenStart int, targetDecl symbols.Indexable) bool {
	if indexableIsNil(targetDecl) {
		return false
	}

	modulePath := targetDecl.GetModuleString()
	if modulePath == "" {
		return false
	}

	i := tokenStart - 1
	for i >= 0 && (source[i] == ' ' || source[i] == '\t' || source[i] == '\n' || source[i] == '\r') {
		i--
	}
	if i < 1 || source[i] != ':' || source[i-1] != ':' {
		return false
	}

	moduleEnd := i - 2
	j := moduleEnd
	for j >= 0 && (isIdentifierByte(source[j]) || source[j] == ':' || source[j] == ' ' || source[j] == '\t') {
		j--
	}
	moduleStart := j + 1
	if moduleStart > moduleEnd {
		return false
	}

	leftExpr := strings.ReplaceAll(source[moduleStart:moduleEnd+1], " ", "")
	leftExpr = strings.ReplaceAll(leftExpr, "\t", "")
	if leftExpr == "" {
		return false
	}

	return leftExpr == modulePath
}

func (h *Server) qualifiedStructMemberAccessMatch(docURI string, source string, tokenStart int, targetDecl symbols.Indexable, cache *renameExecutionCache) bool {
	member, ok := targetDecl.(*symbols.StructMember)
	if !ok {
		return false
	}
	if member == nil {
		return false
	}

	ownerStructName, found := h.structOwnerName(member, cache)
	if !found || ownerStructName == "" {
		return false
	}

	parentStart, parentEnd, ok := tokenParentIdentifierBounds(source, tokenStart)
	if !ok {
		return false
	}

	parentName := source[parentStart:parentEnd]
	parentPos := byteIndexToLSPPosition(source, parentStart)
	parentDecl, resolved := h.resolveDeclarationAtCandidate(docURI, source, parentName, parentPos, cache)
	if !resolved {
		return false
	}

	typeable, ok := parentDecl.(symbols.Typeable)
	if !ok || typeable.GetType() == nil {
		return false
	}

	return typeable.GetType().GetName() == ownerStructName
}

func tokenParentIdentifierBounds(source string, tokenStart int) (int, int, bool) {
	i := tokenStart - 1
	for i >= 0 && (source[i] == ' ' || source[i] == '\t' || source[i] == '\n' || source[i] == '\r') {
		i--
	}
	if i < 0 || source[i] != '.' {
		return 0, 0, false
	}

	i--
	for i >= 0 && (source[i] == ' ' || source[i] == '\t') {
		i--
	}
	if i < 0 {
		return 0, 0, false
	}

	end := i + 1
	for i >= 0 && ((source[i] >= 'a' && source[i] <= 'z') || (source[i] >= 'A' && source[i] <= 'Z') || (source[i] >= '0' && source[i] <= '9') || source[i] == '_') {
		i--
	}
	start := i + 1
	if start >= end {
		return 0, 0, false
	}

	return start, end, true
}

func tokenInModulePathContext(source string, tokenStart int, tokenEnd int) bool {
	right := tokenEnd
	for right < len(source) && (source[right] == ' ' || source[right] == '\t') {
		right++
	}
	if right+1 < len(source) && source[right] == ':' && source[right+1] == ':' {
		return true
	}

	return false
}

func (h *Server) structOwnerName(target *symbols.StructMember, cache *renameExecutionCache) (string, bool) {
	if target == nil {
		return "", false
	}

	memberKey := fmt.Sprintf("%s:%d:%d-%d:%d:%s", target.GetDocumentURI(), target.GetIdRange().Start.Line, target.GetIdRange().Start.Character, target.GetIdRange().End.Line, target.GetIdRange().End.Character, target.GetName())
	if cache != nil {
		cache.structOwnerLookups++
		if cached, ok := cache.structOwnerByMember[memberKey]; ok {
			cache.structOwnerHits++
			return cached.owner, cached.found
		}
	}

	var ownerName string
	found := h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		for _, strukt := range module.Structs {
			for _, member := range strukt.GetMembers() {
				if member.GetDocumentURI() == target.GetDocumentURI() && member.GetIdRange() == target.GetIdRange() {
					ownerName = strukt.GetName()
					return true
				}
			}
		}
		return false
	})
	if found {
		if cache != nil {
			cache.structOwnerByMember[memberKey] = cachedStructOwnerResult{owner: ownerName, found: true}
		}
		return ownerName, true
	}
	if cache != nil {
		cache.structOwnerByMember[memberKey] = cachedStructOwnerResult{found: false}
	}

	return "", false
}

func (h *Server) structMemberDeclarationMatch(docURI string, source string, tokenStart int, targetDecl symbols.Indexable, cache *renameExecutionCache) bool {
	targetMember, ok := targetDecl.(*symbols.StructMember)
	if !ok || targetMember == nil {
		return false
	}

	ownerName, found := h.structOwnerName(targetMember, cache)
	if !found || ownerName == "" {
		return false
	}

	unitModules := h.state.GetUnitModulesByDoc(docURI)
	if unitModules == nil {
		return false
	}

	pos := symbols.NewPositionFromLSPPosition(byteIndexToLSPPosition(source, tokenStart))
	for _, module := range unitModules.Modules() {
		if module == nil {
			continue
		}
		if module.GetName() != targetMember.GetModuleString() {
			continue
		}

		for _, strukt := range module.Structs {
			if strukt == nil || strukt.GetName() != ownerName {
				continue
			}
			for _, member := range strukt.GetMembers() {
				if member == nil {
					continue
				}
				if member.GetName() != targetMember.GetName() {
					continue
				}
				if member.GetIdRange().HasPosition(pos) {
					return true
				}
			}
		}
	}

	return false
}

func (h *Server) resolveDeclarationAtCandidate(docURI string, source string, name string, pos protocol.Position, cache *renameExecutionCache) (symbols.Indexable, bool) {
	if cache != nil {
		cache.declarationLookups++
		cacheKey := fmt.Sprintf("%s:%d:%d:%s", docURI, pos.Line, pos.Character, name)
		if cached, ok := cache.declarationByCandidate[cacheKey]; ok {
			cache.declarationHits++
			return cached.decl, cached.found
		}
		decl, found := h.resolveDeclarationAtCandidateNoCache(docURI, source, name, pos)
		cache.declarationByCandidate[cacheKey] = cachedDeclarationResult{decl: decl, found: found}
		return decl, found
	}

	return h.resolveDeclarationAtCandidateNoCache(docURI, source, name, pos)
}

func (h *Server) resolveDeclarationAtCandidateNoCache(docURI string, source string, name string, pos protocol.Position) (symbols.Indexable, bool) {
	decl := h.search.FindSymbolDeclarationInWorkspace(docURI, symbols.NewPositionFromLSPPosition(pos), h.state)
	if decl.IsSome() {
		resolved := decl.Get()
		if !indexableIsNil(resolved) {
			return resolved, true
		}
	}

	unitModules := h.state.GetUnitModulesByDoc(docURI)
	if unitModules == nil {
		return nil, false
	}

	fallbackDecl, found := declarationAtPositionByName(unitModules, symbols.NewPositionFromLSPPosition(pos), name)
	if !found {
		sourceCode := code.NewSourceCode(source)
		cursor := symbols.NewPositionFromLSPPosition(pos)
		word := sourceCode.SymbolInPosition(cursor, unitModules)
		if word.Text() != name {
			return nil, false
		}

		memberDecl, memberFound := h.memberDeclarationFromAccessWord(docURI, word, cursor, unitModules)
		if !memberFound {
			return nil, false
		}

		return memberDecl, true
	}

	return fallbackDecl, true
}

func (h *Server) memberDeclarationFromAccessWord(docURI string, word code.Word, cursor symbols.Position, unitModules *symbols_table.UnitModules) (symbols.Indexable, bool) {
	if unitModules == nil || !word.HasAccessPath() {
		return nil, false
	}

	parentWord := word.PrevAccessPath()
	parentPos := parentWord.TextRange().Start
	parentDeclOpt := h.search.FindSymbolDeclarationInWorkspace(docURI, parentPos, h.state)
	if parentDeclOpt.IsNone() {
		fallbackParent, found := declarationAtPositionByName(unitModules, parentPos, parentWord.Text())
		if found {
			parentDeclOpt = option.Some(fallbackParent)
		} else {
			contextParent, contextFound := resolveContextSymbolByName(unitModules, parentWord.Text(), cursor)
			if !contextFound {
				return nil, false
			}
			parentDeclOpt = option.Some(contextParent)
		}
	}

	parentDecl := parentDeclOpt.Get()
	if indexableIsNil(parentDecl) {
		return nil, false
	}
	if childDecl, found := accessChildDeclaration(parentDecl, word.Text()); found {
		return childDecl, true
	}

	typeable, ok := parentDecl.(symbols.Typeable)
	if !ok || typeable.GetType() == nil {
		return nil, false
	}

	memberDecl, found := h.findStructMemberDeclaration(word.Text(), typeable.GetType(), parentDecl.GetModuleString(), cursor)
	if !found {
		return nil, false
	}

	return memberDecl, true
}

func accessChildDeclaration(parent symbols.Indexable, childName string) (symbols.Indexable, bool) {
	if indexableIsNil(parent) || childName == "" {
		return nil, false
	}

	switch typed := parent.(type) {
	case *symbols.Struct:
		if typed == nil {
			return nil, false
		}
		for _, member := range typed.GetMembers() {
			if member.GetName() == childName {
				return member, true
			}
		}
	case *symbols.Enum:
		if typed == nil {
			return nil, false
		}
		for _, enumerator := range typed.GetEnumerators() {
			if enumerator.GetName() == childName {
				return enumerator, true
			}
		}
	case *symbols.FaultDef:
		if typed == nil {
			return nil, false
		}
		for _, constant := range typed.GetConstants() {
			if constant.GetName() == childName {
				return constant, true
			}
		}
	}

	return nil, false
}

func resolveContextSymbolByName(unitModules *symbols_table.UnitModules, name string, cursor symbols.Position) (symbols.Indexable, bool) {
	if unitModules == nil || name == "" {
		return nil, false
	}

	contextModuleName := unitModules.FindContextModuleInCursorPosition(cursor)
	if contextModuleName != "" {
		if module := unitModules.Get(contextModuleName); module != nil {
			if symbol, found := findTopLevelSymbolByName(module, name); found {
				return symbol, true
			}
		}
	}

	for _, module := range unitModules.Modules() {
		if symbol, found := findTopLevelSymbolByName(module, name); found {
			return symbol, true
		}
	}

	return nil, false
}

func findTopLevelSymbolByName(module *symbols.Module, name string) (symbols.Indexable, bool) {
	if module == nil || name == "" {
		return nil, false
	}

	if enum := module.Enums[name]; enum != nil {
		return enum, true
	}
	if strukt := module.Structs[name]; strukt != nil {
		return strukt, true
	}
	if def := module.Aliases[name]; def != nil {
		return def, true
	}
	if distinct := module.TypeDefs[name]; distinct != nil {
		return distinct, true
	}
	if interf := module.Interfaces[name]; interf != nil {
		return interf, true
	}
	if variable := module.Variables[name]; variable != nil {
		return variable, true
	}
	for _, fault := range module.FaultDefs {
		if fault.GetName() == name {
			return fault, true
		}
	}
	for _, fun := range module.ChildrenFunctions {
		if fun.GetMethodName() == name || fun.GetName() == name {
			return fun, true
		}
	}

	return nil, false
}

func (h *Server) findStructMemberDeclaration(memberName string, typeInfo *symbols.Type, contextModule string, cursor symbols.Position) (symbols.Indexable, bool) {
	if typeInfo == nil || memberName == "" {
		return nil, false
	}

	typeName := typeInfo.GetName()
	typeModule := typeInfo.GetModule()
	if typeName == "" {
		return nil, false
	}

	if typeModule == "" && contextModule != "" {
		typeModule = contextModule
	}

	candidates := []symbols.Indexable{}
	h.state.ForEachModule(func(module *symbols.Module) {
		if typeModule != "" && module.GetName() != typeModule {
			return
		}

		strukt := module.Structs[typeName]
		if strukt == nil {
			return
		}

		for _, member := range strukt.GetMembers() {
			if member.GetName() == memberName {
				candidates = append(candidates, member)
			}
		}
	})

	if len(candidates) == 0 {
		return nil, false
	}

	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		cand := candidates[i]
		if cand.GetDocumentRange().HasPosition(cursor) {
			return cand, true
		}
		if cand.GetModuleString() == contextModule {
			best = cand
		}
	}

	return best, true
}

func declarationAtPositionByName(unitModules *symbols_table.UnitModules, position symbols.Position, name string) (symbols.Indexable, bool) {
	if unitModules == nil || name == "" {
		return nil, false
	}

	candidates := []symbols.Indexable{}
	for _, module := range unitModules.Modules() {
		for _, sym := range module.Children() {
			collectRenameCandidates(sym, name, position, &candidates)
		}
	}

	if len(candidates) == 0 {
		return declarationByNameInContext(unitModules, name, position)
	}

	best := candidates[0]
	for i := 1; i < len(candidates); i++ {
		candStart := candidates[i].GetIdRange().Start
		bestStart := best.GetIdRange().Start
		if candStart.Line > bestStart.Line || (candStart.Line == bestStart.Line && candStart.Character > bestStart.Character) {
			best = candidates[i]
		}
	}

	return best, true
}

func declarationByNameInContext(unitModules *symbols_table.UnitModules, name string, cursor symbols.Position) (symbols.Indexable, bool) {
	if unitModules == nil || name == "" {
		return nil, false
	}

	contextModuleName := unitModules.FindContextModuleInCursorPosition(cursor)
	if contextModuleName != "" {
		if module := unitModules.Get(contextModuleName); module != nil {
			if symbol, ok := findAnySymbolByName(module, name); ok {
				return symbol, true
			}
		}
	}

	for _, module := range unitModules.Modules() {
		if symbol, ok := findAnySymbolByName(module, name); ok {
			return symbol, true
		}
	}

	return nil, false
}

func findAnySymbolByName(root symbols.Indexable, name string) (symbols.Indexable, bool) {
	if indexableIsNil(root) {
		return nil, false
	}

	if isRenameNameMatch(root, name) {
		return root, true
	}

	for _, child := range root.Children() {
		if found, ok := findAnySymbolByName(child, name); ok {
			return found, true
		}
	}
	for _, nested := range root.NestedScopes() {
		if found, ok := findAnySymbolByName(nested, name); ok {
			return found, true
		}
	}

	return nil, false
}

func collectRenameCandidates(sym symbols.Indexable, name string, pos symbols.Position, out *[]symbols.Indexable) {
	if indexableIsNil(sym) {
		return
	}

	if isRenameNameMatch(sym, name) && sym.GetIdRange().HasPosition(pos) {
		*out = append(*out, sym)
	}

	for _, child := range sym.Children() {
		collectRenameCandidates(child, name, pos, out)
	}
	for _, nested := range sym.NestedScopes() {
		collectRenameCandidates(nested, name, pos, out)
	}
}

func isRenameNameMatch(sym symbols.Indexable, name string) bool {
	if indexableIsNil(sym) {
		return false
	}

	if sym.GetName() == name {
		return true
	}

	if fun, ok := sym.(*symbols.Function); ok {
		if fun == nil {
			return false
		}
		return fun.GetMethodName() == name
	}

	return false
}

func symbolIdentityFrom(decl symbols.Indexable) symbolIdentity {
	if indexableIsNil(decl) {
		return symbolIdentity{}
	}

	return symbolIdentity{
		docURI:  string(decl.GetDocumentURI()),
		kind:    decl.GetKind(),
		idRange: decl.GetIdRange(),
	}
}

func indexableIsNil(sym symbols.Indexable) bool {
	if sym == nil {
		return true
	}
	v := reflect.ValueOf(sym)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func symbolIdentityMatches(left symbolIdentity, right symbolIdentity) bool {
	return left.docURI == right.docURI && left.kind == right.kind && left.idRange == right.idRange
}

func rangeEquals(left protocol.Range, right protocol.Range) bool {
	return left.Start.Line == right.Start.Line &&
		left.Start.Character == right.Start.Character &&
		left.End.Line == right.End.Line &&
		left.End.Character == right.End.Character
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func tokenInCommentOrString(source string, index int) bool {
	if index < 0 || index >= len(source) {
		return false
	}

	const (
		stateCode = iota
		stateLineComment
		stateBlockComment
		stateDoubleQuote
		stateSingleQuote
	)

	state := stateCode
	escaped := false

	for i := 0; i < len(source); i++ {
		if i == index {
			return state != stateCode
		}

		ch := source[i]
		next := byte(0)
		hasNext := i+1 < len(source)
		if hasNext {
			next = source[i+1]
		}

		switch state {
		case stateLineComment:
			if ch == '\n' {
				state = stateCode
			}
			continue
		case stateBlockComment:
			if ch == '*' && hasNext && next == '/' {
				state = stateCode
				i++
			}
			continue
		case stateDoubleQuote:
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				state = stateCode
			}
			continue
		case stateSingleQuote:
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '\'' {
				state = stateCode
			}
			continue
		}

		if ch == '/' && hasNext && next == '/' {
			state = stateLineComment
			i++
			continue
		}
		if ch == '/' && hasNext && next == '*' {
			state = stateBlockComment
			i++
			continue
		}
		if ch == '"' {
			state = stateDoubleQuote
			continue
		}
		if ch == '\'' {
			state = stateSingleQuote
		}
	}

	return false
}
