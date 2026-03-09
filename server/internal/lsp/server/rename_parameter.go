package server

import (
	"strings"

	"github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	code "github.com/pherrymason/c3-lsp/pkg/document/sourcecode"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) appendParameterDocContractRenameEdits(changes map[protocol.DocumentUri][]protocol.TextEdit, target renameTarget, newName string) map[protocol.DocumentUri][]protocol.TextEdit {
	variable, ok := target.declaration.(*symbols.Variable)
	if !ok || variable == nil {
		return changes
	}

	owner, ok := h.findFunctionOwningArgument(variable)
	if !ok || owner == nil {
		return changes
	}

	if target.name == "" || target.name == newName {
		return changes
	}

	docPath := utils.NormalizePath(owner.GetDocumentURI())
	doc := h.state.GetDocument(docPath)
	if doc == nil {
		return changes
	}

	edits := parameterDocContractRenameEdits(doc.SourceCode.Text, owner, target.name, newName)
	if len(edits) == 0 {
		return changes
	}

	if changes == nil {
		changes = map[protocol.DocumentUri][]protocol.TextEdit{}
	}

	editURI := toWorkspaceEditURI(owner.GetDocumentURI(), h.options.C3.StdlibPath)
	changes[editURI] = append(changes[editURI], edits...)
	return changes
}

func (h *Server) findFunctionOwningArgument(variable *symbols.Variable) (*symbols.Function, bool) {
	if variable == nil {
		return nil, false
	}

	var result *symbols.Function
	h.state.ForEachModuleUntil(func(module *symbols.Module) bool {
		for _, function := range module.ChildrenFunctions {
			if functionOwnsArgument(function, variable) {
				result = function
				return true
			}
		}
		return false
	})
	if result != nil {
		return result, true
	}
	return nil, false
}

func functionOwnsArgument(function *symbols.Function, variable *symbols.Variable) bool {
	if function == nil || variable == nil {
		return false
	}

	for _, argName := range function.ArgumentIds() {
		arg := function.Variables[argName]
		if arg == nil {
			continue
		}

		if arg == variable {
			return true
		}

		if arg.GetName() == variable.GetName() && arg.GetDocumentURI() == variable.GetDocumentURI() && arg.GetIdRange() == variable.GetIdRange() {
			return true
		}
	}

	return false
}

func parameterDocContractRenameEdits(source string, function *symbols.Function, oldName string, newName string) []protocol.TextEdit {
	if source == "" || function == nil || oldName == "" || oldName == newName {
		return nil
	}

	docStart, docEnd, ok := functionDocCommentBounds(source, function)
	if !ok {
		return nil
	}

	block := source[docStart:docEnd]
	lineOffsets := splitLinesWithOffsets(block)
	edits := make([]protocol.TextEdit, 0)

	for _, line := range lineOffsets {
		trimmed := strings.TrimSpace(line.content)
		if strings.HasPrefix(trimmed, "@param") {
			if edit, ok := paramContractNameRenameEdit(source, docStart+line.start, line.content, oldName, newName); ok {
				edits = append(edits, edit)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "@require") {
			edits = append(edits, requireContractRenameEdits(source, docStart+line.start, line.content, oldName, newName)...)
		}
	}

	return edits
}

func functionDocCommentBounds(source string, function *symbols.Function) (int, int, bool) {
	if source == "" || function == nil {
		return 0, 0, false
	}

	functionStart := function.GetDocumentRange().Start.IndexIn(source)
	if functionStart <= 0 || functionStart > len(source) {
		return 0, 0, false
	}

	prefix := source[:functionStart]
	end := strings.LastIndex(prefix, "*>")
	if end < 0 {
		return 0, 0, false
	}

	start := strings.LastIndex(prefix[:end], "<*")
	if start < 0 {
		return 0, 0, false
	}

	between := source[end+2 : functionStart]
	if strings.TrimSpace(between) != "" {
		return 0, 0, false
	}

	return start, end + 2, true
}

type lineWithOffset struct {
	start   int
	content string
}

func splitLinesWithOffsets(text string) []lineWithOffset {
	if text == "" {
		return nil
	}

	lines := make([]lineWithOffset, 0)
	lineStart := 0
	for lineStart <= len(text) {
		relEnd := strings.IndexByte(text[lineStart:], '\n')
		if relEnd < 0 {
			lines = append(lines, lineWithOffset{start: lineStart, content: text[lineStart:]})
			break
		}

		lineEnd := lineStart + relEnd
		lines = append(lines, lineWithOffset{start: lineStart, content: text[lineStart:lineEnd]})
		lineStart = lineEnd + 1
	}

	return lines
}

func paramContractNameRenameEdit(source string, lineAbsStart int, line string, oldName string, newName string) (protocol.TextEdit, bool) {
	keywordIdx := strings.Index(line, "@param")
	if keywordIdx < 0 {
		return protocol.TextEdit{}, false
	}

	nameStart := keywordIdx + len("@param")
	for nameStart < len(line) && (line[nameStart] == ' ' || line[nameStart] == '\t') {
		nameStart++
	}
	if nameStart >= len(line) {
		return protocol.TextEdit{}, false
	}

	if nameStart < len(line) && line[nameStart] == '[' {
		qualifierEnd := strings.IndexByte(line[nameStart:], ']')
		if qualifierEnd < 0 {
			return protocol.TextEdit{}, false
		}
		nameStart += qualifierEnd + 1
		for nameStart < len(line) && (line[nameStart] == ' ' || line[nameStart] == '\t') {
			nameStart++
		}
		if nameStart >= len(line) {
			return protocol.TextEdit{}, false
		}
	}

	nameEnd := nameStart
	if (line[nameEnd] == '$' || line[nameEnd] == '#') && nameEnd+1 < len(line) {
		nameEnd++
	}
	for nameEnd < len(line) && isIdentifierByte(line[nameEnd]) {
		nameEnd++
	}

	if line[nameStart:nameEnd] != oldName {
		return protocol.TextEdit{}, false
	}

	start := lineAbsStart + nameStart
	end := lineAbsStart + nameEnd
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: byteIndexToLSPPosition(source, start),
			End:   byteIndexToLSPPosition(source, end),
		},
		NewText: newName,
	}, true
}

func requireContractRenameEdits(source string, lineAbsStart int, line string, oldName string, newName string) []protocol.TextEdit {
	keywordIdx := strings.Index(line, "@require")
	if keywordIdx < 0 {
		return nil
	}

	exprStart := keywordIdx + len("@require")
	for exprStart < len(line) && (line[exprStart] == ' ' || line[exprStart] == '\t') {
		exprStart++
	}
	if exprStart >= len(line) {
		return nil
	}

	exprEnd := len(line)
	if colon := strings.IndexByte(line[exprStart:], ':'); colon >= 0 {
		exprEnd = exprStart + colon
	}

	if exprEnd <= exprStart {
		return nil
	}

	edits := make([]protocol.TextEdit, 0)
	expr := line[exprStart:exprEnd]
	matches := findRenameTokenMatches(expr, oldName)
	for _, match := range matches {
		start := lineAbsStart + exprStart + match[0]
		end := lineAbsStart + exprStart + match[1]
		edits = append(edits, protocol.TextEdit{
			Range: protocol.Range{
				Start: byteIndexToLSPPosition(source, start),
				End:   byteIndexToLSPPosition(source, end),
			},
			NewText: newName,
		})
	}

	return edits
}

func findRenameTokenMatches(source string, oldName string) [][]int {
	if source == "" || oldName == "" {
		return nil
	}

	matches := make([][]int, 0)
	for i := 0; i+len(oldName) <= len(source); {
		rel := strings.Index(source[i:], oldName)
		if rel < 0 {
			break
		}

		at := i + rel
		leftOK := at == 0 || !isIdentifierByte(source[at-1])
		right := at + len(oldName)
		rightOK := right >= len(source) || !isIdentifierByte(source[right])
		if leftOK && rightOK {
			matches = append(matches, []int{at, right})
		}

		i = at + len(oldName)
	}

	return matches
}

func quickParameterRenameTargetAtDelimiter(source string, position protocol.Position, unitModules *symbols_table.UnitModules) (renameTarget, bool) {
	if unitModules == nil || source == "" {
		return renameTarget{}, false
	}

	cursor := symbols.NewPositionFromLSPPosition(position)
	cursorIndex := cursor.IndexIn(source)
	if cursorIndex < 0 || cursorIndex >= len(source) {
		return renameTarget{}, false
	}

	if isIdentifierByte(source[cursorIndex]) {
		return renameTarget{}, false
	}

	cursorByte := source[cursorIndex]
	if cursorByte != ',' && cursorByte != ')' {
		return renameTarget{}, false
	}

	sourceCode := code.NewSourceCode(source)
	for _, probeIndex := range preferredIdentifierProbeIndices(source, cursorIndex) {
		probeLspPos := byteIndexToLSPPosition(source, probeIndex)
		if uint(probeLspPos.Line) != cursor.Line {
			continue
		}

		probe := symbols.NewPositionFromLSPPosition(probeLspPos)
		word := sourceCode.SymbolInPosition(probe, unitModules)
		if word.Text() == "" || word.IsSeparator() {
			continue
		}
		if !isRenameCandidateName(word.Text()) {
			continue
		}

		if target, found := parameterRenameTargetAtPosition(unitModules, probe, word); found {
			return target, true
		}
	}

	return renameTarget{}, false
}

func parameterRenameTargetAtPosition(unitModules *symbols_table.UnitModules, cursor symbols.Position, cursorWord code.Word) (renameTarget, bool) {
	if unitModules == nil {
		return renameTarget{}, false
	}

	for _, module := range unitModules.Modules() {
		for _, fun := range module.ChildrenFunctions {
			if !fun.GetDocumentRange().HasPosition(cursor) {
				continue
			}

			for _, argName := range fun.ArgumentIds() {
				arg := fun.Variables[argName]
				if arg == nil {
					continue
				}
				if !arg.GetDocumentRange().HasPosition(cursor) {
					continue
				}

				renameRange := arg.GetIdRange().ToLSP()
				if !arg.GetIdRange().HasPosition(cursor) {
					renameRange = cursorWord.TextRange().ToLSP()
				}

				return renameTarget{
					name:         arg.GetName(),
					renameRange:  renameRange,
					declaration:  arg,
					sourceDocURI: arg.GetDocumentURI(),
				}, true
			}
		}
	}

	return renameTarget{}, false
}

type parameterScope struct {
	ownerDocURI string
	fnRange     symbols.Range
	varIDRange  symbols.Range
	name        string
}

func parameterRenameScope(state *project_state.ProjectState, target symbols.Indexable) option.Option[parameterScope] {
	if indexableIsNil(target) {
		return option.None[parameterScope]()
	}

	variable, ok := target.(*symbols.Variable)
	if !ok {
		return option.None[parameterScope]()
	}
	if variable == nil {
		return option.None[parameterScope]()
	}

	var result option.Option[parameterScope]
	state.ForEachModuleUntil(func(module *symbols.Module) bool {
		for _, fun := range module.ChildrenFunctions {
			for _, argName := range fun.ArgumentIds() {
				arg := fun.Variables[argName]
				if arg == nil {
					continue
				}
				if arg.GetDocumentURI() == variable.GetDocumentURI() && arg.GetIdRange() == variable.GetIdRange() {
					result = option.Some(parameterScope{
						ownerDocURI: fun.GetDocumentURI(),
						fnRange:     fun.GetDocumentRange(),
						varIDRange:  variable.GetIdRange(),
						name:        variable.GetName(),
					})
					return true
				}
			}
		}
		return false
	})
	if result.IsSome() {
		return result
	}
	return option.None[parameterScope]()
}

func allowParameterScopedRenameFallback(
	state *project_state.ProjectState,
	docURI string,
	pos protocol.Position,
	oldName string,
	targetDecl symbols.Indexable,
	resolvedDecl symbols.Indexable,
	paramScope option.Option[parameterScope],
) bool {
	if !paramScope.IsSome() {
		return false
	}

	scope := paramScope.Get()
	if scope.ownerDocURI != docURI {
		return false
	}

	cursor := symbols.NewPositionFromLSPPosition(pos)
	if !scope.fnRange.HasPosition(cursor) {
		return false
	}

	if !isRenameNameMatch(targetDecl, oldName) {
		return false
	}

	if resolvedDecl != nil {
		if varDecl, ok := resolvedDecl.(*symbols.Variable); ok {
			if varDecl == nil {
				return false
			}
			if varDecl.GetName() == scope.name && varDecl.GetIdRange() != scope.varIDRange {
				return false
			}
		}
	}

	return true
}
