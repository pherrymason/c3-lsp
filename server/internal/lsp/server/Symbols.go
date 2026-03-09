package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/symbols"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (h *Server) TextDocumentDocumentSymbol(_ *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	if params == nil {
		return []protocol.DocumentSymbol{}, nil
	}

	h.ensureDocumentIndexed(params.TextDocument.URI)
	_, unitModules := h.getOrLoadDocumentForRename(params.TextDocument.URI)
	if unitModules == nil {
		return []protocol.DocumentSymbol{}, nil
	}

	modules := append([]*symbols.Module(nil), unitModules.Modules()...)
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].GetName() < modules[j].GetName()
	})

	out := make([]protocol.DocumentSymbol, 0, len(modules))
	for _, module := range modules {
		if module == nil {
			continue
		}
		out = append(out, toDocumentSymbol(module, true))
	}

	return out, nil
}

func (h *Server) WorkspaceSymbol(_ *glsp.Context, params *protocol.WorkspaceSymbolParams) ([]protocol.SymbolInformation, error) {
	query := ""
	if params != nil {
		query = strings.ToLower(strings.TrimSpace(params.Query))
	}

	result := make([]protocol.SymbolInformation, 0)
	h.state.ForEachModule(func(module *symbols.Module) {
		collectWorkspaceSymbols(&result, module, "", query)
	})

	sort.Slice(result, func(i, j int) bool {
		leftStarts := strings.HasPrefix(strings.ToLower(result[i].Name), query)
		rightStarts := strings.HasPrefix(strings.ToLower(result[j].Name), query)
		if leftStarts != rightStarts {
			return leftStarts
		}

		if result[i].Location.URI != result[j].Location.URI {
			return result[i].Location.URI < result[j].Location.URI
		}
		if result[i].Location.Range.Start.Line != result[j].Location.Range.Start.Line {
			return result[i].Location.Range.Start.Line < result[j].Location.Range.Start.Line
		}
		if result[i].Location.Range.Start.Character != result[j].Location.Range.Start.Character {
			return result[i].Location.Range.Start.Character < result[j].Location.Range.Start.Character
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (h *Server) WorkspaceSymbolResolve(_ *glsp.Context, params *protocol.SymbolInformation) (*protocol.SymbolInformation, error) {
	if params == nil {
		return nil, nil
	}

	resolved := *params
	return &resolved, nil
}

func collectWorkspaceSymbols(dst *[]protocol.SymbolInformation, item symbols.Indexable, container string, query string) {
	if indexableIsNil(item) {
		return
	}

	name := symbolDisplayName(item)
	if query == "" || strings.Contains(strings.ToLower(name), query) {
		containerName := container
		*dst = append(*dst, protocol.SymbolInformation{
			Name: name,
			Kind: completionKindToSymbolKind(item.GetKind()),
			Location: protocol.Location{
				URI:   protocol.DocumentUri(item.GetDocumentURI()),
				Range: item.GetDocumentRange().ToLSP(),
			},
			ContainerName: &containerName,
		})
	}

	nextContainer := name
	for _, child := range sortedIndexables(outlineChildren(item)) {
		collectWorkspaceSymbols(dst, child, nextContainer, query)
	}
}

func toDocumentSymbol(item symbols.Indexable, includeChildren bool) protocol.DocumentSymbol {
	detail := item.GetCompletionDetail()
	docSymbol := protocol.DocumentSymbol{
		Name:           symbolDisplayName(item),
		Kind:           completionKindToSymbolKind(item.GetKind()),
		Detail:         &detail,
		Range:          item.GetDocumentRange().ToLSP(),
		SelectionRange: item.GetIdRange().ToLSP(),
	}

	if !includeChildren {
		return docSymbol
	}

	children := make([]protocol.DocumentSymbol, 0)
	for _, child := range sortedIndexables(outlineChildren(item)) {
		if _, isFunction := item.(*symbols.Function); isFunction {
			break
		}
		children = append(children, toDocumentSymbol(child, true))
	}
	if len(children) > 0 {
		docSymbol.Children = children
	}

	return docSymbol
}

func sortedIndexables(items []symbols.Indexable) []symbols.Indexable {
	sorted := make([]symbols.Indexable, 0, len(items))
	for _, item := range items {
		if indexableIsNil(item) {
			continue
		}
		sorted = append(sorted, item)
	}

	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i].GetDocumentRange().Start
		right := sorted[j].GetDocumentRange().Start
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Character != right.Character {
			return left.Character < right.Character
		}
		return sorted[i].GetName() < sorted[j].GetName()
	})

	return sorted
}

func outlineChildren(item symbols.Indexable) []symbols.Indexable {
	children := append([]symbols.Indexable{}, item.ChildrenWithoutScopes()...)
	if _, isModule := item.(*symbols.Module); isModule {
		children = append(children, item.NestedScopes()...)
	}

	return dedupeIndexables(children)
}

func dedupeIndexables(items []symbols.Indexable) []symbols.Indexable {
	result := make([]symbols.Indexable, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if indexableIsNil(item) {
			continue
		}

		idRange := item.GetIdRange()
		key := fmt.Sprintf(
			"%s:%s:%d:%d:%d:%d",
			item.GetDocumentURI(),
			item.GetName(),
			idRange.Start.Line,
			idRange.Start.Character,
			idRange.End.Line,
			idRange.End.Character,
		)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}

	return result
}

func completionKindToSymbolKind(kind protocol.CompletionItemKind) protocol.SymbolKind {
	switch kind {
	case protocol.CompletionItemKindModule:
		return protocol.SymbolKindModule
	case protocol.CompletionItemKindFunction:
		return protocol.SymbolKindFunction
	case protocol.CompletionItemKindMethod:
		return protocol.SymbolKindMethod
	case protocol.CompletionItemKindStruct:
		return protocol.SymbolKindStruct
	case protocol.CompletionItemKindInterface:
		return protocol.SymbolKindInterface
	case protocol.CompletionItemKindEnum:
		return protocol.SymbolKindEnum
	case protocol.CompletionItemKindEnumMember:
		return protocol.SymbolKindEnumMember
	case protocol.CompletionItemKindField:
		return protocol.SymbolKindField
	case protocol.CompletionItemKindVariable:
		return protocol.SymbolKindVariable
	case protocol.CompletionItemKindConstant:
		return protocol.SymbolKindConstant
	case protocol.CompletionItemKindTypeParameter:
		return protocol.SymbolKindTypeParameter
	default:
		return protocol.SymbolKindVariable
	}
}

func symbolDisplayName(item symbols.Indexable) string {
	name := item.GetName()
	if name != "" {
		return name
	}

	fault, isFault := item.(*symbols.FaultDef)
	if !isFault {
		return "<anonymous>"
	}

	constants := fault.GetConstants()
	if len(constants) == 0 {
		return "faultdef"
	}

	const maxConstantsInLabel = 3
	labelParts := make([]string, 0, min(len(constants), maxConstantsInLabel)+1)
	for _, c := range constants {
		if c == nil || c.GetName() == "" {
			continue
		}
		if len(labelParts) >= maxConstantsInLabel {
			labelParts = append(labelParts, "...")
			break
		}
		labelParts = append(labelParts, c.GetName())
	}

	if len(labelParts) == 0 {
		return "faultdef"
	}

	return "faultdef " + strings.Join(labelParts, ", ")
}
