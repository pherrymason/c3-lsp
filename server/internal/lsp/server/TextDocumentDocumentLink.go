package server

import (
	"regexp"
	"sort"

	"github.com/pherrymason/c3-lsp/pkg/option"
	"github.com/pherrymason/c3-lsp/pkg/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var (
	modulePathPattern         = `[A-Za-z_][A-Za-z0-9_]*(?:::[A-Za-z_][A-Za-z0-9_]*)*`
	modulePathTokenPattern    = regexp.MustCompile(modulePathPattern)
	importStatementPattern    = regexp.MustCompile(`\bimport\s+([^;\n]*);`)
	moduleStatementPattern    = regexp.MustCompile(`\bmodule\s+([^;\n]*);`)
	qualifiedModuleUsePattern = regexp.MustCompile(`\b(` + modulePathPattern + `)::[A-Za-z_@$#][A-Za-z0-9_@$#]*`)
)

type documentLinkData struct {
	Module string `json:"module"`
	Target string `json:"target"`
}

func (h *Server) TextDocumentDocumentLink(_ *glsp.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	if params == nil {
		return []protocol.DocumentLink{}, nil
	}

	h.ensureDocumentIndexed(params.TextDocument.URI)
	doc, _ := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if doc == nil {
		return []protocol.DocumentLink{}, nil
	}

	moduleTargets := h.moduleNameToTargetURI()
	links := make([]protocol.DocumentLink, 0)
	appendMatches := func(pattern *regexp.Regexp, groupIndex int) {
		for _, match := range pattern.FindAllStringSubmatchIndex(doc.SourceCode.Text, -1) {
			if len(match) <= groupIndex+1 {
				continue
			}

			start := match[groupIndex]
			end := match[groupIndex+1]
			if start < 0 || end <= start {
				continue
			}
			if tokenInCommentOrString(doc.SourceCode.Text, start) {
				continue
			}

			moduleName := doc.SourceCode.Text[start:end]
			target, ok := moduleTargets[moduleName]
			if !ok {
				continue
			}

			targetURI := protocol.DocumentUri(target)
			moduleCopy := moduleName
			data := documentLinkData{Module: moduleName, Target: target}
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: byteIndexToLSPPosition(doc.SourceCode.Text, start),
					End:   byteIndexToLSPPosition(doc.SourceCode.Text, end),
				},
				Target:  &targetURI,
				Tooltip: &moduleCopy,
				Data:    data,
			})
		}
	}

	appendImportStatementMatches := func() {
		for _, match := range importStatementPattern.FindAllStringSubmatchIndex(doc.SourceCode.Text, -1) {
			if len(match) <= 3 {
				continue
			}

			clauseStart := match[2]
			clauseEnd := match[3]
			if clauseStart < 0 || clauseEnd <= clauseStart {
				continue
			}

			clause := doc.SourceCode.Text[clauseStart:clauseEnd]
			for _, token := range modulePathTokenPattern.FindAllStringIndex(clause, -1) {
				start := clauseStart + token[0]
				end := clauseStart + token[1]
				if tokenInCommentOrString(doc.SourceCode.Text, start) {
					continue
				}

				moduleName := doc.SourceCode.Text[start:end]
				target, ok := moduleTargets[moduleName]
				if !ok {
					continue
				}

				targetURI := protocol.DocumentUri(target)
				moduleCopy := moduleName
				data := documentLinkData{Module: moduleName, Target: target}
				links = append(links, protocol.DocumentLink{
					Range: protocol.Range{
						Start: byteIndexToLSPPosition(doc.SourceCode.Text, start),
						End:   byteIndexToLSPPosition(doc.SourceCode.Text, end),
					},
					Target:  &targetURI,
					Tooltip: &moduleCopy,
					Data:    data,
				})
			}
		}
	}

	appendModuleStatementMatches := func() {
		for _, match := range moduleStatementPattern.FindAllStringSubmatchIndex(doc.SourceCode.Text, -1) {
			if len(match) <= 3 {
				continue
			}

			clauseStart := match[2]
			clauseEnd := match[3]
			if clauseStart < 0 || clauseEnd <= clauseStart {
				continue
			}

			clause := doc.SourceCode.Text[clauseStart:clauseEnd]
			token := modulePathTokenPattern.FindStringIndex(clause)
			if token == nil {
				continue
			}

			start := clauseStart + token[0]
			end := clauseStart + token[1]
			if tokenInCommentOrString(doc.SourceCode.Text, start) {
				continue
			}

			moduleName := doc.SourceCode.Text[start:end]
			target, ok := moduleTargets[moduleName]
			if !ok {
				continue
			}

			targetURI := protocol.DocumentUri(target)
			moduleCopy := moduleName
			data := documentLinkData{Module: moduleName, Target: target}
			links = append(links, protocol.DocumentLink{
				Range: protocol.Range{
					Start: byteIndexToLSPPosition(doc.SourceCode.Text, start),
					End:   byteIndexToLSPPosition(doc.SourceCode.Text, end),
				},
				Target:  &targetURI,
				Tooltip: &moduleCopy,
				Data:    data,
			})
		}
	}

	appendImportStatementMatches()
	appendModuleStatementMatches()
	appendMatches(qualifiedModuleUsePattern, 2)

	links = dedupeDocumentLinks(links)
	sort.Slice(links, func(i, j int) bool {
		if links[i].Range.Start.Line != links[j].Range.Start.Line {
			return links[i].Range.Start.Line < links[j].Range.Start.Line
		}
		return links[i].Range.Start.Character < links[j].Range.Start.Character
	})

	return links, nil
}

func (h *Server) DocumentLinkResolve(_ *glsp.Context, params *protocol.DocumentLink) (*protocol.DocumentLink, error) {
	if params == nil {
		return nil, nil
	}
	if params.Target != nil {
		return params, nil
	}

	if data, ok := params.Data.(map[string]any); ok {
		if target, ok := data["target"].(string); ok && target != "" {
			targetURI := protocol.DocumentUri(target)
			params.Target = &targetURI
		}
	}

	if data, ok := params.Data.(documentLinkData); ok && data.Target != "" {
		targetURI := protocol.DocumentUri(data.Target)
		params.Target = &targetURI
	}

	return params, nil
}

func (h *Server) moduleNameToTargetURI() map[string]string {
	moduleToDocs := map[string][]string{}
	for docURI, modules := range h.state.GetAllUnitModules() {
		docID := utils.NormalizePath(string(docURI))
		target := string(toWorkspaceEditURI(docID, option.None[string]()))
		for _, module := range modules.Modules() {
			if module == nil {
				continue
			}
			moduleToDocs[module.GetName()] = append(moduleToDocs[module.GetName()], target)
		}
	}

	out := make(map[string]string, len(moduleToDocs))
	for moduleName, docs := range moduleToDocs {
		sort.Strings(docs)
		if len(docs) > 0 {
			out[moduleName] = docs[0]
		}
	}

	return out
}

func dedupeDocumentLinks(links []protocol.DocumentLink) []protocol.DocumentLink {
	seen := map[[4]uint32]struct{}{}
	out := make([]protocol.DocumentLink, 0, len(links))
	for _, link := range links {
		key := [4]uint32{link.Range.Start.Line, link.Range.Start.Character, link.Range.End.Line, link.Range.End.Character}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, link)
	}

	return out
}
