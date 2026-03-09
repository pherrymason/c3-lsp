package search

import (
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	l "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Search) FindReferencesInWorkspace(
	docId string,
	position symbols.Position,
	state *l.ProjectState,
	includeDeclaration bool,
) []protocol.Location {
	targetOpt := s.FindSymbolDeclarationInWorkspace(docId, position, state)
	if targetOpt.IsNone() {
		return nil
	}

	target := resolveReferenceTarget(docId, position, state, targetOpt.Get())
	if target == nil {
		return nil
	}
	name := target.GetName()
	if name == "" {
		return nil
	}
	_, targetIsVariable := target.(*symbols.Variable)
	targetStructMember, targetIsStructMember := target.(*symbols.StructMember)
	ownerStructName := ""
	if targetIsStructMember && targetStructMember != nil {
		if owner, found := structOwnerNameForMember(state, targetStructMember); found {
			ownerStructName = owner
		}
	}

	targetIdentity := symbolIdentity{
		docURI:  string(target.GetDocumentURI()),
		kind:    target.GetKind(),
		idRange: target.GetIdRange(),
	}

	locations := make([]protocol.Location, 0)
	seen := map[referenceLocationKey]struct{}{}

	for docKey := range state.GetAllUnitModules() {
		doc := state.GetDocument(string(docKey))
		if doc == nil || !strings.Contains(doc.SourceCode.Text, name) {
			continue
		}

		docURI := protocol.DocumentUri(fs.ConvertPathToURI(doc.URI, option.None[string]()))
		spans := buildCommentStringSpans(doc.SourceCode.Text)
		posCursor := newLSPPositionCursor(doc.SourceCode.Text)
		memberAccessCache := map[int]bool{}

		matches := findTokenMatches(doc.SourceCode.Text, name)
		for _, match := range matches {
			if len(match) != 2 || match[1] <= match[0] {
				continue
			}
			if byteIndexInSpans(spans, match[0]) {
				continue
			}

			if targetIsVariable && tokenFollowedByScopeResolution(doc.SourceCode.Text, match[1]) {
				continue
			}
			if targetIsStructMember && tokenFollowedByScopeResolution(doc.SourceCode.Text, match[1]) {
				continue
			}

			start := posCursor.PositionAt(match[0])
			decl := s.FindSymbolDeclarationInWorkspace(string(docKey), symbols.NewPositionFromLSPPosition(start), state)
			if decl.IsSome() {
				resolved := decl.Get()
				if resolved == nil {
					continue
				}

				resolvedIdentity := symbolIdentity{
					docURI:  string(resolved.GetDocumentURI()),
					kind:    resolved.GetKind(),
					idRange: resolved.GetIdRange(),
				}
				if resolvedIdentity != targetIdentity {
					if !structMemberAccessMatchCached(s, string(docKey), doc.SourceCode.Text, match[0], ownerStructName, state, memberAccessCache) &&
						!qualifiedOwnerAccessMatch(doc.SourceCode.Text, match[0], target) &&
						!qualifiedModulePathAccessMatch(doc.SourceCode.Text, match[0], target) {
						continue
					}
				}
			} else {
				if !structMemberAccessMatchCached(s, string(docKey), doc.SourceCode.Text, match[0], ownerStructName, state, memberAccessCache) &&
					!qualifiedOwnerAccessMatch(doc.SourceCode.Text, match[0], target) &&
					!qualifiedModulePathAccessMatch(doc.SourceCode.Text, match[0], target) {
					continue
				}
			}

			rng := protocol.Range{Start: start, End: posCursor.PositionAt(match[1])}
			if !includeDeclaration && string(doc.URI) == targetIdentity.docURI && rangeContainsPosition(target.GetIdRange(), symbols.NewPositionFromLSPPosition(rng.Start)) {
				continue
			}

			key := referenceLocationKey{
				uri:            string(doc.URI),
				startLine:      rng.Start.Line,
				startCharacter: rng.Start.Character,
				endLine:        rng.End.Line,
				endCharacter:   rng.End.Character,
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			locations = append(locations, protocol.Location{URI: docURI, Range: rng})
		}
	}

	if len(locations) == 0 {
		return nil
	}

	return locations
}

func resolveReferenceTarget(docID string, position symbols.Position, state *l.ProjectState, fallback symbols.Indexable) symbols.Indexable {
	if state == nil {
		return fallback
	}

	unitModules := state.GetUnitModulesByDoc(docID)
	if unitModules == nil {
		if doc := state.GetDocument(docID); doc != nil {
			unitModules = state.GetUnitModulesByDoc(doc.URI)
		}
	}
	if unitModules == nil {
		return fallback
	}

	for _, module := range unitModules.Modules() {
		if module == nil {
			continue
		}
		for _, strukt := range module.Structs {
			if strukt == nil {
				continue
			}
			for _, member := range strukt.GetMembers() {
				if member == nil {
					continue
				}
				if member.GetDocumentRange().HasPosition(position) {
					return member
				}
			}
		}
	}

	return fallback
}

type symbolIdentity struct {
	docURI  string
	kind    protocol.CompletionItemKind
	idRange symbols.Range
}

type referenceLocationKey struct {
	uri            string
	startLine      uint32
	startCharacter uint32
	endLine        uint32
	endCharacter   uint32
}

func rangeContainsPosition(r symbols.Range, p symbols.Position) bool {
	return r.HasPosition(p)
}

func tokenFollowedByScopeResolution(source string, tokenEnd int) bool {
	i := tokenEnd
	for i < len(source) && (source[i] == ' ' || source[i] == '\t') {
		i++
	}

	return i+1 < len(source) && source[i] == ':' && source[i+1] == ':'
}

func qualifiedOwnerAccessMatch(source string, tokenStart int, target symbols.Indexable) bool {
	owner := ""
	modulePath := target.GetModuleString()

	switch typed := target.(type) {
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

	ownerStart, ownerEnd, ok := tokenParentIdentifierBounds(source, tokenStart)
	if !ok {
		return false
	}
	if source[ownerStart:ownerEnd] != owner {
		return false
	}

	if moduleExpr, hasModulePath := modulePathBeforeOwner(source, ownerStart); hasModulePath {
		if moduleExpr != modulePath {
			return false
		}
	}

	return true
}

func qualifiedModulePathAccessMatch(source string, tokenStart int, target symbols.Indexable) bool {
	modulePath := target.GetModuleString()
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
	for i >= 0 && isIdentifierByte(source[i]) {
		i--
	}
	start := i + 1
	if start >= end {
		return 0, 0, false
	}

	return start, end, true
}

func modulePathBeforeOwner(source string, ownerStart int) (string, bool) {
	i := ownerStart - 1
	for i >= 0 && (source[i] == ' ' || source[i] == '\t') {
		i--
	}
	if i < 1 || source[i] != ':' || source[i-1] != ':' {
		return "", false
	}

	moduleEnd := i - 2
	j := moduleEnd
	for j >= 0 && (isIdentifierByte(source[j]) || source[j] == ':' || source[j] == ' ' || source[j] == '\t') {
		j--
	}
	moduleStart := j + 1
	if moduleStart > moduleEnd {
		return "", false
	}

	moduleExpr := strings.ReplaceAll(source[moduleStart:moduleEnd+1], " ", "")
	moduleExpr = strings.ReplaceAll(moduleExpr, "\t", "")
	if moduleExpr == "" {
		return "", false
	}

	return moduleExpr, true
}

func isIdentifierByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func structOwnerNameForMember(state *l.ProjectState, target *symbols.StructMember) (string, bool) {
	if state == nil || target == nil {
		return "", false
	}

	var ownerName string
	found := state.ForEachModuleUntil(func(module *symbols.Module) bool {
		for _, strukt := range module.Structs {
			if strukt == nil {
				continue
			}
			for _, member := range strukt.GetMembers() {
				if member == nil {
					continue
				}
				if member.GetDocumentURI() == target.GetDocumentURI() && member.GetIdRange() == target.GetIdRange() {
					ownerName = strukt.GetName()
					return true
				}
			}
		}
		return false
	})
	if found {
		return ownerName, true
	}

	return "", false
}

func structMemberAccessMatch(s *Search, docID string, source string, tokenStart int, ownerStructName string, state *l.ProjectState) bool {
	if ownerStructName == "" {
		return false
	}

	parentStart, _, ok := tokenParentIdentifierBounds(source, tokenStart)
	if !ok {
		return false
	}

	parentPos := byteIndexToLSPPosition(source, parentStart)
	parentDecl := s.FindSymbolDeclarationInWorkspace(docID, symbols.NewPositionFromLSPPosition(parentPos), state)
	if parentDecl.IsNone() {
		return false
	}

	resolved := parentDecl.Get()
	if resolved == nil {
		return false
	}

	typeable, ok := resolved.(symbols.Typeable)
	if !ok || typeable.GetType() == nil {
		return false
	}

	return typeable.GetType().GetName() == ownerStructName
}

func structMemberAccessMatchCached(s *Search, docID string, source string, tokenStart int, ownerStructName string, state *l.ProjectState, cache map[int]bool) bool {
	if cache != nil {
		if v, ok := cache[tokenStart]; ok {
			return v
		}
	}

	result := structMemberAccessMatch(s, docID, source, tokenStart, ownerStructName, state)
	if cache != nil {
		cache[tokenStart] = result
	}

	return result
}

type byteSpan struct {
	start int
	end   int
}

func buildCommentStringSpans(source string) []byteSpan {
	if source == "" {
		return nil
	}

	const (
		stateCode = iota
		stateLineComment
		stateBlockComment
		stateDoubleQuote
		stateSingleQuote
	)

	spans := make([]byteSpan, 0, 64)
	state := stateCode
	stateStart := -1
	escaped := false

	for i := 0; i < len(source); i++ {
		ch := source[i]
		next := byte(0)
		hasNext := i+1 < len(source)
		if hasNext {
			next = source[i+1]
		}

		switch state {
		case stateLineComment:
			if ch == '\n' {
				spans = append(spans, byteSpan{start: stateStart, end: i})
				state = stateCode
				stateStart = -1
			}
			continue
		case stateBlockComment:
			if ch == '*' && hasNext && next == '/' {
				spans = append(spans, byteSpan{start: stateStart, end: i + 2})
				state = stateCode
				stateStart = -1
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
				spans = append(spans, byteSpan{start: stateStart, end: i + 1})
				state = stateCode
				stateStart = -1
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
				spans = append(spans, byteSpan{start: stateStart, end: i + 1})
				state = stateCode
				stateStart = -1
			}
			continue
		}

		if ch == '/' && hasNext && next == '/' {
			state = stateLineComment
			stateStart = i
			i++
			continue
		}
		if ch == '/' && hasNext && next == '*' {
			state = stateBlockComment
			stateStart = i
			i++
			continue
		}
		if ch == '"' {
			state = stateDoubleQuote
			stateStart = i
			continue
		}
		if ch == '\'' {
			state = stateSingleQuote
			stateStart = i
		}
	}

	if state != stateCode && stateStart >= 0 {
		spans = append(spans, byteSpan{start: stateStart, end: len(source)})
	}

	return spans
}

func byteIndexInSpans(spans []byteSpan, index int) bool {
	if len(spans) == 0 || index < 0 {
		return false
	}

	i := sort.Search(len(spans), func(i int) bool {
		return spans[i].end > index
	})
	if i >= len(spans) {
		return false
	}

	span := spans[i]
	return index >= span.start && index < span.end
}

type lspPositionCursor struct {
	content string
	index   int
	line    uint32
	char    uint32
}

func newLSPPositionCursor(content string) *lspPositionCursor {
	return &lspPositionCursor{content: content}
}

func (c *lspPositionCursor) PositionAt(index int) protocol.Position {
	if c == nil {
		return protocol.Position{}
	}

	if index < 0 {
		index = 0
	}
	if index > len(c.content) {
		index = len(c.content)
	}

	if index < c.index {
		c.index = 0
		c.line = 0
		c.char = 0
	}

	for c.index < index {
		r, w := utf8.DecodeRuneInString(c.content[c.index:])
		if r == '\n' {
			c.line++
			c.char = 0
			c.index += w
			continue
		}

		if r == utf8.RuneError && w == 1 {
			c.char++
			c.index += w
			continue
		}

		c.char += uint32(len(utf16.Encode([]rune{r})))
		c.index += w
	}

	return protocol.Position{Line: c.line, Character: c.char}
}

func findTokenMatches(source string, token string) [][]int {
	if source == "" || token == "" {
		return nil
	}

	matches := make([][]int, 0, 32)
	for i := 0; i+len(token) <= len(source); {
		rel := strings.Index(source[i:], token)
		if rel < 0 {
			break
		}

		at := i + rel
		leftOK := at == 0 || !isIdentifierByte(source[at-1])
		right := at + len(token)
		rightOK := right >= len(source) || !isIdentifierByte(source[right])
		if leftOK && rightOK {
			matches = append(matches, []int{at, right})
		}

		i = at + len(token)
	}

	return matches
}

func byteIndexToLSPPosition(content string, index int) protocol.Position {
	return newLSPPositionCursor(content).PositionAt(index)
}
