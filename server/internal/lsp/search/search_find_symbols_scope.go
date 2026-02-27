package search

import (
	"fmt"
	"strings"

	p "github.com/pherrymason/c3-lsp/internal/lsp/project_state"
	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
)

func parseDeclaredModuleName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "module ") {
		return "", false
	}

	withoutKeyword := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
	semicolon := strings.Index(withoutKeyword, ";")
	if semicolon == -1 {
		return "", false
	}

	declaration := strings.TrimSpace(withoutKeyword[:semicolon])
	if declaration == "" {
		return "", false
	}

	for i, r := range declaration {
		if r == ' ' || r == '\t' || r == '@' {
			declaration = declaration[:i]
			break
		}
	}

	if declaration == "" {
		return "", false
	}

	return declaration, true
}

func moduleSectionAnchorLine(source string, moduleName string, line uint) int {
	lines := strings.Split(source, "\n")
	lastAnchor := -1
	for i, sourceLine := range lines {
		if uint(i) > line {
			break
		}

		declaredModule, ok := parseDeclaredModuleName(sourceLine)
		if !ok {
			continue
		}

		if declaredModule == moduleName {
			lastAnchor = i
		}
	}

	return lastAnchor
}

type FindSymbolsParams struct {
	docId              string
	scopedToModulePath option.Option[symbols.ModulePath]
	position           option.Option[symbols.Position]
}

// Returns all symbols in scope.
// Detail: StructMembers and Enumerables are inlined
func (s *Search) findSymbolsInScope(params FindSymbolsParams, state *p.ProjectState) []symbols.Indexable {
	var symbolsCollection []symbols.Indexable

	var currentContextModules []symbols.ModulePath
	var currentModule *symbols.Module
	if params.position.IsSome() {
		// Find current module
		for _, module := range state.GetUnitModulesByDoc(params.docId).Modules() {
			if module.GetDocumentRange().HasPosition(params.position.Get()) {
				// Only include current module in the search if there is no scopedToModule
				if params.scopedToModulePath.IsNone() {
					currentContextModules = append(currentContextModules, module.GetModule())
				}
				currentModule = module
				break
			}
		}
	}

	if params.scopedToModulePath.IsSome() && currentModule != nil {
		// We must take into account that scopedModule path might be a partial path module
		for _, importedModule := range currentModule.Imports {
			if strings.HasSuffix(importedModule, params.scopedToModulePath.Get().GetName()) {
				currentContextModules = append(currentContextModules, symbols.NewModulePathFromString(importedModule))
			}
		}

		currentContextModules = append(currentContextModules, params.scopedToModulePath.Get())
	}

	currentDoc := state.GetDocument(params.docId)
	currentDocSource := ""
	if currentDoc != nil {
		currentDocSource = currentDoc.SourceCode.Text
	}

	cursorSectionAnchorByModule := map[string]int{}
	moduleDocSources := map[string]string{}

	moduleDocSource := func(module *symbols.Module) string {
		docURI := module.GetDocumentURI()
		if source, ok := moduleDocSources[docURI]; ok {
			return source
		}

		doc := state.GetDocument(docURI)
		if doc == nil {
			moduleDocSources[docURI] = ""
			return ""
		}

		moduleDocSources[docURI] = doc.SourceCode.Text
		return doc.SourceCode.Text
	}

	cursorSectionAnchor := func(moduleName string) int {
		if anchor, ok := cursorSectionAnchorByModule[moduleName]; ok {
			return anchor
		}

		if params.position.IsNone() || currentDocSource == "" {
			cursorSectionAnchorByModule[moduleName] = -1
			return -1
		}

		anchor := moduleSectionAnchorLine(currentDocSource, moduleName, params.position.Get().Line)
		cursorSectionAnchorByModule[moduleName] = anchor
		return anchor
	}

	shouldSkipPrivateSymbol := func(module *symbols.Module, symbol symbols.Indexable) bool {
		if params.scopedToModulePath.IsNone() || currentModule == nil {
			return false
		}

		// Imported module completion (e.g. foo::) should not expose private symbols.
		if module.GetName() == currentModule.GetName() {
			return false
		}

		return symbol.IsPrivate()
	}

	shouldSkipLocalSymbol := func(module *symbols.Module, symbol symbols.Indexable) bool {
		if !symbol.IsLocal() {
			return false
		}

		if currentModule == nil || params.position.IsNone() {
			return true
		}

		if module.GetName() != currentModule.GetName() {
			return true
		}

		if module.GetDocumentURI() != params.docId {
			return true
		}

		docSource := moduleDocSource(module)
		if docSource == "" {
			return true
		}

		symbolAnchor := moduleSectionAnchorLine(docSource, module.GetName(), symbol.GetDocumentRange().Start.Line)
		cursorAnchor := cursorSectionAnchor(module.GetName())

		if symbolAnchor == -1 || cursorAnchor == -1 {
			return true
		}

		return symbolAnchor != cursorAnchor
	}

	shouldSkipSymbol := func(module *symbols.Module, symbol symbols.Indexable) bool {
		if shouldSkipPrivateSymbol(module, symbol) {
			return true
		}

		if shouldSkipLocalSymbol(module, symbol) {
			return true
		}

		return false
	}

	// -------------------------------------
	// Modules where we can extract symbols
	// -------------------------------------
	modulesToLook := s.implicitImportedParsedModules(
		state,
		currentContextModules,
		option.None[string](),
	)

	for _, module := range modulesToLook {
		// Only include Module itself, when text is not already prepended with same module name
		isAlreadyPrepended := params.scopedToModulePath.IsNone() ||
			(params.scopedToModulePath.IsSome() && module.GetName() != params.scopedToModulePath.Get().GetName() && !strings.HasSuffix(module.GetName(), params.scopedToModulePath.Get().GetName()))

		if isAlreadyPrepended {
			symbolsCollection = append(symbolsCollection, module)
		}

		for _, variable := range module.Variables {
			if shouldSkipSymbol(module, variable) {
				continue
			}
			symbolsCollection = append(symbolsCollection, variable)
		}
		for _, enum := range module.Enums {
			if shouldSkipSymbol(module, enum) {
				continue
			}
			symbolsCollection = append(symbolsCollection, enum)
			for _, enumerable := range enum.GetEnumerators() {
				if shouldSkipSymbol(module, enumerable) {
					continue
				}
				symbolsCollection = append(symbolsCollection, enumerable)
			}
		}
		for _, strukt := range module.Structs {
			if shouldSkipSymbol(module, strukt) {
				continue
			}
			symbolsCollection = append(symbolsCollection, strukt)
		}
		for _, def := range module.Defs {
			if shouldSkipSymbol(module, def) {
				continue
			}
			symbolsCollection = append(symbolsCollection, def)
		}
		for _, distinct := range module.Distincts {
			if shouldSkipSymbol(module, distinct) {
				continue
			}
			symbolsCollection = append(symbolsCollection, distinct)
		}
		for _, fault := range module.Faults {
			if shouldSkipSymbol(module, fault) {
				continue
			}
			symbolsCollection = append(symbolsCollection, fault)
			for _, constant := range fault.GetConstants() {
				if shouldSkipSymbol(module, constant) {
					continue
				}
				symbolsCollection = append(symbolsCollection, constant)
			}
		}
		for _, interfaces := range module.Interfaces {
			if shouldSkipSymbol(module, interfaces) {
				continue
			}
			symbolsCollection = append(symbolsCollection, interfaces)
		}

		for _, function := range module.ChildrenFunctions {
			if shouldSkipSymbol(module, function) {
				continue
			}
			symbolsCollection = append(symbolsCollection, function)
			if params.position.IsSome() && function.GetDocumentRange().HasPosition(params.position.Get()) {
				for _, variable := range function.Variables {
					s.logger.Debug(fmt.Sprintf("Checking %s variable:", variable.GetName()))
					declarationPosition := variable.GetIdRange().End
					if declarationPosition.Line > uint(params.position.Get().Line) ||
						(declarationPosition.Line == uint(params.position.Get().Line) && declarationPosition.Character > uint(params.position.Get().Character)) {
						continue
					}

					symbolsCollection = append(symbolsCollection, variable)
				}
			}
		}
	}

	return symbolsCollection
}
